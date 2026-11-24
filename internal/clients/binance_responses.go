package clients

import (
	"encoding/json"

	"github.com/getmiroir/trading-service/internal/utils"
)

// Position represents a position from Binance
type BinancePosition struct {
	Symbol           string               `json:"symbol"`
	PositionAmt      utils.Float64String  `json:"positionAmt"`
	EntryPrice       utils.Float64String  `json:"entryPrice"`
	BreakEvenPrice   utils.Float64String  `json:"breakEvenPrice"`
	MarkPrice        utils.Float64String  `json:"markPrice"`
	UnrealizedProfit *utils.Float64String `json:"unRealizedProfit,omitempty"`
	LiquidationPrice utils.Float64String  `json:"liquidationPrice"`
	Leverage         utils.Int64String    `json:"leverage"`
	MaxNotionalValue utils.Float64String  `json:"maxNotionalValue"`
	MarginType       string               `json:"marginType"` // "cross" or "isolated"
	IsolatedMargin   utils.Float64String  `json:"isolatedMargin"`
	IsAutoAddMargin  utils.BoolString     `json:"isAutoAddMargin"`
	PositionSide     string               `json:"positionSide"` // "LONG", "SHORT", or "BOTH"
	Notional         utils.Float64String  `json:"notional"`
	IsolatedWallet   utils.Float64String  `json:"isolatedWallet"`
	UpdateTime       utils.Int64String    `json:"updateTime"`
}

// Side returns the side for a position
// "Buy" or "Sell"
func (p *BinancePosition) Side() string {
	switch p.PositionSide {
	case "LONG":
		return "Buy"
	case "SHORT":
		return "Sell"
	default:
		if p.PositionAmt.Float64() > 0 {
			return "Buy"
		} else {
			return "Sell"
		}
	}
}

// BinancePositionGenericInfo represents generic position information from Binance
type BinancePositionGenericInfo struct {
	Symbol       string              `json:"symbol"`
	Leverage     utils.Float64String `json:"leverage"`
	PositionSide string              `json:"positionSide"` // "LONG", "SHORT", or "BOTH"
	PositionAmt  utils.Float64String `json:"positionAmt"`
	MarkPrice    utils.Float64String `json:"markPrice"`
}

// Side returns the side for a position
func (p *BinancePositionGenericInfo) Side() string {
	switch p.PositionSide {
	case "LONG":
		return "Buy"
	case "SHORT":
		return "Sell"
	default:
		if p.PositionAmt.Float64() > 0 {
			return "Buy"
		} else {
			return "Sell"
		}
	}
}

// BinanceAccountInfoAssets represents asset information in an account
type BinanceAccountInfoAssets struct {
	Asset                  string              `json:"asset"`
	WalletBalance          utils.Float64String `json:"walletBalance"`
	UnrealizedProfit       utils.Float64String `json:"unrealizedProfit"`
	MarginBalance          utils.Float64String `json:"marginBalance"`
	MaintMargin            utils.Float64String `json:"maintMargin"`
	InitialMargin          utils.Float64String `json:"initialMargin"`
	PositionInitialMargin  utils.Float64String `json:"positionInitialMargin"`
	OpenOrderInitialMargin utils.Float64String `json:"openOrderInitialMargin"`
	CrossWalletBalance     utils.Float64String `json:"crossWalletBalance"`
	CrossUnPnl             utils.Float64String `json:"crossUnPnl"`
	AvailableBalance       utils.Float64String `json:"availableBalance"`
	MaxWithdrawAmount      utils.Float64String `json:"maxWithdrawAmount"`
	UpdateTime             utils.Int64String   `json:"updateTime"`
}

