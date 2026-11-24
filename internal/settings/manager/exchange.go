package manager

import (
	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// ExchangeAccountParams holds the parameters for adding or updating an exchange account.
// For updates, empty APIKey/APISecret means "keep existing keyring values".
type ExchangeAccountParams struct {
	Demo          bool
	Testnet       bool // Bybit only: routes to testnet endpoints; develop builds only
	APIKey        string
	APISecret     string
	MinimumEquity float64
	Region        string // Bybit only: empty for global, "tr" for Turkey, "kz" for Kazakhstan
}

// ExchangeType identifies a supported exchange. It is an alias of
// subsettings.ClientType so the manager, the settings, and the GUI share a
// single set of client identifiers.
type ExchangeType = subsettings.ClientType

const (
	ExchangeTypeBybit       = subsettings.ClientBybit
	ExchangeTypeBinance     = subsettings.ClientBinance
	ExchangeTypeHyperliquid = subsettings.ClientHyperliquid
)

// AccountInfo holds an account ID, its exchange type, and whether it is a
// demo account (testnet for Hyperliquid).
type AccountInfo struct {
	ID           string
	ExchangeType ExchangeType
	Demo         bool
}

// exchangeAccountManager is the interface for managing exchange accounts of a single exchange type.
type exchangeAccountManager interface {
	validateKeyPermissions(params ExchangeAccountParams) (*clients.ApiKeyInfo, error)
	AddAccount(params ExchangeAccountParams, replace bool) error
	RemoveAccount(accountID string) error
	GetAccountParams(accountID string) (ExchangeAccountParams, error)
	ListAccounts() []AccountInfo
}
