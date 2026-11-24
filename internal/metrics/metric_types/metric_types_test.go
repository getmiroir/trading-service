package metric_types

import (
	"encoding/json"
	"testing"
)

func TestSymbolMetricsMarshalUsesTargetSymbol(t *testing.T) {
	raw, err := json.Marshal(SymbolMetrics{
		Symbol:       "BTCUSDT",
		TargetSymbol: "BTC",
		Size:         1,
	})
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}

	if payload["targetSymbol"] != "BTC" {
		t.Fatalf("targetSymbol=%v, want BTC", payload["targetSymbol"])
	}
	if _, ok := payload["hyperliquidSymbol"]; ok {
		t.Fatalf("hyperliquidSymbol should not be emitted after backend-first deploy")
	}
}
