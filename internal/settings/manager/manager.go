package manager

import (
	"fmt"
	"log/slog"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// Manager provides methods to read and modify settings with automatic persistence.
type Manager struct {
	settings    *settings.Settings
	configPath  string
	bybit       exchangeAccountManager
	binance     exchangeAccountManager
	hyperliquid *hyperliquidAccountManager
	logger      *slog.Logger
}

// New creates a Manager by loading settings from the given config path.
func New(configPath string, settings *settings.Settings) (*Manager, error) {

	m := &Manager{settings: settings, configPath: configPath, logger: logging.NewLogger("settings-manager")}
	m.bybit = &bybitAccountManager{m: m}
	m.binance = &binanceAccountManager{m: m}
	m.hyperliquid = &hyperliquidAccountManager{m: m}

	return m, nil
}

// Settings returns the underlying settings.
func (m *Manager) Settings() *settings.Settings {
	return m.settings
}

// GetLoginURL returns the configured login page URL.
func (m *Manager) GetLoginURL() string {
	return m.settings.CopyTrading.Auth.LoginURL
}

// ListAllAccounts returns accounts from all configured exchanges with their exchange type.
func (m *Manager) ListAllAccounts() []AccountInfo {
	bybit := m.bybit.ListAccounts()
	binance := m.binance.ListAccounts()
	hyperliquid := m.hyperliquid.ListAccounts()
	all := make([]AccountInfo, 0, len(bybit)+len(binance)+len(hyperliquid))
	all = append(all, bybit...)
	all = append(all, binance...)
	all = append(all, hyperliquid...)
	return all
}

// AddAccount adds an account to the appropriate exchange (bybit or binance).
func (m *Manager) AddAccount(exchangeType ExchangeType, params ExchangeAccountParams, replace bool) error {
	if !m.settings.CopyTrading.Features.IsClientEnabled(exchangeType) {
		return fmt.Errorf("adding %s accounts from the GUI is not available in this build", exchangeType)
	}
	if exchangeType == "bybit" {
		return m.bybit.AddAccount(params, replace)
	}
	if exchangeType == "binance" {
		return m.binance.AddAccount(params, replace)
	}
	return fmt.Errorf("invalid exchange type: %s", exchangeType)
}

// GetAccountParams returns the parameters for a bybit or binance account by ID.
func (m *Manager) GetAccountParams(exchangeType ExchangeType, accountID string) (ExchangeAccountParams, error) {
	if exchangeType == "bybit" {
		return m.bybit.GetAccountParams(accountID)
	}
	if exchangeType == "binance" {
		return m.binance.GetAccountParams(accountID)
	}
	return ExchangeAccountParams{}, fmt.Errorf("invalid exchange type: %s", exchangeType)
}

// AddHyperliquidAccount adds a new Hyperliquid account.
func (m *Manager) AddHyperliquidAccount(params HyperliquidAccountParams, replace bool) error {
	if !m.settings.CopyTrading.Features.IsClientEnabled(subsettings.ClientHyperliquid) {
		return fmt.Errorf("adding Hyperliquid accounts from the GUI is not available in this build")
	}
	return m.hyperliquid.AddAccount(params, replace)
}

// GetHyperliquidAccountParams returns the parameters for a Hyperliquid account.
func (m *Manager) GetHyperliquidAccountParams(accountID string) (HyperliquidAccountParams, error) {
	return m.hyperliquid.GetAccountParams(accountID)
}

// EditHyperliquidAccount updates credentials for an existing Hyperliquid account.
func (m *Manager) EditHyperliquidAccount(accountID string, params HyperliquidAccountParams) error {
	return m.hyperliquid.EditAccount(accountID, params)
}

// IsAccountMirroring checks whether the given account has an active copy target
// with mirroring enabled. Returns true when the global copy toggle is on AND
// the account has an enabled target entry.
func (m *Manager) IsAccountMirroring(accountID string) (bool, error) {
	cfg := m.settings.CopyTrading
	controlApi, err := clients.NewControlApi(cfg.ControlApi, cfg.Exchange.Accounts())
	if err != nil {
		return false, fmt.Errorf("creating control API client: %w", err)
	}

	mappings, err := controlApi.GetMappings()
	if err != nil {
		return false, fmt.Errorf("fetching copy mappings: %w", err)
	}

	for _, mapping := range mappings {
		if mapping.Name == accountID && mapping.Target.Enabled {
			return true, nil
		}
	}

	return false, nil
}

// RemoveAccount removes an account from whichever exchange owns the given ID.
// Account IDs are prefixed with the exchange name (e.g. "bybit-123", "binance-456").
func (m *Manager) RemoveAccount(accountID string) error {
	if err := m.bybit.RemoveAccount(accountID); err == nil {
		return nil
	}
	if err := m.binance.RemoveAccount(accountID); err == nil {
		return nil
	}
	return m.hyperliquid.RemoveAccount(accountID)
}

// EditAccount updates credentials for an existing bybit or binance account.
func (m *Manager) EditAccount(exchangeType string, params ExchangeAccountParams) error {
	if exchangeType == "bybit" {
		return m.bybit.AddAccount(params, true)
	}
	if exchangeType == "binance" {
		return m.binance.AddAccount(params, true)
	}
	return fmt.Errorf("invalid exchange type: %s", exchangeType)
}

func (m *Manager) save() error {
	return settings.SaveSettings(m.settings)
}
