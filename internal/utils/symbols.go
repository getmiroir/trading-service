package utils

import (
	"sort"
	"strings"
)

// BareCoin strips the builder dex prefix of a Hyperliquid coin ("xyz:DRAM" -> "DRAM").
func BareCoin(coin string) string {
	if i := strings.IndexByte(coin, ':'); i >= 0 {
		return coin[i+1:]
	}
	return coin
}

// DexCollisions groups the coins sharing a bare name across perp dexes (e.g. xyz:DRAM
// and io:DRAM, or BTC and xyz:BTC), keyed by bare name. Only groups of at least two
// coins are returned, each sorted.
func DexCollisions(coins []string) map[string][]string {
	byBare := make(map[string][]string, len(coins))
	for _, coin := range coins {
		bare := BareCoin(coin)
		byBare[bare] = append(byBare[bare], coin)
	}

	collisions := make(map[string][]string)
	for bare, group := range byBare {
		if len(group) > 1 {
			sort.Strings(group)
			collisions[bare] = group
		}
	}
	return collisions
}
