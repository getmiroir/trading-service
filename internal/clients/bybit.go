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

// Bybit represents the Bybit API client
type Bybit struct {
	client           *http.Client
	websocketClients map[string]*BybitWebsocket // one client per account
	symbolInfoCache  map[string]*BybitSymbolInfo
	cacheMu          sync.RWMutex
	semaphore        chan struct{} // Buffered channel for concurrency control
	rateLimiter      *RateLimiter
	cfg              subsettings.Bybit
	creds            *secrets.SecretCache
	logApiResponses  bool
	logger           *slog.Logger
}

// NewBybit creates a new Bybit client
func NewBybit(cfg subsettings.Bybit, logApiResponses bool) *Bybit {
	client := &http.Client{
		Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HttpClient.PoolSize,
			MaxIdleConnsPerHost: cfg.HttpClient.PoolSize,
		},
	}

	creds := secrets.NewSecretCache()

	// create websocket clients for each account
	websocketClients := make(map[string]*BybitWebsocket)
	for accountKey, account := range cfg.Accounts {
		websocketClients[accountKey] = NewBybitWebsocket(
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

	return &Bybit{
		client:           client,
		websocketClients: websocketClients,
		symbolInfoCache:  make(map[string]*BybitSymbolInfo),
		semaphore:        make(chan struct{}, cfg.HttpClient.MaxConcurrent),
		rateLimiter:      NewRateLimiter(limits),
		cfg:              cfg,
		creds:            creds,
		logApiResponses:  logApiResponses,
		logger:           logging.NewLogger("bybit-client"),
	}
}

// ConnectWebsocket connects to a websocket client for an account
func (b *Bybit) ConnectWebsocket(accountKey string) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BybitError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.Connect()
	if err != nil {
		return &internal.BybitError{Message: "failed to connect to websocket: " + err.Error()}
	}
	return nil
}

// CloseWebsocket closes a websocket client for an account
func (b *Bybit) CloseWebsocket(accountKey string) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BybitError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.Close()
	if err != nil {
		return &internal.BybitError{Message: "failed to close websocket: " + err.Error()}
	}
	return nil
}

// ConnectAllWebsockets connects all websocket clients and starts supervisors.
func (b *Bybit) ConnectAllWebsockets(ctx context.Context) {
	for accountKey, websocket := range b.websocketClients {
		account, ok := b.cfg.Accounts[accountKey]
		if !ok || !BybitWebsocketsEnabled(account.IsDemo, account.IsTestnet) {
			continue
		}
		b.logger.Info("Connecting websocket", "account", accountKey)
		websocket.ConnectWithSupervisor(ctx)
	}
}

// CloseAllWebsockets closes all websocket clients.
func (b *Bybit) CloseAllWebsockets() {
	for accountKey, websocket := range b.websocketClients {
		if err := websocket.Close(); err != nil {
			b.logger.Warn("Failed to close websocket", "account", accountKey, "error", err)
		}
	}
}