// BinanceAccountInfo represents account information from Binance
type BinanceAccountInfo struct {
	TotalInitialMargin          utils.Float64String        `json:"totalInitialMargin"`
	TotalMaintMargin            utils.Float64String        `json:"totalMaintMargin"`
	TotalWalletBalance          utils.Float64String        `json:"totalWalletBalance"`
	TotalUnrealizedProfit       utils.Float64String        `json:"totalUnrealizedProfit"`
	TotalMarginBalance          utils.Float64String        `json:"totalMarginBalance"`
	TotalPositionInitialMargin  utils.Float64String        `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin utils.Float64String        `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     utils.Float64String        `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             utils.Float64String        `json:"totalCrossUnPnl"`
	AvailableBalance            utils.Float64String        `json:"availableBalance"`
	MaxWithdrawAmount           utils.Float64String        `json:"maxWithdrawAmount"`
	Assets                      []BinanceAccountInfoAssets `json:"assets"`
}

// BinanceAccountInfoAssets Equity returns the equity for the account
func (b *BinanceAccountInfoAssets) Equity() float64 {
	return b.WalletBalance.Float64() + b.UnrealizedProfit.Float64()
}

// BinancePriceFilter represents price filter constraints
type BinancePriceFilter struct {
	FilterType string              `json:"filterType"` // "PRICE_FILTER"
	MinPrice   utils.Float64String `json:"minPrice"`
	MaxPrice   utils.Float64String `json:"maxPrice"`
	TickSize   utils.Float64String `json:"tickSize"`
}

// BinanceLotSizeFilter represents lot size filter constraints
type BinanceLotSizeFilter struct {
	FilterType string              `json:"filterType"` // "LOT_SIZE"
	MinQty     utils.Float64String `json:"minQty"`
	MaxQty     utils.Float64String `json:"maxQty"`
	StepSize   utils.Float64String `json:"stepSize"`
}

// BinanceMarketLotSizeFilter represents market lot size filter constraints
type BinanceMarketLotSizeFilter struct {
	FilterType string              `json:"filterType"` // "MARKET_LOT_SIZE"
	MinQty     utils.Float64String `json:"minQty"`
	MaxQty     utils.Float64String `json:"maxQty"`
	StepSize   utils.Float64String `json:"stepSize"`
}

// BinanceMinNotionalFilter represents min notional filter constraints
type BinanceMinNotionalFilter struct {
	FilterType string              `json:"filterType"` // "MIN_NOTIONAL"
	Notional   utils.Float64String `json:"notional"`
}

// BinancePercentPriceFilter represents percent price filter constraints
type BinancePercentPriceFilter struct {
	FilterType        string              `json:"filterType"` // "PERCENT_PRICE"
	MultiplierUp      utils.Float64String `json:"multiplierUp"`
	MultiplierDown    utils.Float64String `json:"multiplierDown"`
	MultiplierDecimal int                 `json:"multiplierDecimal"`
}

// BinanceMaxNumOrdersFilter represents max number of orders filter
type BinanceMaxNumOrdersFilter struct {
	FilterType string `json:"filterType"` // "MAX_NUM_ORDERS"
	Limit      int    `json:"limit"`
}

// BinanceSymbolInfoPriceFilter is a helper to match Bybit's interface
type BinanceSymbolInfoPriceFilter struct {
	MinPrice float64
	MaxPrice float64
	TickSize float64
}

// BinanceSymbolInfoLotSizeFilter is a helper to match Bybit's interface
type BinanceSymbolInfoLotSizeFilter struct {
	MinOrderQty      float64
	MaxOrderQty      float64
	MaxMktOrderQty   float64
	QtyStep          float64
	MinNotionalValue float64
}

// BinanceSymbolInfo represents symbol information from Binance
type BinanceSymbolInfo struct {
	Symbol                string                   `json:"symbol"`
	Pair                  string                   `json:"pair"`
	ContractType          string                   `json:"contractType"`
	DeliveryDate          utils.Int64String        `json:"deliveryDate"`
	OnboardDate           utils.Int64String        `json:"onboardDate"`
	Status                string                   `json:"status"`
	MaintMarginPercent    utils.Float64String      `json:"maintMarginPercent"`
	RequiredMarginPercent utils.Float64String      `json:"requiredMarginPercent"`
	BaseAsset             string                   `json:"baseAsset"`
	QuoteAsset            string                   `json:"quoteAsset"`
	MarginAsset           string                   `json:"marginAsset"`
	PricePrecision        int                      `json:"pricePrecision"`
	QuantityPrecision     int                      `json:"quantityPrecision"`
	BaseAssetPrecision    int                      `json:"baseAssetPrecision"`
	QuotePrecision        int                      `json:"quotePrecision"`
	UnderlyingType        string                   `json:"underlyingType"`
	UnderlyingSubType     []string                 `json:"underlyingSubType"`
	SettlePlan            *int                     `json:"settlePlan,omitempty"`
	TriggerProtect        utils.Float64String      `json:"triggerProtect"`
	LiquidationFee        utils.Float64String      `json:"liquidationFee"`
	MarketTakeBound       utils.Float64String      `json:"marketTakeBound"`
	Filters               []map[string]interface{} `json:"filters"`
	OrderType             []string                 `json:"orderType,omitempty"`
	TimeInForce           []string                 `json:"timeInForce"`

	// Cached parsed filters
	parsedPriceFilter   *BinancePriceFilter
	parsedLotSizeFilter *BinanceLotSizeFilter
	parsedMarketLotSize *BinanceMarketLotSizeFilter
	parsedMinNotional   *BinanceMinNotionalFilter
}

// parseFilters parses the filters array into typed structs
func (s *BinanceSymbolInfo) parseFilters() {
	for _, filterMap := range s.Filters {
		filterBytes, _ := json.Marshal(filterMap)
		filterType, _ := filterMap["filterType"].(string)

		switch filterType {
		case "PRICE_FILTER":
			var f BinancePriceFilter
			if json.Unmarshal(filterBytes, &f) == nil {
				s.parsedPriceFilter = &f
			}
		case "LOT_SIZE":
			var f BinanceLotSizeFilter
			if json.Unmarshal(filterBytes, &f) == nil {
				s.parsedLotSizeFilter = &f
			}
		case "MARKET_LOT_SIZE":
			var f BinanceMarketLotSizeFilter
			if json.Unmarshal(filterBytes, &f) == nil {
				s.parsedMarketLotSize = &f
			}
		case "MIN_NOTIONAL":
			var f BinanceMinNotionalFilter
			if json.Unmarshal(filterBytes, &f) == nil {
				s.parsedMinNotional = &f
			}
		}
	}
}

// PriceFilter returns the price filter for the symbol
func (s *BinanceSymbolInfo) PriceFilter() *BinanceSymbolInfoPriceFilter {
	if s.parsedPriceFilter == nil {
		s.parseFilters()
	}
	if s.parsedPriceFilter != nil {
		return &BinanceSymbolInfoPriceFilter{
			MinPrice: s.parsedPriceFilter.MinPrice.Float64(),
			MaxPrice: s.parsedPriceFilter.MaxPrice.Float64(),
			TickSize: s.parsedPriceFilter.TickSize.Float64(),
		}
	}
	return nil
}

// LotSizeFilter returns the lot size filter for the symbol
func (s *BinanceSymbolInfo) LotSizeFilter() *BinanceSymbolInfoLotSizeFilter {
	if s.parsedLotSizeFilter == nil || s.parsedMarketLotSize == nil || s.parsedMinNotional == nil {
		s.parseFilters()
	}
	if s.parsedLotSizeFilter != nil && s.parsedMarketLotSize != nil && s.parsedMinNotional != nil {
		return &BinanceSymbolInfoLotSizeFilter{
			MinOrderQty:      s.parsedMarketLotSize.MinQty.Float64(),
			MaxOrderQty:      s.parsedMarketLotSize.MaxQty.Float64(),
			MaxMktOrderQty:   s.parsedMarketLotSize.MaxQty.Float64(),
			QtyStep:          s.parsedLotSizeFilter.StepSize.Float64(),
			MinNotionalValue: s.parsedMinNotional.Notional.Float64(),
		}
	}
	return nil
}

// BinanceExchangeInfo represents exchange information from Binance
type BinanceExchangeInfo struct {
	Symbols []BinanceSymbolInfo `json:"symbols"`
}

// BinanceLeverageBracketEntry represents a single notional/leverage tier for a symbol.
type BinanceLeverageBracketEntry struct {
	Bracket          int     `json:"bracket"`
	InitialLeverage  int     `json:"initialLeverage"`
	NotionalCap      float64 `json:"notionalCap"`
	NotionalFloor    float64 `json:"notionalFloor"`
	MaintMarginRatio float64 `json:"maintMarginRatio"`
	Cum              float64 `json:"cum"`
}

// BinanceLeverageBracket represents the leverage brackets for a single symbol
type BinanceLeverageBracket struct {
	Symbol       string                        `json:"symbol"`
	NotionalCoef float64                       `json:"notionalCoef"`
	Brackets     []BinanceLeverageBracketEntry `json:"brackets"`
}

// MaxLeverage returns the highest initialLeverage across the symbol's brackets,
// i.e. the exchange's absolute leverage ceiling for the symbol (bracket 1, the
// lowest notional tier). Returns 0 when no brackets are present.
func (b *BinanceLeverageBracket) MaxLeverage() int {
	maxLeverage := 0
	for _, bracket := range b.Brackets {
		if bracket.InitialLeverage > maxLeverage {
			maxLeverage = bracket.InitialLeverage
		}
	}
	return maxLeverage
}

// BinanceSymbolConfig represents symbol configuration from Binance
type BinanceSymbolConfig struct {
	Symbol           string              `json:"symbol"`
	MarginType       string              `json:"marginType"` // "ISOLATED" or "CROSSED"
	IsAutoAddMargin  utils.BoolString    `json:"isAutoAddMargin"`
	Leverage         int                 `json:"leverage"`
	MaxNotionalValue utils.Float64String `json:"maxNotionalValue"`
}

// BinanceTickerPrice represents ticker price information from Binance
type BinanceTickerPrice struct {
	Symbol string              `json:"symbol"`
	Price  utils.Float64String `json:"price"`
	Time   utils.Int64String   `json:"time"`
}

// BinancePlaceOrder represents a placed order response from Binance
type BinancePlaceOrder struct {
	OrderID utils.Int64String `json:"orderId"`
}

// BinancePlaceAlgoOrder represents a placed algo order response from Binance
type BinancePlaceAlgoOrder struct {
	AlgoID utils.Int64String `json:"algoId"`
}

// BinanceOpenOrder represents an open order from Binance
type BinanceOpenOrder struct {
	AlgoID                  utils.Int64String    `json:"algoId"`
	ClientAlgoID            string               `json:"clientAlgoId"`
	AlgoType                string               `json:"algoType"`
	OrderType               string               `json:"orderType"`
	Symbol                  string               `json:"symbol"`
	Side                    string               `json:"side"`         // "BUY" or "SELL"
	PositionSide            string               `json:"positionSide"` // "LONG", "SHORT", or "BOTH"
	TimeInForce             string               `json:"timeInForce"`
	Quantity                utils.Float64String  `json:"quantity"`
	AlgoStatus              string               `json:"algoStatus"`
	ActualOrderID           string               `json:"actualOrderId"`
	ActualPrice             utils.Float64String  `json:"actualPrice"`
	TriggerPrice            utils.Float64String  `json:"triggerPrice"`
	Price                   utils.Float64String  `json:"price"`
	IcebergQuantity         *utils.Float64String `json:"icebergQuantity,omitempty"`
	TpTriggerPrice          *utils.Float64String `json:"tpTriggerPrice,omitempty"`
	TpPrice                 *utils.Float64String `json:"tpPrice,omitempty"`
	SlTriggerPrice          *utils.Float64String `json:"slTriggerPrice,omitempty"`
	SlPrice                 *utils.Float64String `json:"slPrice,omitempty"`
	TpOrderType             string               `json:"tpOrderType"`
	SelfTradePreventionMode string               `json:"selfTradePreventionMode"`
	WorkingType             string               `json:"workingType"`
	PriceMatch              string               `json:"priceMatch"`
	ClosePosition           utils.BoolString     `json:"closePosition"`
	PriceProtect            utils.BoolString     `json:"priceProtect"`
	ReduceOnly              utils.BoolString     `json:"reduceOnly"`
	CreateTime              utils.Int64String    `json:"createTime"`
	UpdateTime              utils.Int64String    `json:"updateTime"`
	TriggerTime             utils.Int64String    `json:"triggerTime"`
	GoodTillDate            utils.Int64String    `json:"goodTillDate"`
}

// BinanceWebsocketRateLimit represents rate limit information in websocket responses
type BinanceWebsocketRateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
	Count         int    `json:"count"`
}

// BinanceWebsocketOrderResult represents the order result in websocket responses
type BinanceWebsocketOrderResult struct {
	OrderID                 int64               `json:"orderId"`
	Symbol                  string              `json:"symbol"`
	Status                  string              `json:"status"`
	ClientOrderID           string              `json:"clientOrderId"`
	Price                   utils.Float64String `json:"price"`
	AvgPrice                utils.Float64String `json:"avgPrice"`
	OrigQty                 utils.Float64String `json:"origQty"`
	ExecutedQty             utils.Float64String `json:"executedQty"`
	CumQty                  utils.Float64String `json:"cumQty"`
	CumQuote                utils.Float64String `json:"cumQuote"`
	TimeInForce             string              `json:"timeInForce"`
	Type                    string              `json:"type"`
	ReduceOnly              utils.BoolString    `json:"reduceOnly"`
	ClosePosition           utils.BoolString    `json:"closePosition"`
	Side                    string              `json:"side"`
	PositionSide            string              `json:"positionSide"`
	StopPrice               utils.Float64String `json:"stopPrice"`
	WorkingType             string              `json:"workingType"`
	PriceProtect            utils.BoolString    `json:"priceProtect"`
	OrigType                string              `json:"origType"`
	PriceMatch              string              `json:"priceMatch"`
	SelfTradePreventionMode string              `json:"selfTradePreventionMode"`
	GoodTillDate            int64               `json:"goodTillDate"`
	UpdateTime              int64               `json:"updateTime"`
}

// BinanceWebsocketResponse represents a websocket response from Binance
type BinanceWebsocketResponse struct {
	ID         string                      `json:"id"`
	Status     int                         `json:"status"`
	Result     BinanceWebsocketOrderResult `json:"result"`
	RateLimits []BinanceWebsocketRateLimit `json:"rateLimits"`
	Error      *BinanceWebsocketError      `json:"error,omitempty"`
}

// BinanceWebsocketError represents an error in websocket responses
type BinanceWebsocketError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// BinancePositionModeResponse represents the position mode response from Binance
type BinancePositionModeResponse struct {
	DualSidePosition utils.BoolString `json:"dualSidePosition"`
}

// EmptyResponse represents an empty response
type BinanceEmptyResponse struct{}

// BinanceSpotCommissionRates represents the commission rates in a spot account info response
type BinanceSpotCommissionRates struct {
	Maker  utils.Float64String `json:"maker"`
	Taker  utils.Float64String `json:"taker"`
	Buyer  utils.Float64String `json:"buyer"`
	Seller utils.Float64String `json:"seller"`
}

// BinanceSpotBalance represents a single asset balance in a spot account
type BinanceSpotBalance struct {
	Asset  string              `json:"asset"`
	Free   utils.Float64String `json:"free"`
	Locked utils.Float64String `json:"locked"`
}

// BinanceSpotAccountInfo represents the response from GET /api/v3/account
type BinanceSpotAccountInfo struct {
	MakerCommission            int                        `json:"makerCommission"`
	TakerCommission            int                        `json:"takerCommission"`
	BuyerCommission            int                        `json:"buyerCommission"`
	SellerCommission           int                        `json:"sellerCommission"`
	CommissionRates            BinanceSpotCommissionRates `json:"commissionRates"`
	CanTrade                   bool                       `json:"canTrade"`
	CanWithdraw                bool                       `json:"canWithdraw"`
	CanDeposit                 bool                       `json:"canDeposit"`
	Brokered                   bool                       `json:"brokered"`
	RequireSelfTradePrevention bool                       `json:"requireSelfTradePrevention"`
	PreventSor                 bool                       `json:"preventSor"`
	UpdateTime                 int64                      `json:"updateTime"`
	AccountType                string                     `json:"accountType"`
	Balances                   []BinanceSpotBalance       `json:"balances"`
	Permissions                []string                   `json:"permissions"`
	UID                        int64                      `json:"uid"`
}

// BinanceApiKeyPermissions represents the API key permission response from Binance
type BinanceApiKeyPermissions struct {
	IPRestrict                   bool  `json:"ipRestrict"`
	CreateTime                   int64 `json:"createTime"`
	EnableReading                bool  `json:"enableReading"`
	EnableWithdrawals            bool  `json:"enableWithdrawals"`
	EnableInternalTransfer       bool  `json:"enableInternalTransfer"`
	EnableMargin                 bool  `json:"enableMargin"`
	EnableFutures                bool  `json:"enableFutures"`
	PermitsUniversalTransfer     bool  `json:"permitsUniversalTransfer"`
	EnableVanillaOptions         bool  `json:"enableVanillaOptions"`
	EnableFixApiTrade            bool  `json:"enableFixApiTrade"`
	EnableFixReadOnly            bool  `json:"enableFixReadOnly"`
	EnableSpotAndMarginTrading   bool  `json:"enableSpotAndMarginTrading"`
	EnablePortfolioMarginTrading bool  `json:"enablePortfolioMarginTrading"`
}
