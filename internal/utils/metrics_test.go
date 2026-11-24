package utils

import (
	"testing"

	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
)

func TestMergeSymbolMetricsKeepsTheSyncedMetricAsBase(t *testing.T) {
	closeErr := metric_types.SymbolError{Exception: "close failed", ErrorCode: 4001}
	collision := metric_types.SymbolError{Exception: "collision", ErrorCode: 4020}
	synced := []metric_types.SymbolMetrics{
		{Symbol: "xyz:DRAM", Size: 12, PnL: 8.5, EntryPrice: 28, Errors: []metric_types.SymbolError{closeErr}},
	}
	prebuilt := []metric_types.SymbolMetrics{
		{Symbol: "xyz:DRAM", TargetSymbol: "xyz:DRAM", Errors: []metric_types.SymbolError{collision}},
	}

	merged := MergeSymbolMetrics(synced, prebuilt)

	if len(merged) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(merged))
	}
	m := merged[0]
	if m.Size != 12 || m.PnL != 8.5 || m.EntryPrice != 28 {
		t.Fatalf("synced values were lost: %+v", m)
	}
	if m.TargetSymbol != "xyz:DRAM" {
		t.Fatalf("expected TargetSymbol xyz:DRAM, got %q", m.TargetSymbol)
	}
	if len(m.Errors) != 2 || m.Errors[0] != closeErr || m.Errors[1] != collision {
		t.Fatalf("expected close error then collision, got %+v", m.Errors)
	}
}

func TestMergeSymbolMetricsAppendsMetricsOfOtherSymbols(t *testing.T) {
	synced := []metric_types.SymbolMetrics{{Symbol: "DRAMUSDT"}}
	prebuilt := []metric_types.SymbolMetrics{
		{Symbol: "xyz:DRAM", TargetSymbol: "xyz:DRAM"},
		{Symbol: "io:DRAM", TargetSymbol: "io:DRAM"},
	}

	merged := MergeSymbolMetrics(synced, prebuilt)

	if len(merged) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(merged))
	}
	if merged[1].Symbol != "xyz:DRAM" || merged[2].Symbol != "io:DRAM" {
		t.Fatalf("unexpected order: %+v", merged)
	}
}

func TestMergeSymbolMetricsKeepsAStoredTargetSymbol(t *testing.T) {
	synced := []metric_types.SymbolMetrics{{Symbol: "BTC", TargetSymbol: "BTC"}}
	prebuilt := []metric_types.SymbolMetrics{{Symbol: "BTC", TargetSymbol: "other"}}

	merged := MergeSymbolMetrics(synced, prebuilt)

	if merged[0].TargetSymbol != "BTC" {
		t.Fatalf("expected TargetSymbol BTC, got %q", merged[0].TargetSymbol)
	}
}
