package exchange

import (
	"fmt"
	"strings"
	"time"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// ValidateHyperliquidExchangeAccount validates a Hyperliquid account:
//   - the private key maps to a known API wallet
//   - the API key has not expired
//   - if a subaccount is configured, it exists and its master matches the configured master wallet
func ValidateHyperliquidExchangeAccount(account subsettings.HyperliquidExchangeAccount, cfg subsettings.HyperliquidExchange) error {
	c := clients.NewHyperliquidExchange(cfg, false)

	agent, err := c.ApiKeyInfo(account)
	if err != nil {
		return fmt.Errorf("invalid private key for wallet address: %w", err)
	}

	if agent.ValidUntil > 0 && agent.ValidUntil < time.Now().UnixMilli() {
		return fmt.Errorf("API key expired at %s", time.UnixMilli(agent.ValidUntil).UTC().Format(time.RFC3339))
	}

	if account.SubaccountAddress != nil {
		subaccounts, err := c.Subaccounts(account)
		if err != nil {
			return fmt.Errorf("could not fetch subaccounts: %w", err)
		}
		var found bool
		for _, s := range subaccounts {
			if strings.EqualFold(s.SubaccountUser, *account.SubaccountAddress) {
				found = true
				if !strings.EqualFold(s.Master, account.MasterWalletAddress) {
					return fmt.Errorf("subaccount %s master is %s, expected %s", *account.SubaccountAddress, s.Master, account.MasterWalletAddress)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("subaccount %s not found under master wallet %s", *account.SubaccountAddress, account.MasterWalletAddress)
		}
	}

	return nil
}
