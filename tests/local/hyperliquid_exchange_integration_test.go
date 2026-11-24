package local

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
	"github.com/joho/godotenv"
	"go.yaml.in/yaml/v4"
)

// TestHyperliquidExchangeIntegration runs a full trading cycle against the Hyperliquid testnet.
//
// Required environment variables (set in .env.test or shell):
//
//	HL_PRIVATE_KEY           - hex private key (master wallet or API/agent wallet)
//	HL_MASTER_WALLET_ADDRESS - master wallet address
//	HL_ORDER_SIDE            - "Buy" or "Sell"
//
// Optional:
//
//	HL_SUBACCOUNT_ADDRESS - subaccount address to trade on behalf of
//	                        (HL_PRIVATE_KEY must be the master wallet key)
//
// To run (API/agent wallet):
//
//	HL_PRIVATE_KEY=<agent_key> HL_MASTER_WALLET_ADDRESS=<master_addr> HL_ORDER_SIDE=Buy \
//	  go test ./tests/local/... -run TestHyperliquidExchangeIntegration -v -timeout 120s
//
// To run (master key with sub-account):
//
//	HL_PRIVATE_KEY=<master_key> HL_MASTER_WALLET_ADDRESS=<master_addr> HL_SUBACCOUNT_ADDRESS=<sub_addr> HL_ORDER_SIDE=Buy \
//	  go test ./tests/local/... -run TestHyperliquidExchangeIntegration -v -timeout 120s
func TestHyperliquidExchangeIntegration(t *testing.T) {
	godotenv.Load(".env.test") //nolint:errcheck

	privateKey := os.Getenv("HL_PRIVATE_KEY")
	orderSide := os.Getenv("HL_ORDER_SIDE")
	masterWalletAddress := os.Getenv("HL_MASTER_WALLET_ADDRESS")
	subaccountAddress := os.Getenv("HL_SUBACCOUNT_ADDRESS")

	if privateKey == "" || orderSide == "" || masterWalletAddress == "" {
		t.Skip("HL_PRIVATE_KEY, HL_MASTER_WALLET_ADDRESS and HL_ORDER_SIDE must be set to run this test")
	}

	closeOrderSide := utils.QuantityToSide(-float64(utils.SideToMultiplier(orderSide)))

	// Build account via YAML unmarshaling to apply defaults.
	accountYAML := `
is_testnet: true
private_key:
  from_env_var: HL_PRIVATE_KEY
master_wallet_address: ` + masterWalletAddress + `
`
	if subaccountAddress != "" {
		accountYAML += "subaccount_address: " + subaccountAddress + "\n"
	}

	var account subsettings.HyperliquidExchangeAccount
	if err := yaml.Unmarshal([]byte(accountYAML), &account); err != nil {
		t.Fatalf("unmarshal account YAML: %v", err)
	}

	cfg := subsettings.DefaultHyperliquidExchange()
	cfg.Accounts["test"] = account

	hl := clients.NewHyperliquidExchange(cfg, true)
	acc := cfg.Accounts["test"]

	// ---- 1. Settle coin balance ----
	state, err := hl.ClearinghouseState(acc)
	if err != nil {
		t.Fatalf("ClearinghouseState: %v", err)
	}
	spotState, err := hl.SpotClearinghouseState(acc)
	if err != nil {
		t.Fatalf("SpotClearinghouseState: %v", err)
	}
	equity := state.CrossMarginSummary.AccountValue.Float64()
	if equity == 0 {
		equity = state.MarginSummary.AccountValue.Float64()
	}
	var spotUSDC float64
	for _, b := range spotState.Balances {
		if b.Coin == "USDC" {
			spotUSDC = b.Total.Float64()
			break
		}
	}
	equity += spotUSDC
	t.Logf("Equity: %.4f USDC (perps=%.4f spot=%.4f)", equity, equity-spotUSDC, spotUSDC)
	if equity < 10 {
		t.Fatalf("insufficient testnet balance: %.4f USDC (need ≥ 10)", equity)
	}

	// ---- 2. Ticker ----
	const coin = "BTC"
	ctx, err := hl.Ticker(acc, coin)
	if err != nil {
		t.Fatalf("Ticker(%s): %v", coin, err)
	}
	markPrice := ctx.MarkPx.Float64()
	t.Logf("BTC mark price: %.2f", markPrice)

	// ---- 3. Symbol info ----
	symbolInfo, err := hl.SymbolInfo(acc, coin, false)
	if err != nil {
		t.Fatalf("SymbolInfo(%s): %v", coin, err)
	}
	t.Logf("BTC szDecimals=%d maxLeverage=%d", symbolInfo.SzDecimals, symbolInfo.MaxLeverage)

	qtyStep := 1.0
	for i := 0; i < symbolInfo.SzDecimals; i++ {
		qtyStep /= 10
	}
	stepPrecision := symbolInfo.SzDecimals

	// Use a small trade size: 1% of equity / markPrice, at least 1 qty step, at least $10 notional
	tradeSize := equity * 0.01 / markPrice
	minByNotional := math.Ceil(10.0/markPrice/qtyStep) * qtyStep
	if tradeSize < minByNotional {
		tradeSize = minByNotional
	}
	tradeSize = math.Round(tradeSize/qtyStep) * qtyStep
	tradeSizeStr := hlFormatFloat(tradeSize, stepPrecision)
	t.Logf("Trade size: %s BTC (%.2f USDC notional)", tradeSizeStr, tradeSize*markPrice)

	// ---- 4. Set leverage ----
	if err := hl.SetLeverage(acc, coin, 5); err != nil {
		t.Fatalf("SetLeverage: %v", err)
	}
	t.Log("SetLeverage: ok")

	// ---- 5. Place market order ----
	takeProfit := markPrice * 1.10
	stopLoss := markPrice * 0.90
	if orderSide == "Sell" {
		takeProfit = markPrice * 0.90
		stopLoss = markPrice * 1.10
	}

	data, err := hl.PlaceMarketOrder(acc, coin, orderSide, tradeSizeStr, nil, nil, false, false)
	if err != nil {
		t.Fatalf("PlaceMarketOrder: %v", err)
	}
	t.Logf("PlaceMarketOrder: oid=%s", clients.HLOrderIDFromData(data))

	// ---- 6. Positions ----
	positions, ctxs, _, err := hl.Positions(acc)
	if err != nil {
		t.Fatalf("Positions: %v", err)
	}
	var btcPos *clients.HLAssetPosition
	var btcCtx *clients.HLAssetCtx
	for i, p := range positions {
		if p.Coin == coin {
			btcPos = &positions[i]
			if i < len(ctxs) {
				btcCtx = &ctxs[i]
			}
			break
		}
	}
	if btcPos == nil {
		t.Fatalf("expected %s position after order, found none", coin)
	}
	t.Logf("Position: szi=%.6f entryPx=%.2f markPx=%.2f",
		btcPos.Szi.Float64(),
		btcPos.EntryPx.Float64(),
		func() float64 {
			if btcCtx != nil {
				return btcCtx.MarkPx.Float64()
			}
			return 0
		}())

	// ---- 7. Single position lookup ----
	posInfo, _, err := hl.Position(acc, coin)
	if err != nil {
		t.Fatalf("Position(%s): %v", coin, err)
	}
	if posInfo == nil {
		t.Fatalf("expected non-nil position for %s", coin)
	}

	// ---- 8. Set TP/SL ----
	positionSide := "Buy"
	if btcPos.Szi.Float64() < 0 {
		positionSide = "Sell"
	}
	if err := hl.SetPositionTpsl(acc, coin, positionSide, &takeProfit, &stopLoss); err != nil {
		t.Fatalf("SetPositionTpsl: %v", err)
	}
	t.Log("SetPositionTpsl: ok")

	// ---- 9. Open stop orders ----
	stopOrders, err := hl.OpenStopOrders(acc)
	if err != nil {
		t.Fatalf("OpenStopOrders: %v", err)
	}
	t.Logf("OpenStopOrders: %d orders", len(stopOrders))
	if len(stopOrders) < 2 {
		t.Logf("WARNING: expected ≥ 2 stop orders (TP+SL), got %d", len(stopOrders))
	}

	// ---- 9b. Adjust TP/SL via the adapter: old orders must be canceled ----
	exchangeCfg := subsettings.DefaultExchange()
	exchangeCfg.Hyperliquid = cfg
	exchangeType := "hyperliquid"
	exchangeAccount := subsettings.NewExchangeAccount("test", &acc, &exchangeType)
	exchange := clients.NewExchange(exchangeCfg, true, nil)

	newTakeProfit := markPrice * 1.15
	newStopLoss := markPrice * 0.85
	if orderSide == "Sell" {
		newTakeProfit = markPrice * 0.85
		newStopLoss = markPrice * 1.15
	}
	if err := exchange.SetPositionTpsl(exchangeAccount, coin, positionSide, &newTakeProfit, &newStopLoss); err != nil {
		t.Fatalf("Exchange.SetPositionTpsl: %v", err)
	}
	stopOrders, err = hl.OpenStopOrders(acc)
	if err != nil {
		t.Fatalf("OpenStopOrders after adjust: %v", err)
	}
	t.Logf("OpenStopOrders after adjust: %d orders", len(stopOrders))
	if len(stopOrders) != 2 {
		t.Errorf("expected exactly 2 stop orders after TPSL adjustment (old ones canceled), got %d", len(stopOrders))
	}

	// ---- 10. Close position ----
	sz := math.Abs(btcPos.Szi.Float64())
	szStr := hlFormatFloat(sz, stepPrecision)
	_, err = hl.PlaceMarketOrder(acc, coin, closeOrderSide, szStr, nil, nil, true, true)
	if err != nil {
		t.Fatalf("ClosePosition (PlaceMarketOrder): %v", err)
	}
	t.Log("ClosePosition: ok")

	// ---- 11. Verify empty positions ----
	positions, _, _, err = hl.Positions(acc)
	if err != nil {
		t.Fatalf("Positions after close: %v", err)
	}
	for _, p := range positions {
		if p.Coin == coin {
			t.Errorf("expected no %s position after close, but found szi=%.6f", coin, p.Szi.Float64())
		}
	}
	t.Log("Position closed successfully")
}

