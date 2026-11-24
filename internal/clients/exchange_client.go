package clients

import (
	"context"
	"fmt"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

type Exchange struct {
	clients map[string]exchangeClient
	mapper  *SymbolMapper
}

func NewExchange(cfg subsettings.Exchange, logApiResponses bool, mapper *SymbolMapper) *Exchange {
	bybit := NewBybit(cfg.Bybit, logApiResponses)
	binance := NewBinance(cfg.Binance, logApiResponses)
	hl := NewHyperliquidExchange(cfg.Hyperliquid, logApiResponses)
	return &Exchange{
		clients: map[string]exchangeClient{
			"bybit":       &bybitExchangeAdapter{client: bybit, mapper: mapper},
			"binance":     &binanceExchangeAdapter{client: binance, mapper: mapper},
			"hyperliquid": &hyperliquidExchangeAdapter{client: hl, apiKeyCache: make(map[string]*HLExtraAgent)},
		},
		mapper: mapper,
	}
}

func (e *Exchange) InvalidateCache() {
	for _, client := range e.clients {
		client.InvalidateCache()
	}
	if e.mapper != nil {
		e.mapper.InvalidateCache()
	}
}

func (e *Exchange) ConnectWebsockets(ctx context.Context) {
	for _, client := range e.clients {
		client.ConnectWebsockets(ctx)
	}
}

func (e *Exchange) CloseWebsockets() {
	for _, client := range e.clients {
		client.CloseWebsockets()
	}
}

func (e *Exchange) client(account subsettings.ExchangeAccount) (exchangeClient, error) {
	client, ok := e.clients[*account.ExchangeType]
	if !ok {
		return nil, fmt.Errorf("invalid exchange type: %s", *account.ExchangeType)
	}
	return client, nil
}

func (e *Exchange) ClosePosition(exchangeAccount subsettings.ExchangeAccount, position Position) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.ClosePosition(exchangeAccount, position)
}

func (e *Exchange) PlaceOrder(
	exchangeAccount subsettings.ExchangeAccount,
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	reduceOnly bool,
	closeOnTrigger bool,
) (*PlaceOrder, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.PlaceOrder(exchangeAccount, symbol, side, size, takeProfit, stopLoss, reduceOnly, closeOnTrigger)
}

func (e *Exchange) Positions(exchangeAccount subsettings.ExchangeAccount) ([]Position, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.Positions(exchangeAccount)
}

func (e *Exchange) Position(exchangeAccount subsettings.ExchangeAccount, symbol string) (*PositionGenericInfo, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.Position(exchangeAccount, symbol)
}

func (e *Exchange) OpenStopOrders(exchangeAccount subsettings.ExchangeAccount) ([]OpenStopOrder, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.OpenStopOrders(exchangeAccount)
}

func (e *Exchange) SettleCoinBalance(exchangeAccount subsettings.ExchangeAccount) (*AccountBalanceCoin, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.SettleCoinBalance(exchangeAccount)
}

func (e *Exchange) AccountInfo(exchangeAccount subsettings.ExchangeAccount) (*AccountInfo, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.AccountInfo(exchangeAccount)
}

func (e *Exchange) SetAccountMarginMode(exchangeAccount subsettings.ExchangeAccount, marginMode *string) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetAccountMarginMode(exchangeAccount, marginMode)
}

func (e *Exchange) SymbolMarginType(exchangeAccount subsettings.ExchangeAccount, symbol string) (string, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return "", err
	}
	return client.SymbolMarginType(exchangeAccount, symbol)
}

func (e *Exchange) SetSymbolMarginType(exchangeAccount subsettings.ExchangeAccount, symbol string, marginType string) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetSymbolMarginType(exchangeAccount, symbol, marginType)
}

func (e *Exchange) SetAccountPositionMode(exchangeAccount subsettings.ExchangeAccount, positionMode string) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetAccountPositionMode(exchangeAccount, positionMode)
}

func (e *Exchange) SetLeverage(exchangeAccount subsettings.ExchangeAccount, symbol string, leverage int) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetLeverage(exchangeAccount, symbol, leverage)
}

func (e *Exchange) SetLeverageWebsocket(exchangeAccount subsettings.ExchangeAccount, symbol string, leverage int) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetLeverageWebsocket(exchangeAccount, symbol, leverage)
}

func (e *Exchange) Ticker(exchangeAccount subsettings.ExchangeAccount, symbol string) (*Ticker, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.Ticker(exchangeAccount, symbol)
}

func (e *Exchange) SymbolInfo(exchangeAccount subsettings.ExchangeAccount, symbol string, cached bool) (*SymbolInfo, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.SymbolInfo(exchangeAccount, symbol, cached)
}

// SetPositionTpsl sets the position's take profit and stop loss. A nil
// takeProfit or stopLoss means the position must not have that leg: any
// existing order for it is canceled.
func (e *Exchange) SetPositionTpsl(
	exchangeAccount subsettings.ExchangeAccount,
	symbol string,
	positionSide string,
	takeProfit *float64,
	stopLoss *float64,
) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetPositionTpsl(exchangeAccount, symbol, positionSide, takeProfit, stopLoss)
}

func (e *Exchange) WebsocketsEnabled(exchangeAccount subsettings.ExchangeAccount) bool {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return false
	}
	return client.WebsocketsEnabled(exchangeAccount)
}

// SetPositionTpslWebsocket is SetPositionTpsl over the websocket connection
// when possible, with the same nil-cancels semantics.
func (e *Exchange) SetPositionTpslWebsocket(
	exchangeAccount subsettings.ExchangeAccount,
	symbol string,
	positionSide string,
	takeProfit *float64,
	stopLoss *float64,
	tpOrder *OpenStopOrder,
	slOrder *OpenStopOrder,
) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.SetPositionTpslWebsocket(exchangeAccount, symbol, positionSide, takeProfit, stopLoss, tpOrder, slOrder)
}

func (e *Exchange) CancelStopOrders(exchangeAccount subsettings.ExchangeAccount, orders []OpenStopOrder) error {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return err
	}
	return client.CancelStopOrders(exchangeAccount, orders)
}

func (e *Exchange) HyperToExchangeSymbol(
	exchangeAccount subsettings.ExchangeAccount,
	hyperSymbol string,
) (string, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return "", err
	}
	return client.HyperToExchangeSymbol(exchangeAccount, hyperSymbol)
}

func (e *Exchange) GetApiKeyAndUserInformation(exchangeAccount subsettings.ExchangeAccount, getApiKeyInformation bool) (*ApiKeyInfo, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.GetApiKeyAndUserInformation(exchangeAccount, getApiKeyInformation)
}

func (e *Exchange) UserRateLimit(exchangeAccount subsettings.ExchangeAccount) (*RateLimitInfo, error) {
	client, err := e.client(exchangeAccount)
	if err != nil {
		return nil, err
	}
	return client.UserRateLimit(exchangeAccount)
}