// Positions retrieves all positions for an account
func (b *Bybit) Positions(account subsettings.BybitAccount) ([]BybitPosition, error) {
	results, err := b.RequestPaginated(
		account,
		"GET",
		"/position/list",
		&BybitPosition{},
		map[string]string{
			"category":   "linear",
			"settleCoin": account.SettleCoin,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	positions := make([]BybitPosition, len(results))
	for i, r := range results {
		if pos, ok := r.(*BybitPosition); ok {
			positions[i] = *pos
		} else {
			return nil, &internal.BybitError{Message: "type assertion failed"}
		}
	}
	return positions, nil
}

// Position retrieves a single position for a symbol
func (b *Bybit) Position(account subsettings.BybitAccount, symbol string) (*BybitPositionGenericInfo, error) {
	results, err := b.RequestPaginated(
		account,
		"GET",
		"/position/list",
		&BybitPositionGenericInfo{},
		map[string]string{
			"category": "linear",
			"symbol":   symbol,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Convert to PositionGenericInfo slice
	infos := make([]*BybitPositionGenericInfo, len(results))
	for i, r := range results {
		if info, ok := r.(*BybitPositionGenericInfo); ok {
			infos[i] = info
		} else {
			return nil, &internal.BybitError{Message: "type assertion failed"}
		}
	}

	if len(infos) == 1 {
		return infos[0], nil
	}

	// Filter by valid status
	var validInfos []*BybitPositionGenericInfo
	for _, info := range infos {
		if info.PositionStatus != "" {
			validInfos = append(validInfos, info)
		}
	}

	if len(validInfos) == 1 {
		return validInfos[0], nil
	}
	// if no positions are found, return nil
	if len(validInfos) == 0 {
		return nil, nil
	}

	return nil, &internal.BybitError{
		Message: fmt.Sprintf("Expected 0 or 1 position in response for %s, got %d", symbol, len(validInfos)),
	}
}

// OpenStopOrders retrieves all open stop orders for an account
func (b *Bybit) OpenStopOrders(account subsettings.BybitAccount) ([]BybitOpenOrder, error) {
	var orders []BybitOpenOrder
	// Fetch conditional orders (StopOrder includes untriggered TP/SL orders)
	results, err := b.RequestPaginated(
		account,
		"GET",
		"/order/realtime",
		&BybitOpenOrder{},
		map[string]string{
			"category":    "linear",
			"settleCoin":  account.SettleCoin,
			"openOnly":    "0",
			"orderFilter": "StopOrder",
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if order, ok := result.(*BybitOpenOrder); ok {
			orders = append(orders, *order)
		} else {
			return nil, &internal.BybitError{Message: "type assertion failed for conditional order"}
		}
	}

	return orders, nil
}

// AccountBalance retrieves account balance
func (b *Bybit) AccountBalance(account subsettings.BybitAccount) (*BybitAccountBalance, error) {
	results, err := b.RequestPaginated(
		account,
		"GET",
		"/account/wallet-balance",
		&BybitAccountBalance{},
		map[string]string{
			"accountType": "UNIFIED",
			"coin":        account.SettleCoin,
		},
		nil,
	)
	if err != nil {
		if b.logApiResponses {
			b.logger.Error("AccountBalance: RequestPaginated failed", "error", err)
		}
		return nil, err
	}

	if b.logApiResponses {
		b.logger.Debug("AccountBalance: Got results", "count", len(results))
	}

	if len(results) == 1 {
		if balance, ok := results[0].(*BybitAccountBalance); ok {
			if b.logApiResponses {
				b.logger.Debug("AccountBalance: Successfully parsed balance", "coins", len(balance.Coin))
			}
			return balance, nil
		}
		if b.logApiResponses {
			b.logger.Error("AccountBalance: Type assertion failed", "result_type", fmt.Sprintf("%T", results[0]))
		}
	}

	return nil, &internal.BybitError{
		Message: fmt.Sprintf("Expected 1 account balance in response, got %d", len(results)),
	}
}

// AccountInfo retrieves account information
func (b *Bybit) AccountInfo(account subsettings.BybitAccount) (*BybitAccountInfo, error) {
	result, err := b.Request(
		account,
		"GET",
		"/account/info",
		&BybitAccountInfo{},
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if info, ok := result.(*BybitAccountInfo); ok {
		return info, nil
	}

	return nil, &internal.BybitError{Message: "type assertion failed"}
}

// ApiKeyInfo retrieves the API key information including the user ID
func (b *Bybit) ApiKeyInfo(account subsettings.BybitAccount) (*BybitApiKeyInfoResponse, error) {
	result, err := b.Request(
		account,
		"GET",
		"/user/query-api",
		&BybitApiKeyInfoResponse{},
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*BybitApiKeyInfoResponse)
	if !ok {
		return nil, &internal.BybitError{Message: "type assertion failed"}
	}

	return resp, nil
}

// AccountUid retrieves the account UID and wallet types
func (b *Bybit) AccountUid(account subsettings.BybitAccount) (string, error) {
	result, err := b.Request(
		account,
		"GET",
		"/user/get-member-type",
		&BybitMemberTypeResponse{},
		nil,
		nil,
	)
	if err != nil {
		return "", err
	}

	resp, ok := result.(*BybitMemberTypeResponse)
	if !ok {
		return "", &internal.BybitError{Message: "type assertion failed"}
	}

	if len(resp.Accounts) == 0 {
		return "", &internal.BybitError{Message: "no accounts in response"}
	}

	return resp.Accounts[0].UID, nil
}

// SettleCoinBalance retrieves the settle coin balance
func (b *Bybit) SettleCoinBalance(account subsettings.BybitAccount) (*BybitAccountBalanceCoin, error) {
	balance, err := b.AccountBalance(account)
	if err != nil {
		return nil, err
	}

	var coins []BybitAccountBalanceCoin
	for _, coin := range balance.Coin {
		if coin.Coin == account.SettleCoin {
			coins = append(coins, coin)
		}
	}

	if len(coins) == 1 {
		return &coins[0], nil
	}

	return nil, &internal.BybitError{
		Message: fmt.Sprintf("Expected 1 coin balance entry for %s, got %d", account.SettleCoin, len(coins)),
	}
}

// Ticker retrieves ticker information
func (b *Bybit) Ticker(account subsettings.BybitAccount, symbol string) (*BybitTicker, error) {
	results, err := b.RequestPaginated(
		account,
		"GET",
		"/market/tickers",
		&BybitTicker{},
		map[string]string{
			"category": "linear",
			"symbol":   symbol,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(results) == 1 {
		if ticker, ok := results[0].(*BybitTicker); ok {
			return ticker, nil
		}
	}

	if len(results) == 0 {
		return nil, nil
	}

	return nil, &internal.BybitError{
		Message: fmt.Sprintf("Expected 1 ticker in response for %s, got %d", symbol, len(results)),
	}
}

// SymbolInfo retrieves symbol information (with optional caching)
func (b *Bybit) SymbolInfo(account subsettings.BybitAccount, symbol string, cached bool) (*BybitSymbolInfo, error) {
	if cached {
		b.cacheMu.RLock()
		if info, ok := b.symbolInfoCache[symbol]; ok {
			b.cacheMu.RUnlock()
			return info, nil
		}
		b.cacheMu.RUnlock()
	}

	results, err := b.RequestPaginated(
		account,
		"GET",
		"/market/instruments-info",
		&BybitSymbolInfo{},
		map[string]string{
			"category": "linear",
			"symbol":   symbol,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, &internal.BybitError{
			Message: fmt.Sprintf("Expected 1 symbol info in response, got %d", len(results)),
		}
	}

	info, ok := results[0].(*BybitSymbolInfo)
	if !ok {
		return nil, &internal.BybitError{Message: "type assertion failed"}
	}

	// Cache the result
	b.cacheMu.Lock()
	b.symbolInfoCache[symbol] = info
	b.cacheMu.Unlock()

	return info, nil
}

// PlaceMarketOrder places a market order
func (b *Bybit) PlaceMarketOrder(
	account subsettings.BybitAccount,
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	reduceOnly bool,
	closeOnTrigger bool,
) (*BybitPlaceOrder, error) {
	params := map[string]interface{}{
		"category":       "linear",
		"symbol":         symbol,
		"orderType":      "Market",
		"side":           side,
		"qty":            size,
		"reduceOnly":     reduceOnly,
		"closeOnTrigger": closeOnTrigger,
	}

	if takeProfit != nil || stopLoss != nil {
		params["tpslMode"] = "Full"
	}
	if takeProfit != nil {
		params["takeProfit"] = strconv.FormatFloat(*takeProfit, 'f', -1, 64)
		params["tpTriggerBy"] = "LastPrice"
	}
	if stopLoss != nil {
		params["stopLoss"] = strconv.FormatFloat(*stopLoss, 'f', -1, 64)
		params["slTriggerBy"] = "LastPrice"
	}

	result, err := b.Request(
		account,
		"POST",
		"/order/create",
		&BybitPlaceOrder{},
		nil,
		params,
	)
	if err != nil {
		return nil, err
	}

	if order, ok := result.(*BybitPlaceOrder); ok {
		return order, nil
	}

	return nil, &internal.BybitError{Message: "type assertion failed"}
}

// PlaceMarketOrderWebsocket places a market order on a websocket client for an account
func (b *Bybit) PlaceMarketOrderWebsocket(
	accountKey string,
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	reduceOnly bool,
	closeOnTrigger bool,
) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BybitError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.PlaceMarketOrder(symbol, side, size, takeProfit, stopLoss, reduceOnly, closeOnTrigger)
	if err != nil {
		return &internal.BybitError{Message: "failed to place market order: " + err.Error()}
	}
	return nil
}

// SetPositionTpsl sets take profit and stop loss for a position
func (b *Bybit) SetPositionTpsl(
	account subsettings.BybitAccount,
	symbol string,
	takeProfit float64,
	stopLoss float64,
) error {
	params := map[string]interface{}{
		"category":    "linear",
		"symbol":      symbol,
		"tpslMode":    "Full",
		"tpTriggerBy": "LastPrice",
		"slTriggerBy": "LastPrice",
		"takeProfit":  strconv.FormatFloat(takeProfit, 'f', -1, 64),
		"stopLoss":    strconv.FormatFloat(stopLoss, 'f', -1, 64),
	}

	_, err := b.Request(
		account,
		"POST",
		"/position/trading-stop",
		&BybitEmptyResponse{},
		nil,
		params,
	)
	return err
}

// Both takeProfitOrderId and stopLossOrderId must be provided to amend the TP/SL orders
func (b *Bybit) SetPositionTpslWebsocket(
	accountKey string,
	symbol string,
	takeProfitOrderId *string,
	stopLossOrderId *string,
	takeProfit float64,
	stopLoss float64,
) error {
	websocket := b.websocketClients[accountKey]
	if websocket == nil {
		return &internal.BybitError{Message: "websocket client not found for account: " + accountKey}
	}
	err := websocket.SetPositionTpsl(symbol, takeProfitOrderId, stopLossOrderId, takeProfit, stopLoss)
	if err != nil {
		return &internal.BybitError{Message: "failed to set position tpsl: " + err.Error()}
	}
	return nil
}

// SetLeverage sets leverage for a symbol
func (b *Bybit) SetLeverage(account subsettings.BybitAccount, symbol string, leverage int) error {
	params := map[string]interface{}{
		"category":     "linear",
		"symbol":       symbol,
		"buyLeverage":  strconv.Itoa(leverage),
		"sellLeverage": strconv.Itoa(leverage),
	}

	_, err := b.Request(
		account,
		"POST",
		"/position/set-leverage",
		&BybitEmptyResponse{},
		nil,
		params,
	)

	if err != nil {
		if bybitErr, ok := err.(*internal.BybitBadReturnCodeError); ok {
			if bybitErr.RetCode == 110043 {
				b.logger.Warn("Symbol is already set to leverage", "symbol", symbol, "leverage", leverage)
				return nil
			}
		}
		return err
	}

	return err
}

// SetAccountPositionMode sets the position mode for an account
// positionMode can be "hedge" or "one-way"
func (b *Bybit) SetAccountPositionMode(account subsettings.BybitAccount, positionMode string) error {
	if positionMode != "hedge" && positionMode != "one-way" {
		return fmt.Errorf("invalid position mode: %s", positionMode)
	}
	mode := 0
	if positionMode == "hedge" {
		mode = 3
	}

	params := map[string]interface{}{
		"category": "linear",
		"coin":     account.SettleCoin,
		"mode":     mode,
	}

	_, err := b.Request(
		account,
		"POST",
		"/position/switch-mode",
		&BybitEmptyResponse{},
		nil,
		params,
	)
	return err
}

// SetAccountMarginMode sets the margin mode for an account
// marginMode can be "ISOLATED_MARGIN", "REGULAR_MARGIN", or "PORTFOLIO_MARGIN"
func (b *Bybit) SetAccountMarginMode(account subsettings.BybitAccount, marginMode string) error {
	params := map[string]interface{}{
		"setMarginMode": marginMode,
	}

	_, err := b.Request(
		account,
		"POST",
		"/account/set-margin-mode",
		&BybitEmptyResponse{},
		nil,
		params,
	)
	return err
}

// RequestPaginated makes a paginated request
func (b *Bybit) RequestPaginated(
	account subsettings.BybitAccount,
	method string,
	endpoint string,
	model interface{},
	params map[string]string,
	jsonData map[string]interface{},
) ([]interface{}, error) {
	paramsWithCursor := make(map[string]string)
	for k, v := range params {
		paramsWithCursor[k] = v
	}

	var allResults []interface{}
	done := false

	for !done {

		result, err := b.Request(
			account,
			method,
			endpoint,
			&BybitPaginatedResponse{},
			paramsWithCursor,
			jsonData,
		)
		if err != nil {
			return nil, err
		}

		paginatedResp, ok := result.(*BybitPaginatedResponse)
		if !ok {
			return nil, &internal.BybitError{Message: "type assertion failed"}
		}

		if b.logApiResponses {
			b.logger.Debug("RequestPaginated: Found items in Results", "count", len(paginatedResp.Results))
		}
		// Parse results into the target model
		for i, item := range paginatedResp.Results {
			itemBytes, err := json.Marshal(item)
			if err != nil {
				if b.logApiResponses {
					b.logger.Error("RequestPaginated: Failed to marshal result item", "index", i, "error", err)
				}
				return nil, fmt.Errorf("failed to marshal result item: %w", err)
			}
			if b.logApiResponses {
				b.logger.Debug("RequestPaginated: Item JSON", "index", i, "json", string(itemBytes))
			}

			modelCopy := createModelCopy(model)
			if err := json.Unmarshal(itemBytes, modelCopy); err != nil {
				if b.logApiResponses {
					b.logger.Error("RequestPaginated: Failed to unmarshal result item",
						"index", i, "error", err, "json", string(itemBytes))
				}
				return nil, fmt.Errorf("failed to unmarshal result item: %w", err)
			}

			allResults = append(allResults, modelCopy)
		}

		if paginatedResp.NextPageCursor == nil || *paginatedResp.NextPageCursor == "" || len(paginatedResp.Results) == 0 {
			done = true
		} else {
			paramsWithCursor["cursor"] = *paginatedResp.NextPageCursor
		}
	}

	return allResults, nil
}

// createModelCopy creates a copy of the model type (workaround for Go 1.13 lack of generics)
func createModelCopy(model interface{}) interface{} {
	switch model.(type) {
	case *BybitApiKeyInfoResponse:
		return &BybitApiKeyInfoResponse{}
	case *BybitMemberTypeResponse:
		return &BybitMemberTypeResponse{}
	case *BybitPosition:
		return &BybitPosition{}
	case *BybitPositionGenericInfo:
		return &BybitPositionGenericInfo{}
	case *BybitAccountBalance:
		return &BybitAccountBalance{}
	case *BybitAccountBalanceCoin:
		return &BybitAccountBalanceCoin{}
	case *BybitAccountInfo:
		return &BybitAccountInfo{}
	case *BybitTicker:
		return &BybitTicker{}
	case *BybitSymbolInfo:
		return &BybitSymbolInfo{}
	case *BybitPaginatedResponse:
		return &BybitPaginatedResponse{}
	case *BybitPlaceOrder:
		return &BybitPlaceOrder{}
	case *BybitOpenOrder:
		return &BybitOpenOrder{}
	case *BybitEmptyResponse:
		return &BybitEmptyResponse{}
	default:
		// Fallback: use JSON marshal/unmarshal - but return a pointer to map
		data, _ := json.Marshal(model)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return &result
	}
}

// Request makes a single HTTP request with retry logic
func (b *Bybit) Request(
	account subsettings.BybitAccount,
	method string,
	endpoint string,
	responseModel interface{},
	params map[string]string,
	jsonData map[string]interface{},
) (interface{}, error) {
	// Acquire semaphore
	b.semaphore <- struct{}{}
	defer func() { <-b.semaphore }()

	maxRetries := b.cfg.HttpClient.RateLimiter.MaxRetries
	defaultRetry := b.cfg.HttpClient.RateLimiter.DefaultRetry
	baseURL := resolveBybitBaseURL(account.IsDemo, account.IsTestnet)

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

			headers, err := b.headers(account, params, contentStr)
			if err != nil {
				return nil, err
			}

			reqURL := baseURL + endpoint
			if len(params) > 0 {
				values := url.Values{}
				for k, v := range params {
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
				b.logger.Debug("Bybit API Request", "method", method, "url", reqURL)
				if len(params) > 0 {
					b.logger.Debug("Bybit API Request params", "params", params)
				}
				if contentStr != nil {
					b.logger.Debug("Bybit API Request body", "body", *contentStr)
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
					b.logger.Error("Bybit API Request failed", "error", err, "url", reqURL)
				}
				return nil, fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				if b.logApiResponses {
					b.logger.Error("Bybit API Failed to read response body", "error", err, "url", reqURL)
				}
				return nil, fmt.Errorf("failed to read response: %w", err)
			}

			// Debug: Print raw response body
			if b.logApiResponses {
				b.logger.Debug("Bybit API Response",
					"status", resp.StatusCode,
					"method", method,
					"endpoint", endpoint,
					"body", string(bodyBytes))
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if b.logApiResponses {
					b.logger.Error("Bybit API Bad HTTP status",
						"status", resp.StatusCode,
						"response", string(bodyBytes))
				}
				return nil, &internal.BybitBadReturnCodeError{
					Message: "Bad HTTP status",
					RetCode: resp.StatusCode,
				}
			}

			var jsonResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonResp); err != nil {
				if b.logApiResponses {
					b.logger.Error("Bybit API Failed to unmarshal response",
						"error", err,
						"body", string(bodyBytes),
						"url", reqURL)
				}
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			retCode, _ := jsonResp["retCode"].(float64)
			if retCode != 0 {
				retCodeInt := int(retCode)
				retMsg, _ := jsonResp["retMsg"].(string)
				if b.logApiResponses {
					b.logger.Error("Bybit API returned non-zero retCode",
						"retCode", retCodeInt,
						"retMsg", retMsg,
						"url", reqURL)
				}
				// Check if we should retry for this return code
				if retrySettings, ok := b.cfg.HttpClient.RateLimiter.RetryRetcode[retCodeInt]; ok {
					shouldRetry = &retrySettings
					retryWarning = fmt.Sprintf("Hit %d rate limit on %s", retCodeInt, baseURL)
				} else {
					return nil, &internal.BybitBadReturnCodeError{
						Message: retMsg,
						RetCode: retCodeInt,
					}
				}
			} else {
				// Success - parse result
				result, ok := jsonResp["result"]
				if !ok {
					if b.logApiResponses {
						b.logger.Error("Bybit API Response missing 'result' field", "url", reqURL)
					}
					return nil, &internal.BybitError{Message: "no result in response"}
				}

				resultBytes, err := json.Marshal(result)
				if err != nil {
					if b.logApiResponses {
						b.logger.Error("Bybit API Failed to marshal result", "error", err, "url", reqURL)
					}
					return nil, fmt.Errorf("failed to marshal result: %w", err)
				}

				modelCopy := createModelCopy(responseModel)
				if err := json.Unmarshal(resultBytes, modelCopy); err != nil {
					if b.logApiResponses {
						b.logger.Error("Bybit API Failed to unmarshal result",
							"error", err,
							"result", string(resultBytes),
							"url", reqURL)
					}
					return nil, fmt.Errorf("failed to unmarshal result: %w", err)
				}

				if b.logApiResponses {
					b.logger.Debug("Bybit API Request successful", "method", method, "endpoint", endpoint)
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

	return nil, &internal.BybitError{
		Message: fmt.Sprintf("Max retries (%d) reached for %s", maxRetries, baseURL),
	}
}

// headers generates authentication headers for Bybit API
func (b *Bybit) headers(account subsettings.BybitAccount, params map[string]string, content *string) (map[string]string, error) {
	apiKey, keyErr := b.creds.Get(account.Key)
	if keyErr != nil {
		return nil, newExchangeError(ErrApiKeyNotSet, &internal.BybitError{Message: "Bybit API key is not set: " + keyErr.Error()})
	}
	apiSecret, secretErr := b.creds.Get(account.Secret)
	if secretErr != nil {
		return nil, newExchangeError(ErrApiKeyNotSet, &internal.BybitError{Message: "Bybit API secret is not set: " + secretErr.Error()})
	}

	receiveWindowMs := b.cfg.ReceiveWindowMs

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signatureBody := timestamp + apiKey + strconv.Itoa(receiveWindowMs)

	if params != nil {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		signatureBody += values.Encode()
	}

	if content != nil {
		signatureBody += *content
	}

	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(signatureBody))
	signature := hex.EncodeToString(mac.Sum(nil))

	headers := map[string]string{
		"Content-Type":       "application/json",
		"X-BAPI-API-KEY":     apiKey,
		"X-BAPI-SIGN-TYPE":   "2",
		"X-BAPI-TIMESTAMP":   timestamp,
		"X-BAPI-RECV-WINDOW": strconv.Itoa(receiveWindowMs),
		"X-BAPI-SIGN":        signature,
	}

	setProxyHeader(headers, resolveBybitProxyKey())

	return headers, nil
}

// InvalidateCache clears the symbol info cache
func (b *Bybit) InvalidateCache() {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()
	b.symbolInfoCache = make(map[string]*BybitSymbolInfo)
}
