package clients

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// binanceSymbolCacheEntry holds the two API objects a symbol is built from:
// the exchangeInfo entry and the leverage bracket. Both are cached together so
// SymbolInfo can return them without re-fetching.
type binanceSymbolCacheEntry struct {
	info    *BinanceSymbolInfo
	bracket *BinanceLeverageBracket
}

// Binance represents the Binance API client
type Binance struct {
	client           *http.Client
	websocketClients map[string]*BinanceWebsocket // websocket clients for each account
	symbolInfoCache  map[string]*binanceSymbolCacheEntry
	cacheMu          sync.RWMutex
	semaphore        chan struct{} // Buffered channel for concurrency control
	rateLimiter      *RateLimiter
	cfg              subsettings.Binance
	creds            *secrets.SecretCache
	logApiResponses  bool
	logger           *slog.Logger
}

// NewBinance creates a new Binance client
func NewBinance(cfg subsettings.Binance, logApiResponses bool) *Binance {
	client := &http.Client{
		Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HttpClient.PoolSize,
			MaxIdleConnsPerHost: cfg.HttpClient.PoolSize,
		},
	}

	creds := secrets.NewSecretCache()

	// create websocket clients for each account
	websocketClients := make(map[string]*BinanceWebsocket)
	for accountKey, account := range cfg.Accounts {
		websocketClients[accountKey] = NewBinanceWebsocket(
			account,
			cfg.Websockets.PingInterval,
			cfg.Websockets.ReconnectDelay,
			cfg.Websockets.Timeout,
			creds,
		)
	}

	// Convert rate limiter limits
	limits := make([]RateLimit, len(cfg.HttpClient.RateLimiter.Limits))
	for i, limit := range cfg.HttpClient.RateLimiter.Limits {
		limits[i] = RateLimit{
			Requests: limit.Requests,
			Seconds:  limit.Seconds,
		}
	}

	return &Binance{
		client:           client,
		websocketClients: websocketClients,
		symbolInfoCache:  make(map[string]*binanceSymbolCacheEntry),
		semaphore:        make(chan struct{}, cfg.HttpClient.MaxConcurrent),
		rateLimiter:      NewRateLimiter(limits),
		cfg:              cfg,
		creds:            creds,
		logApiResponses:  logApiResponses,
		logger:           logging.NewLogger("binance-client"),
	}
}

// ConnectWebsocket connects to the websocket client for an account
func (b *Binance) ConnectWebsocket(accountKey string) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BinanceError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.Connect()
	if err != nil {
		return &internal.BinanceError{Message: "failed to connect to websocket: " + err.Error()}
	}
	return nil
}

// CloseWebsocket closes the websocket client for an account
func (b *Binance) CloseWebsocket(accountKey string) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BinanceError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.Close()
	if err != nil {
		return &internal.BinanceError{Message: "failed to close websocket: " + err.Error()}
	}
	return nil
}

// ConnectAllWebsockets connects all websocket clients and starts supervisors.
func (b *Binance) ConnectAllWebsockets(ctx context.Context) {
	for accountKey, websocket := range b.websocketClients {
		b.logger.Info("Connecting websocket", "account", accountKey)
		websocket.ConnectWithSupervisor(ctx)
	}
}

// CloseAllWebsockets closes all websocket clients.
func (b *Binance) CloseAllWebsockets() {
	for accountKey, websocket := range b.websocketClients {
		if err := websocket.Close(); err != nil {
			b.logger.Warn("Failed to close websocket", "account", accountKey, "error", err)
		}
	}
}

// Positions retrieves all positions for an account
func (b *Binance) Positions(account subsettings.BinanceAccount) ([]BinancePosition, error) {
	results, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v2/positionRisk",
		&[]BinancePosition{},
		nil,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	positions, ok := results.(*[]BinancePosition)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	// Filter positions with non-zero position amount
	filtered := make([]BinancePosition, 0)
	for _, pos := range *positions {
		if math.Abs(pos.PositionAmt.Float64()) > 0 {
			filtered = append(filtered, pos)
		}
	}

	return filtered, nil
}

