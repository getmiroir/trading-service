package clients

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
)

const MS_IN_S = 1000

type hyperliquidExchangeAdapter struct {
	client        *HyperliquidExchange
	apiKeyCache   map[string]*HLExtraAgent
	apiKeyCacheMu sync.RWMutex
}

func (a *hyperliquidExchangeAdapter) ClosePosition(account subsettings.ExchangeAccount, position Position) error {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	closeOrderSide := utils.QuantityToSide(-float64(utils.SideToMultiplier(position.Side)))
	sizeStr := strconv.FormatFloat(position.Size, 'f', -1, 64)
	_, err := a.client.PlaceMarketOrder(*hlAccount, position.Symbol, closeOrderSide, sizeStr, nil, nil, true, true)
	return mapHyperliquidExchangeError(err)
}

func (a *hyperliquidExchangeAdapter) PlaceOrder(account subsettings.ExchangeAccount, symbol, side, size string, takeProfit, stopLoss *float64, reduceOnly, closeOnTrigger bool) (*PlaceOrder, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	data, err := a.client.PlaceMarketOrder(*hlAccount, symbol, side, size, takeProfit, stopLoss, reduceOnly, closeOnTrigger)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	orderID := hlOrderID(data)
	return &PlaceOrder{OrderID: orderID}, nil
}

