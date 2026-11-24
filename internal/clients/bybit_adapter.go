package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
)

type bybitExchangeAdapter struct {
	client *Bybit
	mapper *SymbolMapper
}

func (a *bybitExchangeAdapter) ClosePosition(account subsettings.ExchangeAccount, position Position) error {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	closeOrderSide := utils.QuantityToSide(-float64(utils.SideToMultiplier(position.Side)))
	_, err := a.client.PlaceMarketOrder(*bybitAccount, position.Symbol, closeOrderSide, "0", nil, nil, true, true)
	return mapBybitError(err)
}

func (a *bybitExchangeAdapter) PlaceOrder(account subsettings.ExchangeAccount, symbol, side, size string, takeProfit, stopLoss *float64, reduceOnly, closeOnTrigger bool) (*PlaceOrder, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitPlaceOrder, err := a.client.PlaceMarketOrder(*bybitAccount, symbol, side, size, takeProfit, stopLoss, reduceOnly, closeOnTrigger)
	if err != nil {
		return nil, mapBybitError(err)
	}
	placeOrder := NewPlaceOrderFromBybitPlaceOrder(*bybitPlaceOrder)
	return &placeOrder, nil
}

func (a *bybitExchangeAdapter) Positions(account subsettings.ExchangeAccount) ([]Position, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitPositions, err := a.client.Positions(*bybitAccount)
	if err != nil {
		return nil, mapBybitError(err)
	}
	positions := make([]Position, len(bybitPositions))
	for i, p := range bybitPositions {
		positions[i] = NewPositionFromBybitPosition(p)
	}
	return positions, nil
}

func (a *bybitExchangeAdapter) Position(account subsettings.ExchangeAccount, symbol string) (*PositionGenericInfo, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitPosition, err := a.client.Position(*bybitAccount, symbol)
	if err != nil {
		return nil, mapBybitError(err)
	}
	if bybitPosition == nil {
		return nil, nil
	}
	position := NewPositionGenericInfoFromBybitPosition(*bybitPosition)
	return &position, nil
}

func (a *bybitExchangeAdapter) OpenStopOrders(account subsettings.ExchangeAccount) ([]OpenStopOrder, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitOrders, err := a.client.OpenStopOrders(*bybitAccount)
	if err != nil {
		return nil, mapBybitError(err)
	}
	orders := make([]OpenStopOrder, len(bybitOrders))
	for i, order := range bybitOrders {
		orders[i] = NewOpenStopOrderFromBybitOpenOrder(order)
	}
	return orders, nil
}

func (a *bybitExchangeAdapter) SettleCoinBalance(account subsettings.ExchangeAccount) (*AccountBalanceCoin, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	balance, err := a.client.SettleCoinBalance(*bybitAccount)
	if err != nil {
		return nil, mapBybitError(err)
	}
	accountBalanceCoin := NewAccountBalanceCoinFromBybitAccountBalance(*balance)
	return &accountBalanceCoin, nil
}

func (a *bybitExchangeAdapter) AccountInfo(account subsettings.ExchangeAccount) (*AccountInfo, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	info, err := a.client.AccountInfo(*bybitAccount)
	if err != nil {
		return nil, mapBybitError(err)
	}
	accountInfo := NewAccountInfoFromBybitAccountInfo(*info)
	return &accountInfo, nil
}

func (a *bybitExchangeAdapter) SetAccountMarginMode(account subsettings.ExchangeAccount, marginMode *string) error {
	if marginMode == nil {
		return nil
	}
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	return mapBybitError(a.client.SetAccountMarginMode(*bybitAccount, *marginMode))
}

func (a *bybitExchangeAdapter) SymbolMarginType(account subsettings.ExchangeAccount, symbol string) (string, error) {
	// Symbol margin type is not supported for Bybit
	return "", nil
}

func (a *bybitExchangeAdapter) SetSymbolMarginType(account subsettings.ExchangeAccount, symbol, marginType string) error {
	// Symbol margin mode is not supported for Bybit
	return nil
}

func (a *bybitExchangeAdapter) SetAccountPositionMode(account subsettings.ExchangeAccount, positionMode string) error {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	return mapBybitError(a.client.SetAccountPositionMode(*bybitAccount, positionMode))
}

func (a *bybitExchangeAdapter) SetLeverage(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	return mapBybitError(a.client.SetLeverage(*bybitAccount, symbol, leverage))
}

func (a *bybitExchangeAdapter) SetLeverageWebsocket(account subsettings.ExchangeAccount, symbol string, leverage int) error {
	// Bybit has no websocket leverage endpoint; fall back to REST.
	return a.SetLeverage(account, symbol, leverage)
}

func (a *bybitExchangeAdapter) Ticker(account subsettings.ExchangeAccount, symbol string) (*Ticker, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitTicker, err := a.client.Ticker(*bybitAccount, symbol)
	if err != nil {
		return nil, mapBybitError(err)
	}
	if bybitTicker == nil {
		return nil, fmt.Errorf("no ticker found for symbol %s", symbol)
	}
	ticker := NewTickerFromBybitTicker(*bybitTicker)
	return &ticker, nil
}

func (a *bybitExchangeAdapter) SymbolInfo(account subsettings.ExchangeAccount, symbol string, cached bool) (*SymbolInfo, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	bybitSymbolInfo, err := a.client.SymbolInfo(*bybitAccount, symbol, cached)
	if err != nil {
		return nil, mapBybitError(err)
	}
	symbolInfo := NewSymbolInfoFromBybitSymbolInfo(*bybitSymbolInfo)
	return &symbolInfo, nil
}