// Position retrieves a single position for a symbol
func (b *Binance) Position(account subsettings.BinanceAccount, symbol string) (*BinancePositionGenericInfo, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	results, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v2/positionRisk",
		&[]BinancePositionGenericInfo{},
		params,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	positions, ok := results.(*[]BinancePositionGenericInfo)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	// Filter positions with non-zero position amount and if the symbol is the one we are looking for
	filtered := make([]*BinancePositionGenericInfo, 0)
	for i := range *positions {
		if math.Abs((*positions)[i].PositionAmt.Float64()) > 0 && (*positions)[i].Symbol == symbol {
			filtered = append(filtered, &(*positions)[i])
		}
	}

	if len(filtered) == 1 {
		return filtered[0], nil
	}
	// if no positions are found, return nil
	if len(filtered) == 0 {
		return nil, nil
	}

	return nil, &internal.BinanceError{
		Message: fmt.Sprintf("Expected 0 or 1 position, got %d", len(filtered)),
	}
}

// OpenOrders retrieves all open orders for a symbol
func (b *Binance) OpenOrders(account subsettings.BinanceAccount, symbol *string) ([]BinanceOpenOrder, error) {
	params := map[string]string{}
	if symbol != nil {
		params["symbol"] = *symbol
	}

	results, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/openAlgoOrders",
		&[]BinanceOpenOrder{},
		params,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	orders, ok := results.(*[]BinanceOpenOrder)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return *orders, nil
}

// AccountInfo retrieves account information
// Note: Different use than Bybit's AccountInfo
func (b *Binance) AccountInfo(account subsettings.BinanceAccount) (*BinanceAccountInfo, error) {
	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v3/account",
		&BinanceAccountInfo{},
		nil,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	info, ok := result.(*BinanceAccountInfo)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return info, nil
}

// AccountPositionMode returns the position mode for an account
func (b *Binance) AccountPositionMode(account subsettings.BinanceAccount) (string, error) {
	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/positionSide/dual",
		&BinancePositionModeResponse{},
		nil,
		nil,
		true,
	)
	if err != nil {
		return "", err
	}

	response, ok := result.(*BinancePositionModeResponse)
	if !ok {
		return "", &internal.BinanceError{Message: "type assertion failed"}
	}

	if response.DualSidePosition {
		return "hedge", nil
	}
	return "one-way", nil
}

// SetAccountPositionMode sets the position mode for an account
// positionMode can be "hedge" or "one-way"
func (b *Binance) SetAccountPositionMode(account subsettings.BinanceAccount, positionMode string) error {
	if positionMode != "hedge" && positionMode != "one-way" {
		return fmt.Errorf("invalid position mode: %s", positionMode)
	}

	dualSidePosition := "false"
	if positionMode == "hedge" {
		dualSidePosition = "true"
	}

	params := map[string]string{
		"dualSidePosition": dualSidePosition,
	}

	_, err := b.Request(
		account,
		"POST",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/positionSide/dual",
		&BinanceEmptyResponse{},
		params,
		nil,
		true,
	)
	if err != nil {
		// Check if error is -4059 (already set)
		if binanceErr, ok := err.(*internal.BinanceBadStatusCodeError); ok {
			if binanceErr.ErrorCode != nil && *binanceErr.ErrorCode == -4059 {
				b.logger.Warn("Account position mode is already set", "position_mode", positionMode)
				return nil
			}
		}
		return err
	}

	return nil
}

// SettleCoinBalance retrieves the settle coin balance
func (b *Binance) SettleCoinBalance(account subsettings.BinanceAccount) (*BinanceAccountInfoAssets, error) {
	accountInfo, err := b.AccountInfo(account)
	if err != nil {
		return nil, err
	}

	var assets []BinanceAccountInfoAssets
	for _, asset := range accountInfo.Assets {
		if asset.Asset == account.SettleCoin {
			assets = append(assets, asset)
		}
	}

	if len(assets) == 1 {
		return &assets[0], nil
	}

	return nil, &internal.BinanceError{
		Message: fmt.Sprintf("Expected 1 asset, got %d", len(assets)),
	}
}

// Ticker retrieves ticker information
func (b *Binance) Ticker(account subsettings.BinanceAccount, symbol string) (*BinanceTickerPrice, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v2/ticker/price",
		&BinanceTickerPrice{},
		params,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}

	ticker, ok := result.(*BinanceTickerPrice)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return ticker, nil
}