func (a *hyperliquidExchangeAdapter) Positions(account subsettings.ExchangeAccount) ([]Position, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	positions, ctxs, _, err := a.client.Positions(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	result := make([]Position, len(positions))
	for i, p := range positions {
		markPrice := 0.0
		if i < len(ctxs) {
			markPrice = ctxs[i].MarkPx.Float64()
		}
		result[i] = newPositionFromHL(p, markPrice)
	}
	return result, nil
}

func (a *hyperliquidExchangeAdapter) Position(account subsettings.ExchangeAccount, symbol string) (*PositionGenericInfo, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	pos, _, err := a.client.Position(*hlAccount, symbol)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	if pos == nil {
		// Hyperliquid omits zero-size position entries. Configured leverage without
		// an open position is not reported here — callers that need to set leverage
		// before opening should just call SetLeverage.
		return nil, nil
	}
	info := newPositionGenericInfoFromHL(*pos)
	return &info, nil
}

func (a *hyperliquidExchangeAdapter) OpenStopOrders(account subsettings.ExchangeAccount) ([]OpenStopOrder, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	orders, err := a.client.OpenStopOrders(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	result := make([]OpenStopOrder, len(orders))
	for i, o := range orders {
		result[i] = newOpenStopOrderFromHL(o)
	}
	return result, nil
}

func (a *hyperliquidExchangeAdapter) SettleCoinBalance(account subsettings.ExchangeAccount) (*AccountBalanceCoin, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)

	accountType, err := a.client.UserAbstraction(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	if accountType != "unifiedAccount" {
		return nil, &HyperliquidInvalidAccountTypeError{AccountType: accountType}
	}

	spotState, err := a.client.SpotClearinghouseState(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	for _, b := range spotState.Balances {
		if b.Coin != hlAccount.SettleCoin {
			continue
		}
		total := b.Total.Float64()
		return &AccountBalanceCoin{
			Equity:           total,
			WalletBalance:    total,
			AvailableBalance: total - b.Hold.Float64(),
		}, nil
	}
	return &AccountBalanceCoin{}, nil
}

func (a *hyperliquidExchangeAdapter) AccountInfo(account subsettings.ExchangeAccount) (*AccountInfo, error) {
	marginMode := "ISOLATED"
	return &AccountInfo{AccountType: "hyperliquid", MarginMode: &marginMode}, nil
}

// SetAccountMarginMode is a no-op for Hyperliquid (cross margin is the default).
func (a *hyperliquidExchangeAdapter) SetAccountMarginMode(_ subsettings.ExchangeAccount, _ *string) error {
	return nil
}

// SymbolMarginType always returns "cross" for Hyperliquid perpetuals.
func (a *hyperliquidExchangeAdapter) SymbolMarginType(_ subsettings.ExchangeAccount, _ string) (string, error) {
	return "ISOLATED", nil
}

// SetSymbolMarginType is a no-op for Hyperliquid.
func (a *hyperliquidExchangeAdapter) SetSymbolMarginType(_ subsettings.ExchangeAccount, _, _ string) error {
	return nil
}

// SetAccountPositionMode is a no-op for Hyperliquid (one-way mode only).
func (a *hyperliquidExchangeAdapter) SetAccountPositionMode(_ subsettings.ExchangeAccount, _ string) error {
	return nil
}

func (a *hyperliquidExchangeAdapter) SetLeverage(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	return mapHyperliquidExchangeError(a.client.SetLeverage(*hlAccount, symbol, leverage))
}

func (a *hyperliquidExchangeAdapter) SetLeverageWebsocket(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	return mapHyperliquidExchangeError(a.client.SetLeverageWebsocket(account.Name, symbol, leverage))
}

func (a *hyperliquidExchangeAdapter) Ticker(account subsettings.ExchangeAccount, symbol string) (*Ticker, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	ctx, err := a.client.Ticker(*hlAccount, symbol)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	return &Ticker{Symbol: symbol, LastPrice: ctx.MarkPx.Float64()}, nil
}

func (a *hyperliquidExchangeAdapter) SymbolInfo(account subsettings.ExchangeAccount, symbol string, cached bool) (*SymbolInfo, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	entry, err := a.client.SymbolInfo(*hlAccount, symbol, cached)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	info := newSymbolInfoFromHL(symbol, *entry)
	return &info, nil
}

func (a *hyperliquidExchangeAdapter) SetPositionTpsl(account subsettings.ExchangeAccount, symbol, positionSide string, takeProfit, stopLoss *float64) error {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)

	// Hyperliquid does not replace existing TP/SL trigger orders when new ones
	// are placed, so collect all existing ones. Replaced legs are canceled
	// after the replacements are placed, so the position is never left
	// unprotected; legs requested as nil are canceled without a replacement.
	oldOids, err := a.openTpslOrderIDs(*hlAccount, symbol)
	if err != nil {
		return mapHyperliquidExchangeError(err)
	}

	if err := a.client.SetPositionTpsl(*hlAccount, symbol, positionSide, takeProfit, stopLoss); err != nil {
		return mapHyperliquidExchangeError(err)
	}

	return mapHyperliquidExchangeError(a.client.CancelOrders(*hlAccount, symbol, oldOids))
}

// tpslOidsFromState returns the order IDs of the open TP/SL orders already
// held in account state.
func tpslOidsFromState(tpOrder, slOrder *OpenStopOrder) ([]int64, error) {
	var oids []int64
	if tpOrder != nil && tpOrder.OrderID != nil {
		oid, err := strconv.ParseInt(*tpOrder.OrderID, 10, 64)
		if err != nil {
			return nil, err
		}
		oids = append(oids, oid)
	}
	if slOrder != nil && slOrder.OrderID != nil {
		oid, err := strconv.ParseInt(*slOrder.OrderID, 10, 64)
		if err != nil {
			return nil, err
		}
		oids = append(oids, oid)
	}
	return oids, nil
}

// openTpslOrderIDs returns the order IDs of the coin's open TP/SL trigger
// orders.
func (a *hyperliquidExchangeAdapter) openTpslOrderIDs(account subsettings.HyperliquidExchangeAccount, coin string) ([]int64, error) {
	orders, err := a.client.OpenStopOrders(account)
	if err != nil {
		return nil, err
	}
	var oids []int64
	for _, o := range orders {
		if o.Coin != coin {
			continue
		}
		kind := hlTpslKind(o)
		if kind == "TP" || kind == "SL" {
			oids = append(oids, o.Oid)
		}
	}
	return oids, nil
}

// HyperToExchangeSymbol returns the symbol unchanged — Hyperliquid symbols ARE the
// canonical Hyperliquid coin names (e.g. "BTC", "ETH").
func (a *hyperliquidExchangeAdapter) HyperToExchangeSymbol(_ subsettings.ExchangeAccount, hyperSymbol string) (string, error) {
	return hyperSymbol, nil
}

func (a *hyperliquidExchangeAdapter) GetApiKeyAndUserInformation(account subsettings.ExchangeAccount, _ bool) (*ApiKeyInfo, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)

	cacheKey := hlAccount.ActiveWalletAddress()
	a.apiKeyCacheMu.RLock()
	cached, ok := a.apiKeyCache[cacheKey]
	a.apiKeyCacheMu.RUnlock()
	if ok {
		return apiKeyInfoFromHLAgent(cached), nil
	}

	agent, err := a.client.ApiKeyInfo(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}

	a.apiKeyCacheMu.Lock()
	a.apiKeyCache[cacheKey] = agent
	a.apiKeyCacheMu.Unlock()

	return apiKeyInfoFromHLAgent(agent), nil
}

func (a *hyperliquidExchangeAdapter) UserRateLimit(account subsettings.ExchangeAccount) (*RateLimitInfo, error) {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	rateLimit, err := a.client.UserRateLimit(*hlAccount)
	if err != nil {
		return nil, mapHyperliquidExchangeError(err)
	}
	return &RateLimitInfo{
		Used: rateLimit.NRequestsUsed,
		Cap:  rateLimit.NRequestsCap,
	}, nil
}

func (a *hyperliquidExchangeAdapter) CancelStopOrders(account subsettings.ExchangeAccount, orders []OpenStopOrder) error {
	hlAccount := account.Account.(*subsettings.HyperliquidExchangeAccount)
	return mapHyperliquidExchangeError(a.client.CancelStopOrders(*hlAccount, orders))
}

func apiKeyInfoFromHLAgent(agent *HLExtraAgent) *ApiKeyInfo {
	var expiredAt *int64
	if agent.ValidUntil > 0 {
		v := agent.ValidUntil / MS_IN_S
		expiredAt = &v
	}
	return &ApiKeyInfo{ExpiredAt: expiredAt}
}

func (a *hyperliquidExchangeAdapter) InvalidateCache() {
	a.client.InvalidateCache()
	a.apiKeyCacheMu.Lock()
	a.apiKeyCache = make(map[string]*HLExtraAgent)
	a.apiKeyCacheMu.Unlock()
}

func (a *hyperliquidExchangeAdapter) SetPositionTpslWebsocket(
	account subsettings.ExchangeAccount,
	symbol string,
	positionSide string,
	takeProfit *float64,
	stopLoss *float64,
	tpOrder *OpenStopOrder,
	slOrder *OpenStopOrder,
) error {
	// Hyperliquid does not replace existing TP/SL trigger orders when new ones
	// are placed, so collect all existing ones (OIDs come from account state,
	// passed in by the symbol runner). Replaced legs are canceled after the
	// replacements are placed, so the position is never left unprotected; legs
	// requested as nil are canceled without a replacement.
	oldOids, err := tpslOidsFromState(tpOrder, slOrder)
	if err != nil {
		return mapHyperliquidExchangeError(err)
	}

	if err := a.client.SetPositionTpslWebsocket(account.Name, symbol, positionSide, takeProfit, stopLoss); err != nil {
		return mapHyperliquidExchangeError(err)
	}

	return mapHyperliquidExchangeError(a.client.CancelOrdersWebsocket(account.Name, symbol, oldOids))
}

func (a *hyperliquidExchangeAdapter) WebsocketsEnabled(_ subsettings.ExchangeAccount) bool {
	return true
}

func (a *hyperliquidExchangeAdapter) ConnectWebsockets(ctx context.Context) {
	a.client.ConnectAllWebsockets(ctx)
}

func (a *hyperliquidExchangeAdapter) CloseWebsockets() {
	a.client.CloseAllWebsockets()
}

// -------------------------------------------------------------------
// Normalization helpers
// -------------------------------------------------------------------

func newPositionFromHL(p HLAssetPosition, markPrice float64) Position {
	sz := p.Szi.Float64()
	side := "Buy"
	if sz < 0 {
		side = "Sell"
	}
	return Position{
		Symbol:        p.Coin,
		Side:          side,
		Size:          math.Abs(sz),
		PositionValue: math.Abs(p.PositionValue.Float64()),
		Leverage:      p.Leverage.Value,
		MarkPrice:     markPrice,
		EntryPrice:    p.EntryPx.Float64(),
		UnrealizedPnl: p.UnrealizedPnl.Float64(),
		LiqPrice:      p.LiquidationPx.Float64(),
	}
}

func newPositionGenericInfoFromHL(p HLAssetPosition) PositionGenericInfo {
	sz := p.Szi.Float64()
	side := "Buy"
	if sz < 0 {
		side = "Sell"
	}
	return PositionGenericInfo{
		Symbol:   p.Coin,
		Leverage: p.Leverage.Value,
		Side:     side,
		Size:     math.Abs(sz),
	}
}

func newOpenStopOrderFromHL(o HLOpenOrder) OpenStopOrder {
	oid := strconv.FormatInt(o.Oid, 10)
	return OpenStopOrder{
		AccountType:  "hyperliquid",
		OrderID:      &oid,
		AlgoID:       nil,
		Symbol:       o.Coin,
		OrderType:    hlTpslKind(o),
		TriggerPrice: o.TriggerPx.Float64(),
	}
}

func newSymbolInfoFromHL(coin string, entry HLUniverseEntry) SymbolInfo {
	// szDecimals defines the quantity step: QtyStep = 10^(-szDecimals)
	qtyStep := 1.0
	for i := 0; i < entry.SzDecimals; i++ {
		qtyStep /= 10
	}
	return SymbolInfo{
		Symbol:           coin,
		MinOrderQty:      qtyStep,
		MaxOrderQty:      1e9,
		MaxMkOrderQty:    1e9,
		MinNotionalValue: 10.0, // Hyperliquid minimum notional ~10 USDC
		QtyStep:          qtyStep,
		MinPrice:         0,
		MaxPrice:         1e9,
		// HL has no fixed tick size, but perp prices allow at most
		// 6 - szDecimals decimal places (5-sig-fig rounding is handled
		// separately by hlPriceToWire).
		TickSize:    math.Pow10(entry.SzDecimals - 6),
		GrowthMode:  entry.GrowthMode == "enabled",
		MaxLeverage: entry.MaxLeverage,
	}
}

// HLOrderIDFromData extracts the first available order ID from an HLOrderData response.
func HLOrderIDFromData(data *HLOrderData) string {
	return hlOrderID(data)
}

// hlOrderID extracts the first available order ID from an HLOrderData response.
func hlOrderID(data *HLOrderData) string {
	if data == nil {
		return ""
	}
	for _, s := range data.Statuses {
		if s.Filled != nil {
			return strconv.FormatInt(s.Filled.Oid, 10)
		}
		if s.Resting != nil {
			return strconv.FormatInt(s.Resting.Oid, 10)
		}
	}
	return ""
}

// mapHyperliquidExchangeError maps known Hyperliquid exchange errors to common sentinel errors.
func mapHyperliquidExchangeError(err error) error {
	if err == nil {
		return nil
	}
	var e *internal.HyperliquidExchangeError
	if errors.As(err, &e) {
		msg := e.Message
		switch {
		case containsAny(msg, "Insufficient margin", "not enough margin", "does not have sufficient margin"):
			return newExchangeError(ErrNotEnoughMargin, err)
		case containsAny(msg, "Asset not found", "not found in universe"):
			return newExchangeError(ErrSymbolNotExist, err)
		case containsAny(msg, "exceed margin tier limit", "exceed the margin tier limit"):
			return newExchangeError(ErrPositionExceedsMarginTier, err)
		case isOpenInterestCapMessage(msg):
			return newExchangeError(ErrOpenInterestCap, err)
		}
	}
	return err
}

// isOpenInterestCapMessage reports whether a Hyperliquid message refuses an
// order because the asset sits at its open interest cap.
func isOpenInterestCapMessage(msg string) bool {
	lowered := strings.ToLower(msg)
	return strings.Contains(lowered, "open interest") && strings.Contains(lowered, "cap")
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// Ensure hyperliquidExchangeAdapter implements exchangeClient at compile time.
var _ exchangeClient = (*hyperliquidExchangeAdapter)(nil)