// TestHyperliquidExchangeSubaccount verifies that subaccount operations are routed
// to the correct address.  Requires HL_MASTER_WALLET_ADDRESS and HL_SUBACCOUNT_ADDRESS to be set.
func TestHyperliquidExchangeSubaccount(t *testing.T) {
	godotenv.Load(".env.test") //nolint:errcheck

	privateKey := os.Getenv("HL_PRIVATE_KEY")
	masterWalletAddress := os.Getenv("HL_MASTER_WALLET_ADDRESS")
	subaccountAddress := os.Getenv("HL_SUBACCOUNT_ADDRESS")

	if privateKey == "" || masterWalletAddress == "" || subaccountAddress == "" {
		t.Skip("HL_PRIVATE_KEY, HL_MASTER_WALLET_ADDRESS and HL_SUBACCOUNT_ADDRESS must be set to run this test")
	}

	accountYAML := `
is_testnet: true
private_key:
  from_env_var: HL_PRIVATE_KEY
master_wallet_address: ` + masterWalletAddress + `
subaccount_address: ` + subaccountAddress + `
`
	var account subsettings.HyperliquidExchangeAccount
	if err := yaml.Unmarshal([]byte(accountYAML), &account); err != nil {
		t.Fatalf("unmarshal account YAML: %v", err)
	}

	cfg := subsettings.DefaultHyperliquidExchange()
	cfg.Accounts["sub"] = account
	hl := clients.NewHyperliquidExchange(cfg, true)
	acc := cfg.Accounts["sub"]

	state, err := hl.ClearinghouseState(acc)
	if err != nil {
		t.Fatalf("ClearinghouseState for subaccount: %v", err)
	}

	t.Logf("Subaccount %s equity: %.4f USDC",
		subaccountAddress,
		state.CrossMarginSummary.AccountValue.Float64())
}

func hlFormatFloat(v float64, precision int) string {
	if precision <= 0 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%."+fmt.Sprintf("%d", precision)+"f", v)
}
