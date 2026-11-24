package clients

import (
	"sync"
	"testing"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

func TestHlNonceSeq_Monotonic(t *testing.T) {
	seq := &hlNonceSeq{}

	prev := seq.next()
	for i := 0; i < 100; i++ {
		next := seq.next()
		if next <= prev {
			t.Fatalf("nonce not increasing: prev=%d next=%d", prev, next)
		}
		prev = next
	}
}

func TestHlNonceSeq_ConcurrentUnique(t *testing.T) {
	seq := &hlNonceSeq{}
	const n = 200

	var wg sync.WaitGroup
	wg.Add(n)
	seen := make(chan int64, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			seen <- seq.next()
		}()
	}
	wg.Wait()
	close(seen)

	unique := make(map[int64]struct{}, n)
	for nonce := range seen {
		if _, dup := unique[nonce]; dup {
			t.Fatalf("duplicate nonce %d", nonce)
		}
		unique[nonce] = struct{}{}
	}
	if len(unique) != n {
		t.Fatalf("got %d unique nonces, want %d", len(unique), n)
	}
}

func TestHyperliquidExchange_NextActionNoncePerAccount(t *testing.T) {
	h := &HyperliquidExchange{nonceSeqs: make(map[string]*hlNonceSeq)}
	acc := subsettings.HyperliquidExchangeAccount{
		MasterWalletAddress: "0xabc",
	}

	n1, err := h.nextActionNonce(acc)
	if err != nil {
		t.Fatalf("nextActionNonce: %v", err)
	}
	n2, err := h.nextActionNonce(acc)
	if err != nil {
		t.Fatalf("nextActionNonce: %v", err)
	}
	if n2 <= n1 {
		t.Fatalf("nonces not increasing for same account: %d then %d", n1, n2)
	}

	other := subsettings.HyperliquidExchangeAccount{
		MasterWalletAddress: "0xdef",
	}
	n3, err := h.nextActionNonce(other)
	if err != nil {
		t.Fatalf("nextActionNonce other: %v", err)
	}
	if n3 <= 0 {
		t.Fatalf("expected positive nonce for other account, got %d", n3)
	}
}
