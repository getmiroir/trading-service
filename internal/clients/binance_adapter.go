package clients

import (
	"context"
	"errors"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

type binanceExchangeAdapter struct {
	client *Binance
	mapper *SymbolMapper
}

func (a *binanceExchangeAdapter) ClosePosition(account subsettings.ExchangeAccount, position Position) error {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	return mapBinanceError(a.client.ClosePosition(*binanceAccount, position.Symbol, position.Side, position.Size))
}

func (a *binanceExchangeAdapter) PlaceOrder(account subsettings.ExchangeAccount, symbol, side, size string, takeProfit, stopLoss *float64, reduceOnly, closeOnTrigger bool) (*PlaceOrder, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binancePlaceOrder, err := a.client.PlaceMarketOrder(*binanceAccount, symbol, side, size, takeProfit, stopLoss, false)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	placeOrder := NewPlaceOrderFromBinancePlaceOrder(*binancePlaceOrder)
	return &placeOrder, nil
}

func (a *binanceExchangeAdapter) Positions(account subsettings.ExchangeAccount) ([]Position, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binancePositions, err := a.client.Positions(*binanceAccount)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	positions := make([]Position, len(binancePositions))
	for i, p := range binancePositions {
		positions[i] = NewPositionFromBinancePosition(p)
	}
	return positions, nil
}

func (a *binanceExchangeAdapter) Position(account subsettings.ExchangeAccount, symbol string) (*PositionGenericInfo, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binancePosition, err := a.client.Position(*binanceAccount, symbol)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	if binancePosition == nil {
		return nil, nil
	}
	position := NewPositionGenericInfoFromBinancePosition(*binancePosition)
	return &position, nil
}

func (a *binanceExchangeAdapter) OpenStopOrders(account subsettings.ExchangeAccount) ([]OpenStopOrder, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binanceOrders, err := a.client.OpenOrders(*binanceAccount, nil)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	orders := make([]OpenStopOrder, 0, len(binanceOrders))
	for _, order := range binanceOrders {
		if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
			orders = append(orders, NewOpenStopOrderFromBinanceOpenOrder(order))
		}
	}
	return orders, nil
}

func (a *binanceExchangeAdapter) SettleCoinBalance(account subsettings.ExchangeAccount) (*AccountBalanceCoin, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	assets, err := a.client.SettleCoinBalance(*binanceAccount)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	accountBalanceCoin := NewAccountBalanceCoinFromBinanceAccountInfoAssets(*assets)
	return &accountBalanceCoin, nil
}

func (a *binanceExchangeAdapter) AccountInfo(account subsettings.ExchangeAccount) (*AccountInfo, error) {
	accountInfo := NewAccountInfoFromBinanceAccountInfo()
	return &accountInfo, nil
}

func (a *binanceExchangeAdapter) SetAccountMarginMode(account subsettings.ExchangeAccount, marginMode *string) error {
	// Account margin mode is not supported for Binance
	return nil
}

func (a *binanceExchangeAdapter) SymbolMarginType(account subsettings.ExchangeAccount, symbol string) (string, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	symbolConfig, err := a.client.SymbolConfig(*binanceAccount, symbol)
	if err != nil {
		return "", mapBinanceError(err)
	}
	return symbolConfig.MarginType, nil
}

func (a *binanceExchangeAdapter) SetSymbolMarginType(account subsettings.ExchangeAccount, symbol, marginType string) error {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	return mapBinanceError(a.client.SetSymbolMarginType(*binanceAccount, symbol, marginType))
}

func (a *binanceExchangeAdapter) SetAccountPositionMode(account subsettings.ExchangeAccount, positionMode string) error {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binancePositionMode, err := a.client.AccountPositionMode(*binanceAccount)
	if err != nil {
		return mapBinanceError(err)
	}
	if binancePositionMode != positionMode {
		return mapBinanceError(a.client.SetAccountPositionMode(*binanceAccount, positionMode))
	}
	return nil
}

func (a *binanceExchangeAdapter) SetLeverage(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	return mapBinanceError(a.client.SetLeverage(*binanceAccount, symbol, leverage))
}

func (a *binanceExchangeAdapter) SetLeverageWebsocket(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	// Binance has no websocket leverage endpoint; fall back to REST.
	return a.SetLeverage(account, symbol, leverage)
}

func (a *binanceExchangeAdapter) Ticker(account subsettings.ExchangeAccount, symbol string) (*Ticker, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binanceTicker, err := a.client.Ticker(*binanceAccount, symbol)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	ticker := NewTickerFromBinanceTickerPrice(*binanceTicker)
	return &ticker, nil
}

func (a *binanceExchangeAdapter) SymbolInfo(account subsettings.ExchangeAccount, symbol string, cached bool) (*SymbolInfo, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	binanceSymbolInfo, leverageBracket, err := a.client.SymbolInfo(*binanceAccount, symbol, cached)
	if err != nil {
		return nil, mapBinanceError(err)
	}
	symbolInfo := NewSymbolInfoFromBinanceSymbolInfo(*binanceSymbolInfo)
	// Max leverage comes from the separate leverageBracket route; the exchange
	// client returns both raw objects and the mapping into our model lives here.
	symbolInfo.MaxLeverage = leverageBracket.MaxLeverage()
	return &symbolInfo, nil
}

