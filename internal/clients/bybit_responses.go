package clients

import "github.com/getmiroir/trading-service/internal/utils"

// PaginatedResponse represents a paginated API response from Bybit
type BybitPaginatedResponse struct {
	Results        []map[string]interface{} `json:"list"`
	NextPageCursor *string                  `json:"nextPageCursor,omitempty"`
}

// Position represents a position from Bybit
type BybitPosition struct {
	Symbol          string              `json:"symbol"`
	PositionIdx     int                 `json:"positionIdx"`
	Side            string              `json:"side"` // "Buy" or "Sell"
	Size            utils.Float64String `json:"size"`
	PositionValue   utils.Float64String `json:"positionValue"`
	PositionBalance utils.Float64String `json:"positionBalance"`
	Leverage        utils.Float64String `json:"leverage"`
	AvgPrice        utils.Float64String `json:"avgPrice"`
	MarkPrice       utils.Float64String `json:"markPrice"`
	LiqPrice        utils.Float64String `json:"liqPrice"`   // Can be empty string or float
	TakeProfit      string              `json:"takeProfit"` // Can be empty string or float
	StopLoss        string              `json:"stopLoss"`   // Can be empty string or float
	TrailingStop    utils.Float64String `json:"trailingStop"`
	UnrealisedPnl   utils.Float64String `json:"unrealisedPnl"`
	PositionIM      string              `json:"positionIM"` // Can be empty string or float
	PositionMM      string              `json:"positionMM"` // Can be empty string or float
	Seq             int                 `json:"seq"`
}

// PositionGenericInfo represents generic position information
type BybitPositionGenericInfo struct {
	Symbol         string              `json:"symbol"`
	PositionIdx    int                 `json:"positionIdx"`
	PositionStatus string              `json:"positionStatus"` // "Normal", "Liq", "Adl", or ""
	Leverage       utils.Float64String `json:"leverage"`
	Side           string              `json:"side"`
	Size           utils.Float64String `json:"size"`
}

// AccountBalanceCoin represents a coin balance in an account
type BybitAccountBalanceCoin struct {
	Coin            string              `json:"coin"`
	UsdValue        utils.Float64String `json:"usdValue"`
	WalletBalance   utils.Float64String `json:"walletBalance"`
	Equity          utils.Float64String `json:"equity"`
	UnrealisedPnl   utils.Float64String `json:"unrealisedPnl"`
	TotalPositionMM utils.Float64String `json:"totalPositionMM"`
	TotalPositionIM utils.Float64String `json:"totalPositionIM"`
}

// AccountBalance represents account balance information
type BybitAccountBalance struct {
	AccountType        string                    `json:"accountType"`
	TotalWalletBalance utils.Float64String       `json:"totalWalletBalance"`
	TotalEquity        utils.Float64String       `json:"totalEquity"`
	Coin               []BybitAccountBalanceCoin `json:"coin"`
}

// Ticker represents ticker information
type BybitTicker struct {
	Symbol            string              `json:"symbol"`
	LastPrice         utils.Float64String `json:"lastPrice"`
	IndexPrice        utils.Float64String `json:"indexPrice"`
	MarkPrice         utils.Float64String `json:"markPrice"`
	OpenInterest      utils.Float64String `json:"openInterest"`
	OpenInterestValue utils.Float64String `json:"openInterestValue"`
	FundingRate       utils.Float64String `json:"fundingRate"`
	Ask1Size          utils.Float64String `json:"ask1Size"`
	Bid1Size          utils.Float64String `json:"bid1Size"`
	Ask1Price         utils.Float64String `json:"ask1Price"`
	Bid1Price         utils.Float64String `json:"bid1Price"`
}

// SymbolInfoPriceFilter represents price filter constraints
type BybitSymbolInfoPriceFilter struct {
	MinPrice utils.Float64String `json:"minPrice"`
	MaxPrice utils.Float64String `json:"maxPrice"`
	TickSize utils.Float64String `json:"tickSize"`
}

