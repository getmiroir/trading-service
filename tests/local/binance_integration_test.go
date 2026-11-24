package local

import (
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/joho/godotenv"
	"go.yaml.in/yaml/v4"
)

func TestBinanceIntegration(t *testing.T) {
	godotenv.Load(".env.test")
	apiKey := os.Getenv("BINANCE_API_KEY")
	apiSecret := os.Getenv("BINANCE_API_SECRET")
	spotApiKey := os.Getenv("BINANCE_SPOT_API_KEY")
	spotApiSecret := os.Getenv("BINANCE_SPOT_API_SECRET")
	orderSide := os.Getenv("BINANCE_ORDER_SIDE")

	if apiKey == "" || apiSecret == "" || spotApiKey == "" || spotApiSecret == "" || orderSide == "" {
		t.Fatalf("BINANCE_API_KEY, BINANCE_API_SECRET, BINANCE_SPOT_API_KEY, BINANCE_SPOT_API_SECRET and BINANCE_ORDER_SIDE must be set!")
	}

	// symbol
	symbol := "BTCUSDT"

	// Create a BinanceAccount using YAML unmarshaling to apply defaults
	// The key and secret will be read from environment variables
	accountYAML := `
is_demo: true
key:
  from_env_var: BINANCE_API_KEY
secret:
  from_env_var: BINANCE_API_SECRET
`

	spotAccountYAML := `
is_demo: true
key:
  from_env_var: BINANCE_SPOT_API_KEY
secret:
  from_env_var: BINANCE_SPOT_API_SECRET
`
	var account subsettings.BinanceAccount
	if err := yaml.Unmarshal([]byte(accountYAML), &account); err != nil {
		t.Fatalf("Failed to unmarshal account YAML: %v", err)
	}

	var spotAccount subsettings.BinanceAccount
	if err := yaml.Unmarshal([]byte(spotAccountYAML), &spotAccount); err != nil {
		t.Fatalf("Failed to unmarshal spot account YAML: %v", err)
	}

	cfg := subsettings.DefaultBinance()
	cfg.Accounts["test"] = account
	cfg.Accounts["test_spot"] = spotAccount

	// create binance client
	binance := clients.NewBinance(cfg, false)

	// test getting account info
	_, err := binance.AccountInfo(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get account info: %v", err)
	}

	// test getting spot account info
	_, err = binance.SpotAccountInfo(cfg.Accounts["test_spot"], false)
	if err != nil {
		t.Fatalf("Failed to get spot account info: %v", err)
	}

	// get account position mode
	positionMode, err := binance.AccountPositionMode(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get account position mode: %v", err)
	}
	if positionMode != "one-way" {
		t.Fatalf("Expected account position mode to be one-way, got %s", positionMode)
	}

	// set account position mode
	err = binance.SetAccountPositionMode(cfg.Accounts["test"], "one-way")
	if err != nil {
		t.Fatalf("Failed to set account position mode: %v", err)
	}

	// recover settle coin balance
	settleCoinBalance, err := binance.SettleCoinBalance(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get settle coin balance: %v", err)
	}

	// recover from settle coin balance how much we can trade with
	equity := settleCoinBalance.Equity()

	// recover ticker for symbol
	ticker, err := binance.Ticker(cfg.Accounts["test"], symbol)
	if err != nil {
		t.Fatalf("Failed to get ticker: %v", err)
	}
	price := ticker.Price.Float64()
	takeProfit := price * 1.10
	stopLoss := price * 0.9
	if orderSide == "Sell" {
		takeProfit = price * 0.9
		stopLoss = price * 1.1
	}

	// recover symbol information
	symbolInfo, _, err := binance.SymbolInfo(cfg.Accounts["test"], symbol, true)
	if err != nil {
		t.Fatalf("Failed to get symbol info: %v", err)
	}
	minOrderQty := 2 * symbolInfo.LotSizeFilter().MinOrderQty
	minNotionalValue := 2 * symbolInfo.LotSizeFilter().MinNotionalValue
	stepQty := symbolInfo.LotSizeFilter().QtyStep
	stepPrecision := int(math.Max(0, math.Ceil(-math.Log10(stepQty))))

	// set up test trade size.
	tradeSize := equity * 0.01 / price
	if tradeSize < minOrderQty {
		tradeSize = minNotionalValue / price
	}
	tradeSize = math.Round(tradeSize*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	tradeSizeStr := strconv.FormatFloat(tradeSize, 'f', stepPrecision, 64)
	takeProfit = math.Round(takeProfit*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)
	stopLoss = math.Round(stopLoss*math.Pow10(stepPrecision)) / math.Pow10(stepPrecision)

	// recover symbol config
	symbolConfig, err := binance.SymbolConfig(cfg.Accounts["test"], symbol)
	if err != nil {
		t.Fatalf("Failed to get symbol config: %v", err)
	}
	leverage := symbolConfig.Leverage
	if leverage == 10 {
		leverage = 5
	} else {
		leverage = 10
	}

	// update symbol leverage
	err = binance.SetLeverage(cfg.Accounts["test"], symbol, leverage)
	if err != nil {
		t.Fatalf("Failed to set leverage: %v", err)
	}

	// update symbol margin type
	err = binance.SetSymbolMarginType(cfg.Accounts["test"], symbol, "ISOLATED")
	if err != nil {
		t.Fatalf("Failed to set symbol margin type: %v", err)
	}

	// place a position
	_, err = binance.PlaceMarketOrder(cfg.Accounts["test"], symbol, orderSide, tradeSizeStr, &takeProfit, &stopLoss, false)
	if err != nil {
		t.Fatalf("Failed to place position: %v", err)
	}

	// list all positions
	positions, err := binance.Positions(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("Expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != symbol {
		t.Fatalf("Expected %s position, got %s", symbol, positions[0].Symbol)
	}
	if math.Abs(positions[0].PositionAmt.Float64()) != tradeSize {
		t.Fatalf("Expected position size %f, got %f", tradeSize, positions[0].PositionAmt.Float64())
	}

	// get only the symbol position
	position, err := binance.Position(cfg.Accounts["test"], symbol)
	if err != nil {
		t.Fatalf("Failed to get %s position: %v", symbol, err)
	}
	if position.Side() != orderSide {
		t.Fatalf("Expected %s position, got %s", orderSide, position.Side())
	}
	if math.Abs(position.PositionAmt.Float64()) != tradeSize {
		t.Fatalf("Expected position size %f, got %f", tradeSize, position.PositionAmt.Float64())
	}

	// get the open orders and check that we have a take profit and stop loss order
	orders, err := binance.OpenOrders(cfg.Accounts["test"], &symbol)
	if err != nil {
		t.Fatalf("Failed to get open orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("Expected 2 open orders, got %d", len(orders))
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
	err = binance.SetPositionTpsl(cfg.Accounts["test"], symbol, orderSide, takeProfit, stopLoss)
	if err != nil {
		t.Fatalf("Failed to set TP/SL: %v", err)
	}

	// get the open orders and check that we have a take profit and stop loss order
	orders, err = binance.OpenOrders(cfg.Accounts["test"], &symbol)
	if err != nil {
		t.Fatalf("Failed to get open orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("Expected 2 open orders, got %d", len(orders))
	}

	// close our position
	err = binance.ClosePosition(cfg.Accounts["test"], symbol, orderSide, tradeSize)
	if err != nil {
		t.Fatalf("Failed to close position: %v", err)
	}

	// list all positions
	positions, err = binance.Positions(cfg.Accounts["test"])
	if err != nil {
		t.Fatalf("Failed to get positions: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("Expected 0 positions, got %d", len(positions))
	}

}
