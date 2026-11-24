package local

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
	"github.com/joho/godotenv"
	"go.yaml.in/yaml/v4"
)

// GetTestBybitAccount creates a BybitAccount using YAML unmarshaling to apply defaults
// The key and secret will be read from environment variables
func GetTestBybitAccount() (interface{}, error) {
	godotenv.Load(".env.test")
	apiKey := os.Getenv("BYBIT_WEBSOCKET_API_KEY")
	apiSecret := os.Getenv("BYBIT_WEBSOCKET_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("BYBIT_WEBSOCKET_API_KEY and BYBIT_WEBSOCKET_API_SECRET must be set!")
	}

	accountYAML := `
is_demo: true
key:
  from_env_var: BYBIT_WEBSOCKET_API_KEY
secret:
  from_env_var: BYBIT_WEBSOCKET_API_SECRET
`
	var account subsettings.BybitAccount
	if err := yaml.Unmarshal([]byte(accountYAML), &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account YAML: %v", err)
	}
	return &account, nil
}

func TestBybitWebsocketsIntegration(t *testing.T) {
	godotenv.Load(".env.test")
	orderSide := os.Getenv("BYBIT_WEBSOCKET_ORDER_SIDE")

	if orderSide == "" {
		t.Fatalf("BYBIT_ORDER_SIDE must be set!")
	}
	closeOrderSide := utils.QuantityToSide(-float64(utils.SideToMultiplier(orderSide)))

	account, err := GetTestBybitAccount()
	if err != nil {
		t.Fatalf("Failed to get test bybit account: %v", err)
	}

	cfg := subsettings.DefaultBybit()
	cfg.Accounts["test"] = *account.(*subsettings.BybitAccount)

	// create bybit client
	bybit := clients.NewBybit(cfg, false)

	// connect to the account websocket
	err = bybit.ConnectWebsocket("test")
	if err != nil {
		t.Fatalf("Failed to connect to websocket: %v", err)
	}

	// test getting account info
	_, err = bybit.AccountInfo(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get account info: %v", err)
	}

	// set account position mode
	err = bybit.SetAccountPositionMode(cfg.Accounts["test"], "one-way")
	if err != nil {
		t.Fatalf("Failed to set account position mode: %v", err)
	}

	// set account margin mode
	err = bybit.SetAccountMarginMode(cfg.Accounts["test"], "ISOLATED_MARGIN")
	if err != nil {
		t.Fatalf("Failed to set account margin mode: %v", err)
	}

	// recover settle coin balance
	settleCoinBalance, err := bybit.SettleCoinBalance(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get settle coin balance: %v", err)
	}

	// recover from settle coin balance how much we can trade with
	equity := settleCoinBalance.Equity.Float64()

	// recover ticker for BTCUSDT
	ticker, err := bybit.Ticker(cfg.Accounts["test"], "BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to get ticker: %v", err)
	}
	price := ticker.LastPrice.Float64()
	takeProfit := price * 1.10
	stopLoss := price * 0.9
	if orderSide == "Sell" {
		takeProfit = price * 0.9
		stopLoss = price * 1.1
	}

	// recover BTCUSDT symbol information
	symbolInfo, err := bybit.SymbolInfo(cfg.Accounts["test"], "BTCUSDT", true)
	if err != nil {
		t.Fatalf("Failed to get symbol info: %v", err)
	}
	minOrderQty := 2 * symbolInfo.LotSizeFilter.MinOrderQty.Float64()
	minNotionalValue := 2 * symbolInfo.LotSizeFilter.MinNotionalValue.Float64()
	stepQty := symbolInfo.LotSizeFilter.QtyStep.Float64()
	stepPrecision := int(math.Max(0, math.Ceil(-math.Log10(stepQty))))

	// set up test trade size.
	tradeSize := equity * 0.1 / price
	if tradeSize < minOrderQty {
		tradeSize = minNotionalValue / price
	}
	tradeSize = math.Round(tradeSize*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	tradeSizeStr := strconv.FormatFloat(tradeSize, 'f', stepPrecision, 64)
	takeProfit = math.Round(takeProfit*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	stopLoss = math.Round(stopLoss*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)

	// update symbol leverage
	err = bybit.SetLeverage(cfg.Accounts["test"], "BTCUSDT", 10)
	if err != nil {
		t.Fatalf("Failed to set leverage: %v", err)
	}

	// place market order
	err = bybit.PlaceMarketOrderWebsocket("test", "BTCUSDT", orderSide, tradeSizeStr, &takeProfit, &stopLoss, false, false)
	if err != nil {
		t.Fatalf("Failed to place market order: %v", err)
	}

	// list all positions
	positions, err := bybit.Positions(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("Expected 1 position, got %d", len(positions))
	}
	if math.Abs(positions[0].Size.Float64()) != tradeSize {
		t.Fatalf("Expected position size %f, got %f", tradeSize, positions[0].Size.Float64())
	}

	// get only the BTCUSDT position
	position, err := bybit.Position(cfg.Accounts["test"], "BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to get position: %v", err)
	}
	if position.Side != orderSide {
		t.Fatalf("Expected %s position, got %s", orderSide, position.Side)
	}
	if math.Abs(position.Size.Float64()) != tradeSize {
		t.Fatalf("Expected position size %f, got %f", tradeSize, position.Size.Float64())
	}

	// list open stop orders
	openStopOrders, err := bybit.OpenStopOrders(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get open stop orders: %v", err)
	}
	// recover BTCUSDT open stop orders
	var btcusdtOpenStopOrders []clients.BybitOpenOrder
	for _, order := range openStopOrders {
		if order.Symbol == "BTCUSDT" {
			btcusdtOpenStopOrders = append(btcusdtOpenStopOrders, order)
		}
	}
	// filter the order using their stopOrderType
	var takeProfitOrder *clients.BybitOpenOrder
	var stopLossOrder *clients.BybitOpenOrder
	for _, order := range btcusdtOpenStopOrders {
		switch order.StopOrderType {
		case "TakeProfit":
			takeProfitOrder = &order
		case "StopLoss":
			stopLossOrder = &order
		}
	}

	// update the TP/SL
	takeProfit = price * 1.20
	stopLoss = price * 0.80
	if orderSide == "Sell" {
		takeProfit = price * 0.80
		stopLoss = price * 1.20
	}
	takeProfit = math.Round(takeProfit*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	stopLoss = math.Round(stopLoss*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	err = bybit.SetPositionTpslWebsocket("test", "BTCUSDT", &takeProfitOrder.OrderID, &stopLossOrder.OrderID, takeProfit, stopLoss)
	if err != nil {
		t.Fatalf("Failed to set TP/SL: %v", err)
	}

	// close the position
	err = bybit.PlaceMarketOrderWebsocket("test", "BTCUSDT", closeOrderSide, "0", nil, nil, true, true)
	if err != nil {
		t.Fatalf("Failed to close position: %v", err)
	}

	// list all positions
	positions, err = bybit.Positions(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get positions: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("Expected 0 positions, got %d", len(positions))
	}

	// close the websocket
	err = bybit.CloseWebsocket("test")
	if err != nil {
		t.Fatalf("Failed to close websocket: %v", err)
	}
}

func TestBybitWebsocketsKeptOpen(t *testing.T) {
	account, err := GetTestBybitAccount()
	if err != nil {
		t.Fatalf("Failed to get test bybit account: %v", err)
	}

	cfg := subsettings.DefaultBybit()
	cfg.Accounts["test"] = *account.(*subsettings.BybitAccount)

	// create bybit client
	bybit := clients.NewBybit(cfg, true)

	// connect to the account websocket
	err = bybit.ConnectWebsocket("test")
	if err != nil {
		t.Fatalf("Failed to connect to websocket: %v", err)
	}

	// test getting account info
	_, err = bybit.AccountInfo(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get account info: %v", err)
	}

	// sleep for 30 seconds
	time.Sleep(30 * time.Second)

	// test getting account info again
	_, err = bybit.AccountInfo(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get account info: %v", err)
	}

	// close the websocket
	err = bybit.CloseWebsocket("test")
	if err != nil {
		t.Fatalf("Failed to close websocket: %v", err)
	}
}