// SymbolInfoLotSizeFilter represents lot size filter constraints
type BybitSymbolInfoLotSizeFilter struct {
	MaxMktOrderQty   utils.Float64String `json:"maxMktOrderQty"`
	MaxOrderQty      utils.Float64String `json:"maxOrderQty"`
	MinOrderQty      utils.Float64String `json:"minOrderQty"`
	QtyStep          utils.Float64String `json:"qtyStep"`
	MinNotionalValue utils.Float64String `json:"minNotionalValue"`
}

// SymbolInfoLeverageFilter represents leverage constraints for a symbol
type BybitSymbolInfoLeverageFilter struct {
	MinLeverage  utils.Float64String `json:"minLeverage"`
	MaxLeverage  utils.Float64String `json:"maxLeverage"`
	LeverageStep utils.Float64String `json:"leverageStep"`
}

// SymbolInfo represents symbol information
type BybitSymbolInfo struct {
	Symbol           string                        `json:"symbol"`
	ContractType     string                        `json:"contractType"`
	Status           string                        `json:"status"`
	BaseCoin         string                        `json:"baseCoin"`
	QuoteCoin        string                        `json:"quoteCoin"`
	SettleCoin       string                        `json:"settleCoin"`
	LaunchTime       utils.Int64String             `json:"launchTime"`
	PriceFilter      BybitSymbolInfoPriceFilter    `json:"priceFilter"`
	LotSizeFilter    BybitSymbolInfoLotSizeFilter  `json:"lotSizeFilter"`
	LeverageFilter   BybitSymbolInfoLeverageFilter `json:"leverageFilter"`
	FundingInterval  utils.Int64String             `json:"fundingInterval"`
	UpperFundingRate utils.Float64String           `json:"upperFundingRate"`
	LowerFundingRate utils.Float64String           `json:"lowerFundingRate"`
}

// AccountInfo represents account information
type BybitAccountInfo struct {
	MarginMode string `json:"marginMode"` // "ISOLATED_MARGIN", "REGULAR_MARGIN", "PORTFOLIO_MARGIN"
}

// PlaceOrder represents a placed order response
type BybitPlaceOrder struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// BybitMemberType represents a member's UID and wallet types
type BybitMemberType struct {
	UID         string   `json:"uid"`
	AccountType []string `json:"accountType"`
}

// BybitMemberTypeResponse represents the response from get-member-type
type BybitMemberTypeResponse struct {
	Accounts []BybitMemberType `json:"accounts"`
}

// BybitApiKeyPermissions represents the permissions associated with an API key
type BybitApiKeyPermissions struct {
	ContractTrade     []string `json:"ContractTrade"`
	Spot              []string `json:"Spot"`
	Wallet            []string `json:"Wallet"`
	Options           []string `json:"Options"`
	Derivatives       []string `json:"Derivatives"`
	Exchange          []string `json:"Exchange"`
	Earn              []string `json:"Earn"`
	FiatP2P           []string `json:"FiatP2P"`
	FiatBybitPay      []string `json:"FiatBybitPay"`
	FiatConvertBroker []string `json:"FiatConvertBroker"`
	BlockTrade        []string `json:"BlockTrade"`
	Affiliate         []string `json:"Affiliate"`
	NFT               []string `json:"NFT"`
	CopyTrading       []string `json:"CopyTrading"`
}

// BybitApiKeyInfoResponse represents the response from /user/query-api
type BybitApiKeyInfoResponse struct {
	ID            string                 `json:"id"`
	Note          string                 `json:"note"`
	ApiKey        string                 `json:"apiKey"`
	ReadOnly      int                    `json:"readOnly"`
	Secret        string                 `json:"secret"`
	Permissions   BybitApiKeyPermissions `json:"permissions"`
	IPs           []string               `json:"ips"`
	Type          int                    `json:"type"`
	DeadlineDay   int                    `json:"deadlineDay"`
	ExpiredAt     string                 `json:"expiredAt"`
	CreatedAt     string                 `json:"createdAt"`
	Unified       int                    `json:"unified"`
	UTA           int                    `json:"uta"`
	UserID        int64                  `json:"userID"`
	InviterID     int                    `json:"inviterID"`
	VipLevel      string                 `json:"vipLevel"`
	MktMakerLevel string                 `json:"mktMakerLevel"`
	AffiliateID   int64                  `json:"affiliateID"`
	RsaPublicKey  string                 `json:"rsaPublicKey"`
	IsMaster      bool                   `json:"isMaster"`
	ParentUid     string                 `json:"parentUid"`
	KycLevel      string                 `json:"kycLevel"`
	KycRegion     string                 `json:"kycRegion"`
}

