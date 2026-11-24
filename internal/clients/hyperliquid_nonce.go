package clients

import (
	"fmt"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// hlNonceSeq produces strictly increasing millisecond nonces for Hyperliquid
// exchange actions. Hyperliquid rejects duplicate nonces for the same user.
type hlNonceSeq struct {
	mu   sync.Mutex
	last int64
}

func (s *hlNonceSeq) next() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	if now <= s.last {
		s.last++
	} else {
		s.last = now
	}
	return s.last
}

func hlAccountUserAddress(account subsettings.HyperliquidExchangeAccount) (string, error) {
	if account.SubaccountAddress != nil {
		return *account.SubaccountAddress, nil
	}
	if account.MasterWalletAddress == "" {
		return "", fmt.Errorf("userAddress: master_wallet_address is not set")
	}
	return account.MasterWalletAddress, nil
}

func (h *HyperliquidExchange) nextActionNonce(account subsettings.HyperliquidExchangeAccount) (int64, error) {
	addr, err := hlAccountUserAddress(account)
	if err != nil {
		return 0, err
	}

	h.nonceMu.Lock()
	seq, ok := h.nonceSeqs[addr]
	if !ok {
		seq = &hlNonceSeq{}
		h.nonceSeqs[addr] = seq
	}
	h.nonceMu.Unlock()

	return seq.next(), nil
}