func (a *binanceExchangeAdapter) SetPositionTpsl(account subsettings.ExchangeAccount, symbol, positionSide string, takeProfit, stopLoss *float64) error {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	// The client cancels all of the symbol's algo orders before placing the
	// requested legs, so a nil leg (0 for the client) is canceled and not
	// re-placed.
	return mapBinanceError(a.client.SetPositionTpsl(*binanceAccount, symbol, positionSide, valueOrZero(takeProfit), valueOrZero(stopLoss)))
}

func (a *binanceExchangeAdapter) SetPositionTpslWebsocket(
	account subsettings.ExchangeAccount,
	symbol string,
	positionSide string,
	takeProfit *float64,
	stopLoss *float64,
	tpOrder *OpenStopOrder,
	slOrder *OpenStopOrder,
) error {
	// The websocket client cancels the legs whose order IDs are provided, then
	// places the requested ones, so a nil leg (0 for the client) is canceled
	// and not re-placed.
	var takeProfitOrderId int64
	var stopLossOrderId int64
	if tpOrder != nil && tpOrder.AlgoID != nil {
		takeProfitOrderId = *tpOrder.AlgoID
	}
	if slOrder != nil && slOrder.AlgoID != nil {
		stopLossOrderId = *slOrder.AlgoID
	}
	return mapBinanceError(a.client.SetPositionTpslWebsocket(
		account.Name,
		takeProfitOrderId,
		stopLossOrderId,
		symbol,
		positionSide,
		valueOrZero(takeProfit),
		valueOrZero(stopLoss),
	))
}

func (a *binanceExchangeAdapter) HyperToExchangeSymbol(account subsettings.ExchangeAccount, hyperSymbol string) (string, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)
	return a.mapper.HyperToBinanceSymbol(binanceAccount.SettleCoin, hyperSymbol)
}

// UserRateLimit only applies to Hyperliquid
func (a *binanceExchangeAdapter) UserRateLimit(_ subsettings.ExchangeAccount) (*RateLimitInfo, error) {
	return nil, nil
}

func (a *binanceExchangeAdapter) CancelStopOrders(_ subsettings.ExchangeAccount, _ []OpenStopOrder) error {
	// NOT IMPLEMENTED as it is not needed yet
	return nil
}

func (a *binanceExchangeAdapter) GetApiKeyAndUserInformation(account subsettings.ExchangeAccount, getApiKeyInformation bool) (*ApiKeyInfo, error) {
	binanceAccount := account.Account.(*subsettings.BinanceAccount)

	var spotApiKeyInfo *BinanceApiKeyPermissions
	var spotAccountInfo *BinanceSpotAccountInfo
	var err error

	if getApiKeyInformation {
		// we only receive createTime as a timestamp with format: 1698645219000
		// we compute manually the expiredAt by adding 90 days to the createdAt
		// timestamps are in milliseconds
		spotApiKeyInfo, err = a.client.ApiKeyPermissions(*binanceAccount)
		if err != nil {
			return nil, mapBinanceError(err)
		}
		spotAccountInfo, err = a.client.SpotAccountInfo(*binanceAccount, false)
		if err != nil {
			return nil, mapBinanceError(err)
		}
	}

	apiKeyInfo := NewApiKeyInfoFromBinanceApiKeyInfo(spotApiKeyInfo, spotAccountInfo)
	return &apiKeyInfo, nil
}

func (a *binanceExchangeAdapter) InvalidateCache() {
	a.client.InvalidateCache()
}

func (a *binanceExchangeAdapter) WebsocketsEnabled(account subsettings.ExchangeAccount) bool {
	return true
}

func (a *binanceExchangeAdapter) ConnectWebsockets(ctx context.Context) {
	a.client.ConnectAllWebsockets(ctx)
}

func (a *binanceExchangeAdapter) CloseWebsockets() {
	a.client.CloseAllWebsockets()
}

func mapBinanceError(err error) error {
	var e *internal.BinanceBadStatusCodeError
	if errors.As(err, &e) && e.ErrorCode != nil {
		switch *e.ErrorCode {
		case -2019:
			return newExchangeError(ErrNotEnoughMargin, err)
		case -1121:
			return newExchangeError(ErrSymbolNotExist, err)
		case -1021:
			return newExchangeError(ErrMachineInternalClock, err)
		case -4411:
			return newExchangeError(ErrTradFiAgreementRequired, err)
		case -2027:
			return newExchangeError(ErrPositionExceedsMarginTier, err)
		case -4421:
			return newExchangeError(ErrSubaccountLeverageRestricted, err)
		case -4300:
			return newExchangeError(ErrNewFuturesAccountLeverageRestricted, err)
		case -2015:
			return newExchangeError(ErrInvalidApiKeyOrPermissions, err)
		}
	}
	return err
}
