package runners

import (
	"reflect"
	"testing"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// skhynixInfo mirrors Bybit SKHYNIXUSDT, whose step-rounded order sizes once
// fell below the minimum notional after passing the raw-size check.
var skhynixInfo = &clients.SymbolInfo{
	MinOrderQty:      0.001,
	MaxMkOrderQty:    1000,
	QtyStep:          0.001,
	MinNotionalValue: 5,
}

const skhynixPrice = 888.86

func newTestSymbolSync() *SymbolSync {
	cfg := subsettings.DefaultService()
	cfg.MinNotionalSafetyFactor = 1.1 // min notional 5 * 1.1 = 5.5
	return &SymbolSync{cfg: &cfg}
}

func TestFractionOrderSizesChecksMinimumNotionalAfterRounding(t *testing.T) {
	s := newTestSymbolSync()
	// 0.0066 * 888.86 = 5.86 clears 5.5, but the order rounds down to 0.006 = 5.33.
	if got := s.fractionOrderSizes(0.00659651525782224, skhynixPrice, skhynixInfo); len(got) != 0 {
		t.Fatalf("expected no valid order, got %v", got)
	}
	if got := s.fractionOrderSizes(0.0075, skhynixPrice, skhynixInfo); !reflect.DeepEqual(got, []string{"0.007"}) {
		t.Fatalf("expected [0.007], got %v", got)
	}
}

func TestOrderBelowExchangeMinimumUsesTheRoundedSize(t *testing.T) {
	s := newTestSymbolSync()
	cases := []struct {
		size  float64
		below bool
	}{
		{0.0005, true},              // below min quantity
		{0.006, true},               // 5.33 below min notional
		{0.00659651525782224, true}, // 5.86 raw, 5.33 once rounded
		{0.0075, false},             // rounds to 0.007 = 6.22
	}
	for _, c := range cases {
		if got := s.orderBelowExchangeMinimum(c.size, skhynixPrice, skhynixInfo); got != c.below {
			t.Errorf("orderBelowExchangeMinimum(%v) = %v, want %v", c.size, got, c.below)
		}
	}
}

// Any size the gate lets through must yield at least one order in placeOrder,
// otherwise the sync fails with SymbolOrderNotPlaceableErrorCode.
func TestOrderBelowExchangeMinimumAgreesWithFractioning(t *testing.T) {
	s := newTestSymbolSync()
	for size := 0.0001; size < 0.02; size += 0.00001 {
		below := s.orderBelowExchangeMinimum(size, skhynixPrice, skhynixInfo)
		fractions := s.fractionOrderSizes(size, skhynixPrice, skhynixInfo)
		if below != (len(fractions) == 0) {
			t.Fatalf("size %v: gate below=%v but fractions=%v", size, below, fractions)
		}
	}
}
