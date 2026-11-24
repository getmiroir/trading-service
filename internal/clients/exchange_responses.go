package clients

import (
	"math"
	"strconv"
	"time"
)

type Position struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"` // "Buy" or "Sell"
	Size          float64 `json:"size"`
	PositionValue float64 `json:"positionValue"`
	Leverage      float64 `json:"leverage"`
	MarkPrice     float64 `json:"markPrice"`
	EntryPrice    float64 `json:"entryPrice"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
	LiqPrice      float64 `json:"liqPrice"`
}

func NewPositionFromBybitPosition(position BybitPosition) Position {
	return Position{
		Symbol:        position.Symbol,
		Side:          position.Side,
		Size:          position.Size.Float64(),
		PositionValue: position.PositionValue.Float64(),
		Leverage:      position.Leverage.Float64(),
		MarkPrice:     position.MarkPrice.Float64(),
		EntryPrice:    position.AvgPrice.Float64(),
		UnrealizedPnl: position.UnrealisedPnl.Float64(),
		LiqPrice:      position.LiqPrice.Float64(),
	}
}

func NewPositionFromBinancePosition(position BinancePosition) Position {
	return Position{
		Symbol:        position.Symbol,
		Side:          position.Side(),
		Size:          math.Abs(position.PositionAmt.Float64()),
		PositionValue: position.Notional.Float64(),
		Leverage:      float64(position.Leverage.Int64()),
		MarkPrice:     position.MarkPrice.Float64(),
		EntryPrice:    position.EntryPrice.Float64(),
		UnrealizedPnl: position.UnrealizedProfit.Float64(),
		LiqPrice:      position.LiquidationPrice.Float64(),
	}
}

type AccountBalanceCoin struct {
	Equity           float64 `json:"equity"`
	WalletBalance    float64 `json:"walletBalance"`
	UnrealisedPnl    float64 `json:"unrealisedPnl"`
	TotalPositionIM  float64 `json:"totalPositionIM"`
	TotalPositionMM  float64 `json:"totalPositionMM"`
	AvailableBalance float64 `json:"availableBalance"`
}

func NewAccountBalanceCoinFromBybitAccountBalance(balance BybitAccountBalanceCoin) AccountBalanceCoin {
	walletBalance := balance.WalletBalance.Float64()
	totalPositionIM := balance.TotalPositionIM.Float64()
	return AccountBalanceCoin{
		Equity:           balance.Equity.Float64(),
		WalletBalance:    walletBalance,
		UnrealisedPnl:    balance.UnrealisedPnl.Float64(),
		TotalPositionIM:  totalPositionIM,
		TotalPositionMM:  balance.TotalPositionMM.Float64(),
		AvailableBalance: walletBalance - totalPositionIM,
	}
}

func NewAccountBalanceCoinFromBinanceAccountInfoAssets(assets BinanceAccountInfoAssets) AccountBalanceCoin {
	return AccountBalanceCoin{
		Equity:           assets.Equity(),
		WalletBalance:    assets.WalletBalance.Float64(),
		UnrealisedPnl:    assets.UnrealizedProfit.Float64(),
		TotalPositionIM:  assets.PositionInitialMargin.Float64(),
		TotalPositionMM:  assets.MaintMargin.Float64(),
		AvailableBalance: assets.AvailableBalance.Float64(),
	}
}

type AccountInfo struct {
	AccountType string  `json:"accountType"` // "bybit" or "binance"
	MarginMode  *string `json:"marginMode"`
}

func NewAccountInfoFromBybitAccountInfo(info BybitAccountInfo) AccountInfo {
	return AccountInfo{
		AccountType: "bybit",
		MarginMode:  &info.MarginMode,
	}
}

func NewAccountInfoFromBinanceAccountInfo() AccountInfo {
	return AccountInfo{
		AccountType: "binance",
		MarginMode:  nil,
	}
}

type PositionGenericInfo struct {
	Symbol   string  `json:"symbol"`
	Leverage float64 `json:"leverage"`
	Side     string  `json:"side"`
	Size     float64 `json:"size"`
}

func NewPositionGenericInfoFromBybitPosition(position BybitPositionGenericInfo) PositionGenericInfo {
	return PositionGenericInfo{
		Symbol:   position.Symbol,
		Leverage: position.Leverage.Float64(),
		Side:     position.Side,
		Size:     position.Size.Float64(),
	}
}

func NewPositionGenericInfoFromBinancePosition(position BinancePositionGenericInfo) PositionGenericInfo {
	return PositionGenericInfo{
		Symbol:   position.Symbol,
		Leverage: position.Leverage.Float64(),
		Side:     position.Side(),
		Size:     math.Abs(position.PositionAmt.Float64()),
	}
}

type Ticker struct {
	Symbol    string  `json:"symbol"`
	LastPrice float64 `json:"lastPrice"`
}

func NewTickerFromBybitTicker(ticker BybitTicker) Ticker {
	return Ticker{
		Symbol:    ticker.Symbol,
		LastPrice: ticker.LastPrice.Float64(),
	}
}

func NewTickerFromBinanceTickerPrice(ticker BinanceTickerPrice) Ticker {
	return Ticker{
		Symbol:    ticker.Symbol,
		LastPrice: ticker.Price.Float64(),
	}
}

type SymbolInfo struct {
	Symbol           string  `json:"symbol"`
	MinOrderQty      float64 `json:"minOrderQty"`
	MaxOrderQty      float64 `json:"maxOrderQty"`
	MaxMkOrderQty    float64 `json:"maxMkOrderQty"`
	MinNotionalValue float64 `json:"minNotionalValue"`
	QtyStep          float64 `json:"qtyStep"`
	MinPrice         float64 `json:"minPrice"`
	MaxPrice         float64 `json:"maxPrice"`
	TickSize         float64 `json:"tickSize"`
	GrowthMode       bool    `json:"growthMode"`
	MaxLeverage      int     `json:"maxLeverage"`
}

type RateLimitInfo struct {
	Used int64 `json:"used"`
	Cap  int64 `json:"cap"`
}

func NewSymbolInfoFromBybitSymbolInfo(info BybitSymbolInfo) SymbolInfo {
	return SymbolInfo{
		Symbol:           info.Symbol,
		MinOrderQty:      info.LotSizeFilter.MinOrderQty.Float64(),
		MaxOrderQty:      info.LotSizeFilter.MaxOrderQty.Float64(),
		MaxMkOrderQty:    info.LotSizeFilter.MaxMktOrderQty.Float64(),
		MinNotionalValue: info.LotSizeFilter.MinNotionalValue.Float64(),
		QtyStep:          info.LotSizeFilter.QtyStep.Float64(),
		MinPrice:         info.PriceFilter.MinPrice.Float64(),
		MaxPrice:         info.PriceFilter.MaxPrice.Float64(),
		TickSize:         info.PriceFilter.TickSize.Float64(),
		MaxLeverage:      int(info.LeverageFilter.MaxLeverage.Float64()),
	}
}

func NewSymbolInfoFromBinanceSymbolInfo(info BinanceSymbolInfo) SymbolInfo {
	return SymbolInfo{
		Symbol:           info.Symbol,
		MinOrderQty:      info.LotSizeFilter().MinOrderQty,
		MaxOrderQty:      info.LotSizeFilter().MaxOrderQty,
		MaxMkOrderQty:    info.LotSizeFilter().MaxMktOrderQty,
		MinNotionalValue: info.LotSizeFilter().MinNotionalValue,
		QtyStep:          info.LotSizeFilter().QtyStep,
		MinPrice:         info.PriceFilter().MinPrice,
		MaxPrice:         info.PriceFilter().MaxPrice,
		TickSize:         info.PriceFilter().TickSize,
	}
}

type OpenStopOrder struct {
	AccountType  string  `json:"accountType"`
	OrderID      *string `json:"orderId"`
	AlgoID       *int64  `json:"algoId"`
	Symbol       string  `json:"symbol"`
	OrderType    string  `json:"orderType"`
	TriggerPrice float64 `json:"triggerPrice"`
}

func NewOpenStopOrderFromBybitOpenOrder(order BybitOpenOrder) OpenStopOrder {
	var orderType string
	switch order.StopOrderType {
	case "TakeProfit":
		orderType = "TP"
	case "StopLoss":
		orderType = "SL"
	}

	return OpenStopOrder{
		AccountType:  "bybit",
		OrderID:      &order.OrderID,
		AlgoID:       nil,
		Symbol:       order.Symbol,
		OrderType:    orderType,
		TriggerPrice: order.TriggerPrice.Float64(),
	}
}

func NewOpenStopOrderFromBinanceOpenOrder(order BinanceOpenOrder) OpenStopOrder {
	algoID := order.AlgoID.Int64()

	var orderType string
	switch order.OrderType {
	case "TAKE_PROFIT_MARKET":
		orderType = "TP"
	case "STOP_MARKET":
		orderType = "SL"
	}

	return OpenStopOrder{
		AccountType:  "binance",
		OrderID:      nil,
		AlgoID:       &algoID,
		Symbol:       order.Symbol,
		OrderType:    orderType,
		TriggerPrice: order.TriggerPrice.Float64(),
	}
}

type PlaceOrder struct {
	OrderID string `json:"orderId"`
}

func NewPlaceOrderFromBybitPlaceOrder(order BybitPlaceOrder) PlaceOrder {
	return PlaceOrder{
		OrderID: order.OrderID,
	}
}

func NewPlaceOrderFromBinancePlaceOrder(order BinancePlaceOrder) PlaceOrder {
	return PlaceOrder{
		OrderID: strconv.FormatInt(order.OrderID.Int64(), 10),
	}
}

// ApiKeyInfo contains normalized API key information shared across exchanges.
type ApiKeyInfo struct {
	CreatedAt *int64 `json:"createdAt"`
	ExpiredAt *int64 `json:"expiredAt"`
	UserID    *int64 `json:"userId"`
}

func NewApiKeyInfoFromBybitApiKeyInfo(info BybitApiKeyInfoResponse) (*ApiKeyInfo, error) {
	// createdAt and expiredAt are RFC3339 strings, e.g. "2022-10-16T02:24:40Z".
	// A key that never expires is reported either as an empty expiredAt or as the
	// unix epoch; both must yield a nil ExpiredAt rather than a timestamp in the past.
	var createdAtUnix *int64
	if info.CreatedAt != "" {
		createdAt, err := time.ParseInLocation(time.RFC3339, info.CreatedAt, time.UTC)
		if err != nil {
			return nil, err
		}
		v := createdAt.Unix()
		createdAtUnix = &v
	}

	var expiredAtUnix *int64
	if info.ExpiredAt != "" {
		expiredAt, err := time.ParseInLocation(time.RFC3339, info.ExpiredAt, time.UTC)
		if err != nil {
			return nil, err
		}
		if v := expiredAt.Unix(); v > 0 {
			expiredAtUnix = &v
		}
	}

	return &ApiKeyInfo{
		CreatedAt: createdAtUnix,
		ExpiredAt: expiredAtUnix,
		UserID:    &info.UserID,
	}, nil
}

func NewApiKeyInfoFromBinanceApiKeyInfo(apiKeyInfo *BinanceApiKeyPermissions, userInfo *BinanceSpotAccountInfo) ApiKeyInfo {
	var createdAt *int64
	var expiredAt *int64
	var userId *int64

	// we only receive createTime as a timestamp with format: 1698645219000
	// we compute manually the expiredAt by adding 90 days to the createdAt
	// timestamps are in milliseconds
	if apiKeyInfo != nil {
		v := apiKeyInfo.CreateTime / 1000
		createdAt = &v
		if !apiKeyInfo.IPRestrict {
			// if IP restriction is not enabled, the key expires in 90 days
			exp := v + 90*24*60*60
			expiredAt = &exp
		}
	}

	if userInfo != nil {
		userId = &userInfo.UID
	}

	return ApiKeyInfo{
		CreatedAt: createdAt,
		ExpiredAt: expiredAt,
		UserID:    userId,
	}
}