func (a *bybitExchangeAdapter) SetPositionTpsl(account subsettings.ExchangeAccount, symbol, positionSide string, takeProfit, stopLoss *float64) error {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	// Bybit's trading-stop endpoint treats a price of 0 as "cancel that leg",
	// matching the nil-cancels semantics.
	return mapBybitError(a.client.SetPositionTpsl(*bybitAccount, symbol, valueOrZero(takeProfit), valueOrZero(stopLoss)))
}

func (a *bybitExchangeAdapter) SetPositionTpslWebsocket(
	account subsettings.ExchangeAccount,
	symbol string,
	positionSide string,
	takeProfit *float64,
	stopLoss *float64,
	tpOrder *OpenStopOrder,
	slOrder *OpenStopOrder,
) error {
	// Websocket amend can only modify existing orders: fall back to REST when a
	// leg must be canceled (nil with a leftover order) or created (set without
	// an existing order to amend).
	if takeProfit == nil && tpOrder != nil {
		a.client.logger.Info("Using REST for TPSL", "reason", "canceling leftover TP order")
		return a.SetPositionTpsl(account, symbol, positionSide, takeProfit, stopLoss)
	}
	if stopLoss == nil && slOrder != nil {
		a.client.logger.Info("Using REST for TPSL", "reason", "canceling leftover SL order")
		return a.SetPositionTpsl(account, symbol, positionSide, takeProfit, stopLoss)
	}

	var tpOrderID, slOrderID *string
	if takeProfit != nil {
		if tpOrder == nil || tpOrder.OrderID == nil {
			a.client.logger.Info("Using REST for TPSL", "reason", "TP order missing for websocket amend")
			return a.SetPositionTpsl(account, symbol, positionSide, takeProfit, stopLoss)
		}
		tpOrderID = tpOrder.OrderID
	}
	if stopLoss != nil {
		if slOrder == nil || slOrder.OrderID == nil {
			a.client.logger.Info("Using REST for TPSL", "reason", "SL order missing for websocket amend")
			return a.SetPositionTpsl(account, symbol, positionSide, takeProfit, stopLoss)
		}
		slOrderID = slOrder.OrderID
	}
	if tpOrderID == nil && slOrderID == nil {
		// Nothing requested and no leftover orders to cancel
		return nil
	}
	return mapBybitError(a.client.SetPositionTpslWebsocket(
		account.Name,
		symbol,
		tpOrderID,
		slOrderID,
		valueOrZero(takeProfit),
		valueOrZero(stopLoss),
	))
}

func (a *bybitExchangeAdapter) HyperToExchangeSymbol(account subsettings.ExchangeAccount, hyperSymbol string) (string, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	return a.mapper.HyperToBybitSymbol(bybitAccount.SettleCoin, hyperSymbol)
}

// UserRateLimit is not supported for Bybit; the Hyperliquid TP/SL rate-limit
// logic only applies to Hyperliquid destination accounts.
func (a *bybitExchangeAdapter) UserRateLimit(_ subsettings.ExchangeAccount) (*RateLimitInfo, error) {
	return nil, nil
}

func (a *bybitExchangeAdapter) CancelStopOrders(_ subsettings.ExchangeAccount, _ []OpenStopOrder) error {
	// NOT IMPLEMENTED as it is not needed yet
	return nil
}

func (a *bybitExchangeAdapter) GetApiKeyAndUserInformation(account subsettings.ExchangeAccount, getApiKeyInformation bool) (*ApiKeyInfo, error) {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	info, err := a.client.ApiKeyInfo(*bybitAccount)
	if err != nil {
		return nil, mapBybitError(err)
	}

	apiKeyInfo, err := NewApiKeyInfoFromBybitApiKeyInfo(*info)
	if err != nil {
		return nil, err
	}
	return apiKeyInfo, nil
}

func (a *bybitExchangeAdapter) InvalidateCache() {
	a.client.InvalidateCache()
}

func (a *bybitExchangeAdapter) WebsocketsEnabled(account subsettings.ExchangeAccount) bool {
	bybitAccount := account.Account.(*subsettings.BybitAccount)
	return BybitWebsocketsEnabled(bybitAccount.IsDemo, bybitAccount.IsTestnet)
}

func (a *bybitExchangeAdapter) ConnectWebsockets(ctx context.Context) {
	a.client.ConnectAllWebsockets(ctx)
}

func (a *bybitExchangeAdapter) CloseWebsockets() {
	a.client.CloseAllWebsockets()
}

func mapBybitError(err error) error {
	var e *internal.BybitBadReturnCodeError
	if errors.As(err, &e) {
		switch {
		case e.RetCode == 110007:
			return newExchangeError(ErrNotEnoughMargin, err)
		case e.RetCode == 110090:
			return newExchangeError(ErrPositionExceedsMarginTier, err)
		case e.RetCode == 10001 && strings.Contains(e.Message, "symbol invalid"):
			return newExchangeError(ErrSymbolNotExist, err)
		case e.RetCode == 10002:
			return newExchangeError(ErrMachineInternalClock, err)
		case e.RetCode == 10003, e.RetCode == 10005, e.RetCode == 10010, e.RetCode == 33004:
			return newExchangeError(ErrInvalidApiKeyOrPermissions, err)
		// Codes 110123 and 110125 extracted from https://bybit-exchange.github.io/docs/v5/error; 110126 from testing.
		case e.RetCode == 110123, e.RetCode == 110125, e.RetCode == 110126:
			return newExchangeError(ErrTradFiAgreementRequired, err)
		}
	}
	return err
}
