package utils

import (
	"reflect"
	"testing"
)

func TestBareCoin(t *testing.T) {
	cases := map[string]string{"xyz:DRAM": "DRAM", "io:ANTH": "ANTH", "BTC": "BTC"}
	for coin, want := range cases {
		if got := BareCoin(coin); got != want {
			t.Errorf("BareCoin(%q) = %q, want %q", coin, got, want)
		}
	}
}

func TestDexCollisionsGroupsSameBareNameAcrossDexes(t *testing.T) {
	got := DexCollisions([]string{"io:DRAM", "ETH", "xyz:BTC", "xyz:DRAM", "BTC", "io:ANTH"})

	want := map[string][]string{
		"DRAM": {"io:DRAM", "xyz:DRAM"},
		"BTC":  {"BTC", "xyz:BTC"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DexCollisions = %v, want %v", got, want)
	}
}

func TestDexCollisionsWithoutSharedNames(t *testing.T) {
	if got := DexCollisions([]string{"BTC", "xyz:TSLA", "io:ANTH"}); len(got) != 0 {
		t.Fatalf("expected no collision, got %v", got)
	}
}
