package manager

import (
	"fmt"

	"github.com/getmiroir/trading-service/internal/exchange"
	"github.com/getmiroir/trading-service/internal/keyring"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// HyperliquidAccountParams holds the parameters for adding or updating a Hyperliquid account.
type HyperliquidAccountParams struct {
	IsTestNet           bool
	PrivateKey          string
	MasterWalletAddress string
	SubaccountAddress   *string
	MinimumEquity       float64
}

type hyperliquidAccountManager struct {
	m *Manager
}

func (h *hyperliquidAccountManager) createAccount(params HyperliquidAccountParams) subsettings.HyperliquidExchangeAccount {
	account := subsettings.DefaultHyperliquidExchangeAccount()
	account.PrivateKey = secrets.Secret{Raw: &params.PrivateKey}
	account.MasterWalletAddress = params.MasterWalletAddress
	account.SubaccountAddress = params.SubaccountAddress
	account.MinimumEquity = params.MinimumEquity
	if params.IsTestNet {
		account.IsTestnet = true
	}
	return account
}

// AddAccount adds a new Hyperliquid account, stores the private key in the keyring, and saves settings.
// If replace is true and the account already exists, it replaces it instead of returning an error.
func (h *hyperliquidAccountManager) AddAccount(params HyperliquidAccountParams, replace bool) error {
	if params.PrivateKey == "" {
		return fmt.Errorf("private key is required")
	}
	if params.MinimumEquity < 0 {
		return fmt.Errorf("minimum equity must be non-negative")
	}
	account := h.createAccount(params)
	err := exchange.ValidateHyperliquidExchangeAccount(account, h.m.settings.CopyTrading.Exchange.Hyperliquid)
	if err != nil {
		return err
	}

	activeAddress := account.ActiveWalletAddress()
	accountID := subsettings.HyperliquidAccountID(account.IsTestnet, activeAddress)
	oldAccount, exists := h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID]
	if exists && !replace {
		return fmt.Errorf("hyperliquid account %q already exists", accountID)
	}

	keyEntry := fmt.Sprintf("%s-key", accountID)
	if err := keyring.Set(keyEntry, params.PrivateKey); err != nil {
		return fmt.Errorf("storing private key in keyring: %w", err)
	}
	account.PrivateKey = secrets.Secret{FromKeyring: &keyEntry}

	h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID] = account

	if err := h.m.save(); err != nil {
		if exists {
			h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID] = oldAccount
		} else {
			delete(h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts, accountID)
		}
		_ = keyring.Delete(keyEntry)
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

// EditAccount updates an existing Hyperliquid account. When testnet toggles change the account ID,
// the old settings entry and keyring secret are removed after the new one is saved.
func (h *hyperliquidAccountManager) EditAccount(accountID string, params HyperliquidAccountParams) error {
	account := h.createAccount(params)
	newID := subsettings.HyperliquidAccountID(account.IsTestnet, account.ActiveWalletAddress())

	if err := h.AddAccount(params, true); err != nil {
		return err
	}
	if accountID != newID {
		return h.RemoveAccount(accountID)
	}
	return nil
}

// RemoveAccount removes a Hyperliquid account, deletes its keyring entry, and saves settings.
func (h *hyperliquidAccountManager) RemoveAccount(accountID string) error {
	account, exists := h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID]
	if !exists {
		return fmt.Errorf("hyperliquid account %q not found", accountID)
	}

	if account.PrivateKey.FromKeyring != nil {
		_ = keyring.Delete(*account.PrivateKey.FromKeyring)
	}

	delete(h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts, accountID)

	if err := h.m.save(); err != nil {
		h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID] = account
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

// GetAccountParams returns the parameters for a Hyperliquid account.
func (h *hyperliquidAccountManager) GetAccountParams(accountID string) (HyperliquidAccountParams, error) {
	acc, ok := h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts[accountID]
	if !ok {
		return HyperliquidAccountParams{}, fmt.Errorf("hyperliquid account %q not found", accountID)
	}
	privateKey, err := acc.PrivateKey.Value()
	if err != nil {
		return HyperliquidAccountParams{}, fmt.Errorf("getting private key: %w", err)
	}
	return HyperliquidAccountParams{
		IsTestNet:           acc.IsTestnet,
		PrivateKey:          privateKey,
		MasterWalletAddress: acc.MasterWalletAddress,
		SubaccountAddress:   acc.SubaccountAddress,
		MinimumEquity:       acc.MinimumEquity,
	}, nil
}

// ListAccounts returns info for all Hyperliquid accounts. Testnet accounts
// are reported as demo.
func (h *hyperliquidAccountManager) ListAccounts() []AccountInfo {
	accounts := h.m.settings.CopyTrading.Exchange.Hyperliquid.Accounts
	infos := make([]AccountInfo, 0, len(accounts))
	for accountID, account := range accounts {
		infos = append(infos, AccountInfo{ID: accountID, ExchangeType: ExchangeTypeHyperliquid, Demo: account.IsTestnet})
	}
	return infos
}
