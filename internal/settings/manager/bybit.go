package manager

import (
	"fmt"
	"time"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/exchange"
	"github.com/getmiroir/trading-service/internal/keyring"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

type bybitAccountManager struct {
	m *Manager
}

// validateKeyPermissions validates the API key permissions required for copy trading.
func (b *bybitAccountManager) validateKeyPermissions(params ExchangeAccountParams) (*clients.ApiKeyInfo, error) {
	return exchange.ValidateBybitAccount(
		params.APIKey, params.APISecret, params.Demo, params.Testnet,
		b.m.settings.CopyTrading.Exchange.Bybit,
	)
}

// AddAccount adds a new Bybit account, stores credentials in the keyring, and saves settings.
// If replace is true and the account already exists, it replaces it instead of returning an error.
func (b *bybitAccountManager) AddAccount(params ExchangeAccountParams, replace bool) error {
	// check api key and api secret are not empty
	if params.APIKey == "" || params.APISecret == "" {
		return fmt.Errorf("API key and secret are required")
	}
	// check minimum equity is not negative
	if params.MinimumEquity < 0 {
		return fmt.Errorf("minimum equity must be non-negative")
	}

	apiKeyInfo, err := b.validateKeyPermissions(params)
	if err != nil {
		return err
	}

	if apiKeyInfo.ExpiredAt != nil && time.Now().Unix() > *apiKeyInfo.ExpiredAt {
		return fmt.Errorf("API key has expired")
	}

	accountID := fmt.Sprintf("bybit-%d", *apiKeyInfo.UserID)
	oldAccount, exists := b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID]
	if exists && !replace {
		return fmt.Errorf("bybit account %q already exists", accountID)
	}

	secretEntry := fmt.Sprintf("%s-secret", accountID)
	keyEntry := fmt.Sprintf("%s-key", accountID)

	if err := keyring.Set(keyEntry, params.APIKey); err != nil {
		return fmt.Errorf("storing API key in keyring: %w", err)
	}
	if err := keyring.Set(secretEntry, params.APISecret); err != nil {
		_ = keyring.Delete(keyEntry)
		return fmt.Errorf("storing API secret in keyring: %w", err)
	}

	region := subsettings.BybitRegion(params.Region)
	if !region.Valid() {
		return fmt.Errorf("invalid bybit region %q (supported: global, tr, kz)", params.Region)
	}

	if params.Testnet && !params.Demo {
		return fmt.Errorf("testnet accounts must be demo accounts")
	}

	account := subsettings.DefaultBybitAccount()
	account.IsDemo = params.Demo
	account.Key = secrets.Secret{FromKeyring: &keyEntry}
	account.Secret = secrets.Secret{FromKeyring: &secretEntry}
	account.MinimumEquity = params.MinimumEquity
	account.Region = region

	b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID] = account

	if err := b.m.save(); err != nil {
		if exists {
			b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID] = oldAccount
		} else {
			delete(b.m.settings.CopyTrading.Exchange.Bybit.Accounts, accountID)
		}
		_ = keyring.Delete(keyEntry)
		_ = keyring.Delete(secretEntry)
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

// RemoveAccount removes a Bybit account, deletes its keyring entries, and saves settings.
func (b *bybitAccountManager) RemoveAccount(accountID string) error {
	account, exists := b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID]
	if !exists {
		return fmt.Errorf("bybit account %q not found", accountID)
	}

	if account.Key.FromKeyring != nil {
		_ = keyring.Delete(*account.Key.FromKeyring)
	}
	if account.Secret.FromKeyring != nil {
		_ = keyring.Delete(*account.Secret.FromKeyring)
	}

	delete(b.m.settings.CopyTrading.Exchange.Bybit.Accounts, accountID)

	if err := b.m.save(); err != nil {
		b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID] = account
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

// GetAccount returns a single Bybit account wrapped as an ExchangeAccount.
func (b *bybitAccountManager) GetAccountParams(accountID string) (ExchangeAccountParams, error) {
	acc, ok := b.m.settings.CopyTrading.Exchange.Bybit.Accounts[accountID]
	if !ok {
		return ExchangeAccountParams{}, fmt.Errorf("bybit account %q not found", accountID)
	}
	key, err := acc.Key.Value()
	if err != nil {
		return ExchangeAccountParams{}, fmt.Errorf("getting API key: %w", err)
	}
	secret, err := acc.Secret.Value()
	if err != nil {
		return ExchangeAccountParams{}, fmt.Errorf("getting API secret: %w", err)
	}
	return ExchangeAccountParams{
		Demo:          acc.IsDemo,
		Testnet:       acc.IsTestnet,
		APIKey:        key,
		APISecret:     secret,
		MinimumEquity: acc.MinimumEquity,
		Region:        string(acc.Region),
	}, nil
}

// ListAccounts returns info for all Bybit accounts.
func (b *bybitAccountManager) ListAccounts() []AccountInfo {
	accounts := b.m.settings.CopyTrading.Exchange.Bybit.Accounts
	infos := make([]AccountInfo, 0, len(accounts))
	for accountID, account := range accounts {
		infos = append(infos, AccountInfo{ID: accountID, ExchangeType: ExchangeTypeBybit, Demo: account.IsDemo})
	}
	return infos
}
