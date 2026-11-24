package clients

import (
	"github.com/getmiroir/trading-service/internal/utils"
)

// HyperliquidLeverage represents the leverage object from Hyperliquid
type HyperliquidLeverage struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// HyperliquidPosition represents a position from Hyperliquid
type HyperliquidPosition struct {
	Coin          string              `json:"coin"`
	EntryPx       utils.Float64String `json:"entryPx"`
	PositionValue utils.Float64String `json:"positionValue"`
	Szi           utils.Float64String `json:"szi"` // Size (positive for long, negative for short)
	UnrealizedPnl utils.Float64String `json:"unrealizedPnl"`
	Leverage      HyperliquidLeverage `json:"leverage"`
}

// AssetPosition represents an asset position wrapper
type AssetPosition struct {
	Position HyperliquidPosition `json:"position"`
}

// MarginSummary represents margin summary information
type MarginSummary struct {
	AccountValue    utils.Float64String `json:"accountValue"`
	TotalMarginUsed utils.Float64String `json:"totalMarginUsed"`
}

// CrossMarginSummary represents cross margin summary information
type CrossMarginSummary struct {
	AccountValue    utils.Float64String `json:"accountValue"`
	TotalMarginUsed utils.Float64String `json:"totalMarginUsed"`
}

// PositionsResponse represents the positions response from Hyperliquid
type PositionsResponse struct {
	AssetPositions     []AssetPosition    `json:"assetPositions"`
	MarginSummary      MarginSummary      `json:"marginSummary"`
	CrossMarginSummary CrossMarginSummary `json:"crossMarginSummary"`
}

// UserAbstractionMode is the Hyperliquid user abstraction / margin mode for a wallet.
type UserAbstractionMode string

const (
	UserAbstractionUnifiedAccount  UserAbstractionMode = "unifiedAccount"
	UserAbstractionPortfolioMargin UserAbstractionMode = "portfolioMargin"
	UserAbstractionDisabled        UserAbstractionMode = "disabled"
	UserAbstractionDefault         UserAbstractionMode = "default"
	UserAbstractionDexAbstraction  UserAbstractionMode = "dexAbstraction"
)