// EmptyResponse represents an empty response
type BybitEmptyResponse struct{}

// OpenOrder represents an open or closed order from Bybit
type BybitOpenOrder struct {
	OrderID           string              `json:"orderId"`
	OrderLinkID       string              `json:"orderLinkId"`
	ParentOrderLinkID string              `json:"parentOrderLinkId"`
	BlockTradeID      string              `json:"blockTradeId"`
	Symbol            string              `json:"symbol"`
	Price             utils.Float64String `json:"price"`
	Qty               utils.Float64String `json:"qty"`
	Side              string              `json:"side"`
	IsLeverage        string              `json:"isLeverage"`
	PositionIdx       int                 `json:"positionIdx"`
	OrderStatus       string              `json:"orderStatus"`
	CreateType        string              `json:"createType"`
	CancelType        string              `json:"cancelType"`
	RejectReason      string              `json:"rejectReason"`
	AvgPrice          utils.Float64String `json:"avgPrice"`
	LeavesQty         utils.Float64String `json:"leavesQty"`
	LeavesValue       utils.Float64String `json:"leavesValue"`
	CumExecQty        utils.Float64String `json:"cumExecQty"`
	CumExecValue      utils.Float64String `json:"cumExecValue"`
	CumExecFee        utils.Float64String `json:"cumExecFee"`
	TimeInForce       string              `json:"timeInForce"`
	OrderType         string              `json:"orderType"`
	StopOrderType     string              `json:"stopOrderType"`
	OrderIv           string              `json:"orderIv"`
	MarketUnit        string              `json:"marketUnit"`
	TriggerPrice      utils.Float64String `json:"triggerPrice"`
	TakeProfit        utils.Float64String `json:"takeProfit"`
	StopLoss          utils.Float64String `json:"stopLoss"`
	TpslMode          string              `json:"tpslMode"`
	OcoTriggerBy      string              `json:"ocoTriggerBy"`
	TpLimitPrice      utils.Float64String `json:"tpLimitPrice"`
	SlLimitPrice      utils.Float64String `json:"slLimitPrice"`
	TpTriggerBy       string              `json:"tpTriggerBy"`
	SlTriggerBy       string              `json:"slTriggerBy"`
	TriggerDirection  int                 `json:"triggerDirection"`
	TriggerBy         string              `json:"triggerBy"`
	LastPriceOnCreate string              `json:"lastPriceOnCreated"`
	BasePrice         string              `json:"basePrice"`
	ReduceOnly        bool                `json:"reduceOnly"`
	CloseOnTrigger    bool                `json:"closeOnTrigger"`
	PlaceType         string              `json:"placeType"`
	SmpType           string              `json:"smpType"`
	SmpGroup          utils.Int64String   `json:"smpGroup"`
	SmpOrderID        string              `json:"smpOrderId"`
	CreatedTime       utils.Int64String   `json:"createdTime"`
	UpdatedTime       utils.Int64String   `json:"updatedTime"`
	CumFeeDetail      map[string]string   `json:"cumFeeDetail"`
}

// BybitWebsocketResponse
type BybitWebsocketResponse struct {
	RetCode int         `json:"retCode"`
	RetMsg  string      `json:"retMsg"`
	Op      string      `json:"op"`
	ConnID  string      `json:"connId"`
	ReqID   string      `json:"reqId"`
	Args    []string    `json:"args"`
	Data    interface{} `json:"data"`
}
