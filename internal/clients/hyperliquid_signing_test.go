package clients

import (
	"strings"
	"testing"
)

func hlWireFractionalDigits(s string) int {
	if i := strings.Index(s, "."); i >= 0 {
		return len(s) - i - 1
	}
	return 0
}
func TestHlFormatPerpPrice_PUMPBuyCloseSlippage(t *testing.T) {
	// PUMP szDecimals=0 → max 6 decimal places.
	// Closing a short at entry 0.001641 with 5% buy slippage.
	markPrice := 0.001641
	limitPx := markPrice * (1 + hlMarketOrderSlippage)

	formatted := hlFormatPerpPrice(limitPx, 0)
	wire := hlPriceToWire(limitPx, 0)

	if wire == "0.0017231" {
		t.Fatalf("old invalid wire price %q should be truncated", wire)
	}
	if hlWireFractionalDigits(wire) > 6 {
		t.Fatalf("wire price %q has %d decimals, want <= 6", wire, hlWireFractionalDigits(wire))
	}
	if formatted != 0.001723 {
		t.Fatalf("formatted price = %v, want 0.001723", formatted)
	}
	if wire != "0.001723" {
		t.Fatalf("wire price = %q, want 0.001723", wire)
	}
}

func TestHlFormatPerpPrice_BTCSzDecimals5(t *testing.T) {
	// BTC szDecimals=5 → max 1 decimal place.
	px := 97000.55
	formatted := hlFormatPerpPrice(px, 5)
	wire := hlPriceToWire(px, 5)

	if hlWireFractionalDigits(wire) > 1 {
		t.Fatalf("wire price %q has %d decimals, want <= 1", wire, hlWireFractionalDigits(wire))
	}
	if formatted != 97001 {
		t.Fatalf("formatted price = %v, want 97001", formatted)
	}
}

func TestHlFormatPerpPrice_Above100kInteger(t *testing.T) {
	px := 123456.7
	formatted := hlFormatPerpPrice(px, 0)
	if formatted != 123457 {
		t.Fatalf("formatted price = %v, want 123457", formatted)
	}
}

func TestHlFormatPerpPrice_SDKExample(t *testing.T) {
	// From hyperliquid-python-sdk examples/rounding.py with szDecimals=1.
	px := 1.2345678
	szDecimals := 1
	formatted := hlFormatPerpPrice(px, szDecimals)
	if formatted != 1.2346 {
		t.Fatalf("formatted price = %v, want 1.2346", formatted)
	}
}
