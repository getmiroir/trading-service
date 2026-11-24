package runners

import (
	"reflect"
	"sort"
	"testing"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

func newTestAccountSync(globallyExcluded, targetExcluded []string) *AccountSyncForMapping {
	cfg := subsettings.DefaultService()
	cfg.HyperExcludedSymbols = globallyExcluded
	return &AccountSyncForMapping{
		mapping: clients.CopyMapping{Target: clients.CopyTarget{ExcludedSymbols: targetExcluded}},
		logger:  logging.NewLogger("test"),
		cfg:     &cfg,
	}
}

func positionsOf(coins ...string) []*clients.HyperliquidPosition {
	positions := make([]*clients.HyperliquidPosition, len(coins))
	for i, coin := range coins {
		positions[i] = &clients.HyperliquidPosition{Coin: coin}
	}
	return positions
}

func coinsOf(positions []*clients.HyperliquidPosition) []string {
	coins := make([]string, len(positions))
	for i, pos := range positions {
		coins[i] = pos.Coin
	}
	return coins
}

func collisionCoinsOf(t *testing.T, metrics []metric_types.SymbolMetrics) []string {
	t.Helper()
	coins := make([]string, len(metrics))
	for i, m := range metrics {
		if len(m.Errors) != 1 || m.Errors[0].ErrorCode != internal.SymbolDexCollisionErrorCode {
			t.Fatalf("expected a single collision error, got %+v", m.Errors)
		}
		if m.Symbol != m.TargetSymbol {
			t.Fatalf("expected Symbol == TargetSymbol, got %q / %q", m.Symbol, m.TargetSymbol)
		}
		coins[i] = m.Symbol
	}
	sort.Strings(coins)
	return coins
}

func TestNonExcludedTargetPositionsDropsGlobalAndTargetExclusions(t *testing.T) {
	asm := newTestAccountSync([]string{"PURR"}, []string{"xyz:DRAM"})
	positions := []clients.HyperliquidPosition{{Coin: "BTC"}, {Coin: "PURR"}, {Coin: "xyz:DRAM"}, {Coin: "io:DRAM"}}

	kept := asm.nonExcludedTargetPositions(positions)

	if got := coinsOf(kept); !reflect.DeepEqual(got, []string{"BTC", "io:DRAM"}) {
		t.Fatalf("kept %v", got)
	}
}

func TestWithoutDexCollisionsDropsEveryCollidingCoin(t *testing.T) {
	asm := newTestAccountSync(nil, nil)

	kept, metrics := asm.withoutDexCollisions(positionsOf("BTC", "xyz:BTC", "xyz:DRAM", "io:DRAM", "io:ANTH"), "USDT")

	if got := coinsOf(kept); !reflect.DeepEqual(got, []string{"io:ANTH"}) {
		t.Fatalf("kept %v", got)
	}
	want := []string{"BTC", "io:DRAM", "xyz:BTC", "xyz:DRAM"}
	if got := collisionCoinsOf(t, metrics); !reflect.DeepEqual(got, want) {
		t.Fatalf("collision metrics for %v, want %v", got, want)
	}
}

func TestWithoutDexCollisionsAfterUserExcludedOneSide(t *testing.T) {
	asm := newTestAccountSync(nil, []string{"xyz:DRAM"})
	positions := []clients.HyperliquidPosition{{Coin: "xyz:DRAM"}, {Coin: "io:DRAM"}}

	kept, metrics := asm.withoutDexCollisions(asm.nonExcludedTargetPositions(positions), "USDT")

	if got := coinsOf(kept); !reflect.DeepEqual(got, []string{"io:DRAM"}) {
		t.Fatalf("kept %v", got)
	}
	if len(metrics) != 0 {
		t.Fatalf("expected no collision, got %+v", metrics)
	}
}

func TestWithoutResolutionCollisionsDropsCoinsSharingAnExchangeSymbol(t *testing.T) {
	asm := newTestAccountSync(nil, nil)
	positions := positionsOf("xyz:GOLD", "io:XAU", "BTC")
	gold, xau, btc := positions[0], positions[1], positions[2]
	resolved := map[string][]*clients.HyperliquidPosition{
		"XAUUSDT": {gold, xau},
		"BTCUSDT": {btc},
	}

	tradeable, metrics := asm.withoutResolutionCollisions(resolved, "USDT")

	if len(tradeable) != 1 || tradeable["BTCUSDT"] != btc {
		t.Fatalf("tradeable %v", tradeable)
	}
	want := []string{"io:XAU", "xyz:GOLD"}
	if got := collisionCoinsOf(t, metrics); !reflect.DeepEqual(got, want) {
		t.Fatalf("collision metrics for %v, want %v", got, want)
	}
}
