package utils

import (
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
)

// MergeSymbolMetrics folds pre-built metrics (unmapped or colliding target coins) into
// the synced metrics sharing their Symbol. The backend stores one row per symbol and
// loop, so a second metric for the same symbol would overwrite the first one. The
// synced metric stays the base: only the errors and an empty TargetSymbol are taken
// from the pre-built one.
func MergeSymbolMetrics(synced, prebuilt []metric_types.SymbolMetrics) []metric_types.SymbolMetrics {
	merged := synced
	indexBySymbol := make(map[string]int, len(synced)+len(prebuilt))
	for i, m := range merged {
		indexBySymbol[m.Symbol] = i
	}

	for _, extra := range prebuilt {
		i, ok := indexBySymbol[extra.Symbol]
		if !ok {
			indexBySymbol[extra.Symbol] = len(merged)
			merged = append(merged, extra)
			continue
		}
		merged[i].Errors = append(merged[i].Errors, extra.Errors...)
		if merged[i].TargetSymbol == "" {
			merged[i].TargetSymbol = extra.TargetSymbol
		}
	}
	return merged
}
