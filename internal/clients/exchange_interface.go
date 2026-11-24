package clients

import (
	"context"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

type exchangeClient interface {
	ClosePosition(account subsettings.ExchangeAccount, position Position) error
	PlaceOrder(account subsettings.ExchangeAccount, symbol, side, size string, takeProfit, stopLoss *float64, reduceOnly, closeOnTrigger bool) (*PlaceOrder, error)
	Positions(account subsettings.ExchangeAccount) ([]Position, error)
	Position(account subsettings.ExchangeAccount, symbol string) (*PositionGenericInfo, error)
	OpenStopOrders(account subsettings.ExchangeAccount) ([]OpenStopOrder, error)
	CancelStopOrders(account subsettings.ExchangeAccount, orders []OpenStopOrder) error
	SettleCoinBalance(account subsettings.ExchangeAccount) (*AccountBalanceCoin, error)
	AccountInfo(account subsettings.ExchangeAccount) (*AccountInfo, error)
	SetAccountMarginMode(account subsettings.ExchangeAccount, marginMode *string) error
	SymbolMarginType(account subsettings.ExchangeAccount, symbol string) (string, error)
	SetSymbolMarginType(account subsettings.ExchangeAccount, symbol, marginType string) error
	SetAccountPositionMode(account subsettings.ExchangeAccount, positionMode string) error
	SetLeverage(account subsettings.ExchangeAccount, symbol string, leverage int) error
	SetLeverageWebsocket(account subsettings.ExchangeAccount, symbol string, leverage int) error
	Ticker(account subsettings.ExchangeAccount, symbol string) (*Ticker, error)
	SymbolInfo(account subsettings.ExchangeAccount, symbol string, cached bool) (*SymbolInfo, error)
	// SetPositionTpsl sets the position's take profit and stop loss. A nil
	// takeProfit or stopLoss means the position must not have that leg: any
	// existing order for it is canceled.
	SetPositionTpsl(account subsettings.ExchangeAccount, symbol, positionSide string, takeProfit, stopLoss *float64) error
	// SetPositionTpslWebsocket is SetPositionTpsl over the websocket connection
	// when possible, with the same nil-cancels semantics. tpOrder and slOrder
	// are the existing orders from account state, if any.
	SetPositionTpslWebsocket(account subsettings.ExchangeAccount, symbol, positionSide string, takeProfit, stopLoss *float64, tpOrder, slOrder *OpenStopOrder) error
	HyperToExchangeSymbol(account subsettings.ExchangeAccount, hyperSymbol string) (string, error)
	GetApiKeyAndUserInformation(account subsettings.ExchangeAccount, getApiKeyInformation bool) (*ApiKeyInfo, error)
	UserRateLimit(account subsettings.ExchangeAccount) (*RateLimitInfo, error)
	InvalidateCache()
	WebsocketsEnabled(account subsettings.ExchangeAccount) bool
	ConnectWebsockets(ctx context.Context)
	CloseWebsockets()
}

// valueOrZero converts an optional price to the 0-means-unset convention used
// by the underlying exchange clients.
func valueOrZero(price *float64) float64 {
	if price == nil {
		return 0
	}
	return *price
}