// SymbolInfo retrieves symbol information (with optional caching).
func (b *Binance) SymbolInfo(account subsettings.BinanceAccount, symbol string, cached bool) (*BinanceSymbolInfo, *BinanceLeverageBracket, error) {
	if cached {
		b.cacheMu.RLock()
		if entry, ok := b.symbolInfoCache[symbol]; ok {
			info, bracket := entry.info, entry.bracket
			b.cacheMu.RUnlock()
			if bracket == nil {
				return nil, nil, &internal.BinanceError{
					Message: fmt.Sprintf("No leverage bracket found for %s", symbol),
				}
			}
			return info, bracket, nil
		}
		b.cacheMu.RUnlock()
	}

	symbols, err := b.fetchAllExchangeInfo(account)
	if err != nil {
		return nil, nil, err
	}

	brackets, err := b.fetchAllLeverageBrackets(account)
	if err != nil {
		return nil, nil, err
	}

	// Cache both raw objects for the whole universe at once.
	cache := make(map[string]*binanceSymbolCacheEntry, len(symbols))
	for i := range symbols {
		info := &symbols[i]
		cache[info.Symbol] = &binanceSymbolCacheEntry{
			info:    info,
			bracket: brackets[info.Symbol], // may be nil if the universes differ
		}
	}

	b.cacheMu.Lock()
	b.symbolInfoCache = cache
	b.cacheMu.Unlock()

	entry, ok := cache[symbol]
	if !ok {
		return nil, nil, &internal.BinanceError{
			Message: fmt.Sprintf("Expected symbol info for %s, not found", symbol),
		}
	}
	if entry.bracket == nil {
		return nil, nil, &internal.BinanceError{
			Message: fmt.Sprintf("No leverage bracket found for %s", symbol),
		}
	}

	return entry.info, entry.bracket, nil
}

// fetchAllExchangeInfo fetches the public exchangeInfo universe (all symbols).
func (b *Binance) fetchAllExchangeInfo(account subsettings.BinanceAccount) ([]BinanceSymbolInfo, error) {
	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/exchangeInfo",
		&BinanceExchangeInfo{},
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}

	exchangeInfo, ok := result.(*BinanceExchangeInfo)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return exchangeInfo.Symbols, nil
}

// fetchAllLeverageBrackets fetches the signed leverageBracket universe (all
// symbols, one weight-1 request) and returns a symbol -> bracket map.
func (b *Binance) fetchAllLeverageBrackets(account subsettings.BinanceAccount) (map[string]*BinanceLeverageBracket, error) {
	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/leverageBracket",
		&[]BinanceLeverageBracket{},
		nil,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	brackets, ok := result.(*[]BinanceLeverageBracket)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	bracketsBySymbol := make(map[string]*BinanceLeverageBracket, len(*brackets))
	for i := range *brackets {
		bracket := &(*brackets)[i]
		bracketsBySymbol[bracket.Symbol] = bracket
	}

	return bracketsBySymbol, nil
}

