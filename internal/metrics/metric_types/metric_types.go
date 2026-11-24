package metric_types

import "github.com/getmiroir/trading-service/internal"

// LoopMetrics contains all metrics data for a single loop iteration
type LoopMetrics struct {
	ID             string           `json:"id"`
	StartTimestamp int64            `json:"startTimestamp"`
	EndTimestamp   int64            `json:"endTimestamp"`
	Version        string           `json:"version,omitempty"`
	OS             string           `json:"os,omitempty"`
	Platform       string           `json:"platform,omitempty"`
	Accounts       []AccountMetrics `json:"accounts,omitempty"`
	Errors         []LoopError      `json:"errors,omitempty"`
}

// LoopError represents an error that occurred during the loop
type LoopError struct {
	Exception string                 `json:"exception"`
	ErrorCode internal.SyncErrorCode `json:"errorCode"`
}

// AccountMetrics contains metrics for a single account-target pair
type AccountMetrics struct {
	StartTimestamp           int64           `json:"startTimestamp"`
	EndTimestamp             int64           `json:"endTimestamp"`
	Account                  string          `json:"account"`
	Target                   string          `json:"target"`
	Exchange                 string          `json:"exchange"`
	TargetExchange           string          `json:"targetExchange,omitempty"` // always Hyperliquid for now
	IsDemo                   bool            `json:"isDemo"`
	ExchangeUserID           *string         `json:"exchangeUserId"`
	APIKeyExpiredAt          *int64          `json:"apiKeyExpiredAt"`
	Equity                   *float64        `json:"equity,omitempty"`
	Balance                  *float64        `json:"balance,omitempty"`
	PnL                      *float64        `json:"pnl,omitempty"`
	InitialMargin            *float64        `json:"initialMargin,omitempty"`
	MaintenanceMargin        *float64        `json:"maintenanceMargin,omitempty"`
	AvailableMarginRatio     *float64        `json:"availableMarginRatio,omitempty"`
	MinimumEquity            float64         `json:"minimumEquity"`
	Normalization            *float64        `json:"normalization,omitempty"`
	TargetEquity             *float64        `json:"targetEquity,omitempty"`
	TrackingRatio            *float64        `json:"trackingRatio,omitempty"`
	SettleCoin               string          `json:"settleCoin,omitempty"`
	HyperliquidRateLimitUsed *int64          `json:"hyperliquidRateLimitUsed,omitempty"`
	HyperliquidRateLimitCap  *int64          `json:"hyperliquidRateLimitCap,omitempty"`
	Symbols                  []SymbolMetrics `json:"symbols,omitempty"`
	Errors                   []AccountError  `json:"errors,omitempty"`
}

// AccountError represents an error in account sync
type AccountError struct {
	Exception string                 `json:"exception"`
	ErrorCode internal.SyncErrorCode `json:"errorCode"`
}

// SymbolMetrics contains metrics for a single symbol
type SymbolMetrics struct {
	StartTimestamp int64          `json:"startTimestamp"`
	EndTimestamp   int64          `json:"endTimestamp"`
	Symbol         string         `json:"symbol"`                 // symbol in user exchange
	TargetSymbol   string         `json:"targetSymbol,omitempty"` // symbol in target exchange
	Size           float64        `json:"size"`
	Value          float64        `json:"value"`
	Price          float64        `json:"price"`
	EntryPrice     float64        `json:"entryPrice"`
	TakeProfit     float64        `json:"takeProfit"`
	StopLoss       float64        `json:"stopLoss"`
	Liquidation    float64        `json:"liquidation"`
	PnL            float64        `json:"pnl"`
	Leverage       float64        `json:"leverage"`
	TargetSize     float64        `json:"targetSize"`
	SettleCoin     string         `json:"settleCoin,omitempty"`
	Trades         []TradeMetrics `json:"trades,omitempty"`
	Errors         []SymbolError  `json:"errors,omitempty"`
}

// TradeMetrics contains metrics for a trade
type TradeMetrics struct {
	StartTimestamp int64   `json:"startTimestamp"`
	EndTimestamp   int64   `json:"endTimestamp"`
	Side           string  `json:"side"`
	Value          float64 `json:"value"`
	Size           float64 `json:"size"`
}

// SymbolError represents an error in symbol sync
type SymbolError struct {
	Exception string                 `json:"exception"`
	ErrorCode internal.SyncErrorCode `json:"errorCode"`
}