// SymbolConfig retrieves symbol configuration (used to recover symbol margin type)
func (b *Binance) SymbolConfig(account subsettings.BinanceAccount, symbol string) (*BinanceSymbolConfig, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	result, err := b.Request(
		account,
		"GET",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/symbolConfig",
		&[]BinanceSymbolConfig{},
		params,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	configs, ok := result.(*[]BinanceSymbolConfig)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	if len(*configs) == 1 {
		return &(*configs)[0], nil
	}

	return nil, &internal.BinanceError{
		Message: fmt.Sprintf("Expected 1 symbol config, got %d", len(*configs)),
	}
}

// SetSymbolMarginType sets the margin type for a symbol
// marginType can be "ISOLATED" or "CROSSED"
// To be used only when no position is open for the symbol
func (b *Binance) SetSymbolMarginType(account subsettings.BinanceAccount, symbol string, marginType string) error {
	params := map[string]string{
		"symbol":     symbol,
		"marginType": marginType,
	}
	_, err := b.Request(
		account,
		"POST",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/marginType",
		&BinanceEmptyResponse{},
		params,
		nil,
		true,
	)
	if err != nil {
		// Check if error is -4046 (already set)
		if binanceErr, ok := err.(*internal.BinanceBadStatusCodeError); ok {
			if binanceErr.ErrorCode != nil && *binanceErr.ErrorCode == -4046 {
				b.logger.Warn("Symbol is already set to margin type", "symbol", symbol, "margin_type", marginType)
				return nil
			}
		}
		return err
	}

	return nil
}

// PlaceMarketOrder places a market order
func (b *Binance) PlaceMarketOrder(
	account subsettings.BinanceAccount,
	symbol string,
	side string, // "Buy" or "Sell"
	size string,
	takeProfit *float64,
	stopLoss *float64,
	closePosition bool,
) (*BinancePlaceOrder, error) {
	binanceSide := "BUY"
	if side == "Sell" {
		binanceSide = "SELL"
	}
	closeSide := "SELL"
	if side == "Sell" {
		closeSide = "BUY"
	}

	params := map[string]string{
		"symbol":   symbol,
		"side":     binanceSide,
		"type":     "MARKET",
		"quantity": size,
	}
	if closePosition {
		params["side"] = closeSide
		params["reduceOnly"] = "true"
	}

	result, err := b.Request(
		account,
		"POST",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/order",
		&BinancePlaceOrder{},
		params,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	order, ok := result.(*BinancePlaceOrder)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	// do not place any TP/SL if closing position
	if closePosition {
		return order, nil
	}

	// Set take profit if provided
	if takeProfit != nil {
		tpParams := map[string]string{
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "TAKE_PROFIT_MARKET",
			"triggerPrice":  strconv.FormatFloat(*takeProfit, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
		}

		_, err := b.Request(
			account,
			"POST",
			resolveBinanceFuturesBaseURL(account.IsDemo),
			"/v1/algoOrder",
			&BinancePlaceAlgoOrder{},
			tpParams,
			nil,
			true,
		)
		if err != nil {
			return order, err
		}
	}

	// Set stop loss if provided
	if stopLoss != nil {
		slParams := map[string]string{
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "STOP_MARKET",
			"triggerPrice":  strconv.FormatFloat(*stopLoss, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
		}

		_, err := b.Request(
			account,
			"POST",
			resolveBinanceFuturesBaseURL(account.IsDemo),
			"/v1/algoOrder",
			&BinancePlaceAlgoOrder{},
			slParams,
			nil,
			true,
		)
		if err != nil {
			return order, err
		}
	}

	return order, nil
}

// PlaceMarketOrderWebsocket places a market order on a websocket client for an account
func (b *Binance) PlaceMarketOrderWebsocket(
	accountKey string,
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	closePosition bool,
) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BinanceError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.PlaceMarketOrder(symbol, side, size, takeProfit, stopLoss, closePosition)
	if err != nil {
		return &internal.BinanceError{Message: "failed to place market order: " + err.Error()}
	}
	return nil
}

// fractionOrder fractions an order size into multiple smaller orders based on exchange limits
// Returns a slice of order sizes as strings, or an error if the order cannot be fractioned
func (b *Binance) fractionOrder(
	orderSize float64,
	lastPrice float64,
	symbolInfo *BinanceSymbolInfo,
	maxFractionedOrders int,
) ([]string, error) {
	lotSizeFilter := symbolInfo.LotSizeFilter()
	if lotSizeFilter == nil {
		return nil, &internal.BinanceError{Message: "lot size filter not found for symbol"}
	}

	// Extract exchange limits
	minQty := lotSizeFilter.MinOrderQty
	maxQty := lotSizeFilter.MaxMktOrderQty
	minValue := 2 * lotSizeFilter.MinNotionalValue
	minQtyByValue := minValue / lastPrice
	stepQty := lotSizeFilter.QtyStep
	stepPrecision := int(math.Max(0, math.Ceil(-math.Log10(stepQty))))

	// Validate minimum order size
	if orderSize < minQty {
		b.logger.Warn("Order size below exchange minimum",
			"order_size", orderSize,
			"min_qty", minQty,
			"symbol", symbolInfo.Symbol)
		return nil, &internal.BinanceError{
			Message: fmt.Sprintf("order size %.8f below exchange minimum %.8f", orderSize, minQty),
		}
	}

	// Helper function to round quantity to exchange step size
	roundToStep := func(qty float64) float64 {
		return minQty + math.Floor((qty-minQty)/stepQty)*stepQty
	}

	// Split order into fractions
	var fractionedOrderSizes []string
	remaining := orderSize

	for i := 0; i < maxFractionedOrders && remaining >= minQty; i++ {
		isLastFraction := i == maxFractionedOrders-1 || remaining <= maxQty

		var fractionSize float64
		if isLastFraction {
			// Last fraction: use ALL remaining quantity to ensure complete closure
			// Don't round down - we need to close everything that's left
			fractionSize = remaining
		} else {
			// Intermediate fraction: use maxQty (rounded to step)
			fractionSize = roundToStep(maxQty)
		}

		// Ensure fraction meets minimum quantity
		if fractionSize < minQty {
			break
		}

		// Check notional value (price * quantity)
		meetsNotionalValue := fractionSize >= minQtyByValue
		if !meetsNotionalValue && !isLastFraction {
			// Skip intermediate fractions that don't meet notional value
			continue
		}

		// Warn if placing order below notional value (only allowed on last fraction)
		if !meetsNotionalValue {
			b.logger.Warn("Placing order below min notional value to close position",
				"order_size", fractionSize,
				"min_qty_by_value", minQtyByValue,
				"order_value", fractionSize*lastPrice,
				"min_value", minValue)
		}

		// Add fraction and update remaining
		fractionedOrderSizes = append(fractionedOrderSizes, fmt.Sprintf("%.*f", stepPrecision, fractionSize))
		remaining -= fractionSize
	}

	// Verify we have fractions to execute
	if len(fractionedOrderSizes) == 0 {
		b.logger.Warn("Unable to fraction order",
			"order_size", orderSize,
			"min_qty", minQty,
			"max_qty", maxQty,
			"symbol", symbolInfo.Symbol)
		return nil, &internal.BinanceError{Message: "unable to fraction order"}
	}

	return fractionedOrderSizes, nil
}

// ClosePosition closes a position using fractioned orders
func (b *Binance) ClosePosition(
	account subsettings.BinanceAccount,
	symbol string,
	side string, // "Buy" or "Sell"
	size float64,
) error {
	// Get ticker to get last price for fractioning
	ticker, err := b.Ticker(account, symbol)
	if err != nil {
		return err
	}

	// get symbol info
	symbolInfo, _, err := b.SymbolInfo(account, symbol, true)
	if err != nil {
		return err
	}

	// Fraction the order
	fractionedOrderSizes, err := b.fractionOrder(
		size,
		ticker.Price.Float64(),
		symbolInfo,
		b.cfg.HttpClient.MaxConcurrent,
	)
	if err != nil {
		return err
	}

	b.logger.Info("Closing position with fractioned orders",
		"symbol", symbol,
		"side", side,
		"total_size", size,
		"fractions", fractionedOrderSizes)

	// Place all fractioned orders concurrently
	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	for _, fractionSize := range fractionedOrderSizes {
		wg.Add(1)
		go func(sizeStr string) {
			defer wg.Done()
			_, err := b.PlaceMarketOrder(account, symbol, side, sizeStr, nil, nil, true)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(fractionSize)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("errors closing position: %v", errors)
	}

	b.logger.Debug("Closed position", "symbol", symbol, "side", side)
	return nil
}

// ClosePositionWebsocket closes a position using fractioned orders via websocket
func (b *Binance) ClosePositionWebsocket(
	accountKey string,
	symbol string,
	side string, // "Buy" or "Sell"
	size float64,
) error {
	// Get account from accountKey
	account, ok := b.cfg.Accounts[accountKey]
	if !ok {
		return &internal.BinanceError{Message: fmt.Sprintf("account not found: %s", accountKey)}
	}

	// Get ticker to get last price for fractioning
	ticker, err := b.Ticker(account, symbol)
	if err != nil {
		return err
	}

	// get symbol info
	symbolInfo, _, err := b.SymbolInfo(account, symbol, true)
	if err != nil {
		return err
	}

	// Fraction the order
	fractionedOrderSizes, err := b.fractionOrder(
		size,
		ticker.Price.Float64(),
		symbolInfo,
		b.cfg.HttpClient.MaxConcurrent,
	)
	if err != nil {
		return err
	}

	b.logger.Info("Closing position with fractioned orders (websocket)",
		"symbol", symbol,
		"side", side,
		"total_size", size,
		"fractions", fractionedOrderSizes)

	// Place all fractioned orders concurrently
	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	for _, fractionSize := range fractionedOrderSizes {
		wg.Add(1)
		go func(sizeStr string) {
			defer wg.Done()
			err := b.PlaceMarketOrderWebsocket(accountKey, symbol, side, sizeStr, nil, nil, true)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(fractionSize)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("errors closing position: %v", errors)
	}

	b.logger.Debug("Closed position (websocket)", "symbol", symbol, "side", side)
	return nil
}

// SetPositionTpsl sets take profit and stop loss for a position
func (b *Binance) SetPositionTpsl(
	account subsettings.BinanceAccount,
	symbol string,
	positionSide string, // "Buy" or "Sell"
	takeProfit float64,
	stopLoss float64,
) error {
	closeSide := "SELL"
	if positionSide == "Sell" {
		closeSide = "BUY"
	}

	// First, close all algo orders
	closeParams := map[string]string{
		"symbol": symbol,
	}

	_, err := b.Request(
		account,
		"DELETE",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/algoOpenOrders",
		&BinanceEmptyResponse{},
		closeParams,
		nil,
		true,
	)
	if err != nil {
		return err
	}

	// Set take profit if requested (0 means no take profit)
	if takeProfit > 0 {
		tpParams := map[string]string{
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "TAKE_PROFIT_MARKET",
			"triggerPrice":  strconv.FormatFloat(takeProfit, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
		}

		_, err = b.Request(
			account,
			"POST",
			resolveBinanceFuturesBaseURL(account.IsDemo),
			"/v1/algoOrder",
			&BinancePlaceAlgoOrder{},
			tpParams,
			nil,
			true,
		)
		if err != nil {
			return err
		}
	}

	// Set stop loss if requested (0 means no stop loss)
	if stopLoss > 0 {
		slParams := map[string]string{
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "STOP_MARKET",
			"triggerPrice":  strconv.FormatFloat(stopLoss, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
		}

		_, err = b.Request(
			account,
			"POST",
			resolveBinanceFuturesBaseURL(account.IsDemo),
			"/v1/algoOrder",
			&BinancePlaceAlgoOrder{},
			slParams,
			nil,
			true,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetPositionTpslWebsocket sets take profit and stop loss for a position on a websocket client for an account
func (b *Binance) SetPositionTpslWebsocket(
	accountKey string,
	takeProfitOrderId int64,
	stopLossOrderId int64,
	symbol string,
	positionSide string, // "Buy" or "Sell"
	takeProfit float64,
	stopLoss float64,
) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BinanceError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.SetPositionTpsl(symbol, positionSide, takeProfit, stopLoss, takeProfitOrderId, stopLossOrderId)
	if err != nil {
		return &internal.BinanceError{Message: "failed to set position tpsl: " + err.Error()}
	}
	return nil
}

// ApiKeyPermissions retrieves the API key permissions for an account
func (b *Binance) ApiKeyPermissions(account subsettings.BinanceAccount) (*BinanceApiKeyPermissions, error) {
	result, err := b.Request(
		account,
		"GET",
		resolveBinanceSpotAccountBaseURL(account.IsDemo),
		"/v1/account/apiRestrictions",
		&BinanceApiKeyPermissions{},
		nil,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	permissions, ok := result.(*BinanceApiKeyPermissions)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return permissions, nil
}

// SpotAccountInfo retrieves current spot account information
func (b *Binance) SpotAccountInfo(account subsettings.BinanceAccount, omitZeroBalances bool) (*BinanceSpotAccountInfo, error) {
	params := map[string]string{}
	if omitZeroBalances {
		params["omitZeroBalances"] = "true"
	}

	result, err := b.Request(
		account,
		"GET",
		resolveBinanceSpotBaseURL(account.IsDemo),
		"/v3/account",
		&BinanceSpotAccountInfo{},
		params,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	info, ok := result.(*BinanceSpotAccountInfo)
	if !ok {
		return nil, &internal.BinanceError{Message: "type assertion failed"}
	}

	return info, nil
}

// SetLeverage sets leverage for a symbol
func (b *Binance) SetLeverage(account subsettings.BinanceAccount, symbol string, leverage int) error {
	params := map[string]string{
		"symbol":   symbol,
		"leverage": strconv.Itoa(leverage),
	}

	_, err := b.Request(
		account,
		"POST",
		resolveBinanceFuturesBaseURL(account.IsDemo),
		"/v1/leverage",
		&BinanceEmptyResponse{},
		params,
		nil,
		true,
	)
	return err
}

// Request makes a single HTTP request with retry logic
func (b *Binance) Request(
	account subsettings.BinanceAccount,
	method string,
	baseURL string,
	endpoint string,
	responseModel interface{},
	params map[string]string,
	jsonData map[string]interface{},
	signRequest bool,
) (interface{}, error) {
	// Acquire semaphore
	b.semaphore <- struct{}{}
	defer func() { <-b.semaphore }()

	maxRetries := b.cfg.HttpClient.RateLimiter.MaxRetries
	defaultRetry := b.cfg.HttpClient.RateLimiter.DefaultRetry

	for attempt := 0; attempt < maxRetries; attempt++ {
		var shouldRetry *subsettings.RetrySettings
		var retryWarning string

		// Try to acquire rate limiter
		if err := b.rateLimiter.TryAcquire(baseURL); err != nil {
			// Rate limit hit, retry with default settings
			shouldRetry = &defaultRetry
			retryWarning = fmt.Sprintf("Hit rate limit on %s", baseURL)
		} else {
			// Make the request
			var contentStr *string
			if jsonData != nil {
				contentBytes, err := json.Marshal(jsonData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal JSON: %w", err)
				}
				contentStrVal := string(contentBytes)
				contentStr = &contentStrVal
			}

			headers, newParams, err := b.headersAndParams(account, params, contentStr, signRequest)
			if err != nil {
				return nil, err
			}

			reqURL := baseURL + endpoint
			if len(newParams) > 0 {
				values := url.Values{}
				for k, v := range newParams {
					values.Set(k, v)
				}
				reqURL += "?" + values.Encode()
			}

			var body io.Reader
			if contentStr != nil {
				body = strings.NewReader(*contentStr)
			}

			// Debug: Log request details
			if b.logApiResponses {
				b.logger.Debug("Binance API Request", "method", method, "url", reqURL)
				if len(params) > 0 {
					b.logger.Debug("Binance API Request params", "params", params)
				}
				if contentStr != nil {
					b.logger.Debug("Binance API Request body", "body", *contentStr)
				}
			}

			req, err := http.NewRequest(method, reqURL, body)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			for k, v := range b.cfg.HttpClient.ExtraHeaders {
				req.Header.Set(k, v)
			}
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			resp, err := b.client.Do(req)
			if err != nil {
				if b.logApiResponses {
					b.logger.Error("Binance API Request failed", "error", err, "url", reqURL)
				}
				return nil, fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				if b.logApiResponses {
					b.logger.Error("Binance API Failed to read response body", "error", err, "url", reqURL)
				}
				return nil, fmt.Errorf("failed to read response: %w", err)
			}

			// Debug: Print raw response body
			if b.logApiResponses {
				b.logger.Debug("Binance API Response",
					"status", resp.StatusCode,
					"method", method,
					"endpoint", endpoint,
					"body", string(bodyBytes))
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				// Check if we should retry for this status code
				if retrySettings, ok := b.cfg.HttpClient.RateLimiter.RetryStatuscode[resp.StatusCode]; ok {
					shouldRetry = &retrySettings
					switch resp.StatusCode {
					case 418:
						retryWarning = fmt.Sprintf("IP has been auto-banned %d from %s", resp.StatusCode, baseURL)
					case 429:
						retryWarning = fmt.Sprintf("Hit rate limit %d from %s", resp.StatusCode, baseURL)
					default:
						retryWarning = fmt.Sprintf("Unexpected retry status code from %s", baseURL)
					}
				} else {
					// Parse error response
					var errorCode *int
					var errorMessage *string
					var jsonResp map[string]interface{}
					if err := json.Unmarshal(bodyBytes, &jsonResp); err == nil {
						if code, ok := jsonResp["code"].(float64); ok {
							codeInt := int(code)
							errorCode = &codeInt
						}
						if msg, ok := jsonResp["msg"].(string); ok {
							errorMessage = &msg
						}
					}

					if b.logApiResponses {
						b.logger.Error("Binance API Bad HTTP status",
							"status", resp.StatusCode,
							"error_code", errorCode,
							"error_message", errorMessage,
							"response", string(bodyBytes))
					}
					return nil, &internal.BinanceBadStatusCodeError{
						Message:      string(bodyBytes),
						StatusCode:   resp.StatusCode,
						ErrorCode:    errorCode,
						ErrorMessage: errorMessage,
					}
				}
			} else {
				// Success - parse result
				modelCopy := createBinanceModelCopy(responseModel)
				if err := json.Unmarshal(bodyBytes, modelCopy); err != nil {
					if b.logApiResponses {
						b.logger.Error("Binance API Failed to unmarshal result",
							"error", err,
							"body", string(bodyBytes),
							"url", reqURL)
					}
					return nil, fmt.Errorf("failed to unmarshal result: %w", err)
				}

				if b.logApiResponses {
					b.logger.Debug("Binance API Request successful", "method", method, "endpoint", endpoint)
				}
				return modelCopy, nil
			}
		}

		// Retry logic
		if shouldRetry != nil && attempt < maxRetries-1 {
			sleepTime := math.Min(
				shouldRetry.InitialRetryDelay*math.Pow(shouldRetry.RetryDelayFactor, float64(attempt)),
				shouldRetry.MaxRetryDelay,
			)
			jitter := 1.0 + rand.Float64()*shouldRetry.RetryDelayJitter
			sleepTime *= jitter

			b.logger.Warn("Rate limit hit, retrying",
				"warning", retryWarning,
				"retry_in_seconds", sleepTime)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}

	return nil, &internal.BinanceError{
		Message: fmt.Sprintf("Max retries (%d) reached for %s", maxRetries, baseURL),
	}
}

// headersAndParams generates authentication headers and params for Binance API
func (b *Binance) headersAndParams(
	account subsettings.BinanceAccount,
	params map[string]string,
	content *string,
	signRequest bool,
) (map[string]string, map[string]string, error) {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	newParams := make(map[string]string)
	for k, v := range params {
		newParams[k] = v
	}

	if signRequest {
		key, err := b.creds.Get(account.Key)
		if err != nil {
			return nil, nil, newExchangeError(ErrApiKeyNotSet, &internal.BinanceError{Message: "Binance API key is not set: " + err.Error()})
		}
		secret, err := b.creds.Get(account.Secret)
		if err != nil {
			return nil, nil, newExchangeError(ErrApiKeyNotSet, &internal.BinanceError{Message: "Binance API secret is not set: " + err.Error()})
		}

		apiKey := key
		apiSecret := secret
		receiveWindowMs := b.cfg.ReceiveWindowMs

		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		newParams["timestamp"] = timestamp
		newParams["recvWindow"] = strconv.Itoa(receiveWindowMs)

		// Build signature body
		values := url.Values{}
		for k, v := range newParams {
			values.Set(k, v)
		}
		signatureBody := values.Encode()
		if content != nil {
			signatureBody += *content
		}

		// Generate signature
		mac := hmac.New(sha256.New, []byte(apiSecret))
		mac.Write([]byte(signatureBody))
		signature := hex.EncodeToString(mac.Sum(nil))

		newParams["signature"] = signature
		headers["X-MBX-APIKEY"] = apiKey
	}

	setProxyHeader(headers, resolveBinanceProxyKey())

	return headers, newParams, nil
}

// InvalidateCache clears the symbol info cache
func (b *Binance) InvalidateCache() {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()
	b.symbolInfoCache = make(map[string]*binanceSymbolCacheEntry)
}

// createBinanceModelCopy creates a copy of the Binance model type
func createBinanceModelCopy(model interface{}) interface{} {
	switch model.(type) {
	case *BinancePosition:
		return &BinancePosition{}
	case *[]BinancePosition:
		return &[]BinancePosition{}
	case *BinancePositionGenericInfo:
		return &BinancePositionGenericInfo{}
	case *[]BinancePositionGenericInfo:
		return &[]BinancePositionGenericInfo{}
	case *BinanceAccountInfo:
		return &BinanceAccountInfo{}
	case *BinanceAccountInfoAssets:
		return &BinanceAccountInfoAssets{}
	case *BinanceTickerPrice:
		return &BinanceTickerPrice{}
	case *BinanceExchangeInfo:
		return &BinanceExchangeInfo{}
	case *BinanceSymbolInfo:
		return &BinanceSymbolInfo{}
	case *BinanceSymbolConfig:
		return &BinanceSymbolConfig{}
	case *[]BinanceSymbolConfig:
		return &[]BinanceSymbolConfig{}
	case *[]BinanceLeverageBracket:
		return &[]BinanceLeverageBracket{}
	case *BinancePlaceOrder:
		return &BinancePlaceOrder{}
	case *BinancePlaceAlgoOrder:
		return &BinancePlaceAlgoOrder{}
	case *BinanceOpenOrder:
		return &BinanceOpenOrder{}
	case *[]BinanceOpenOrder:
		return &[]BinanceOpenOrder{}
	case *BinancePositionModeResponse:
		return &BinancePositionModeResponse{}
	case *BinanceEmptyResponse:
		return &BinanceEmptyResponse{}
	case *BinanceApiKeyPermissions:
		return &BinanceApiKeyPermissions{}
	case *BinanceSpotAccountInfo:
		return &BinanceSpotAccountInfo{}
	default:
		// Fallback: use JSON marshal/unmarshal
		data, _ := json.Marshal(model)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return &result
	}
}
