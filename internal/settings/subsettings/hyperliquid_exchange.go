package subsettings

import (
	"fmt"

	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"go.yaml.in/yaml/v4"
)

// HyperliquidExchange holds config for the Hyperliquid destination exchange client
// (YAML key: exchange.hyperliquid). This is separate from the Hyperliquid copy-source
// client under source.hyperliquid.
type HyperliquidExchange struct {
	Accounts   map[string]HyperliquidExchangeAccount `yaml:"accounts"`
	HttpClient HttpClient                            `yaml:"http_client"`
	Websockets Websockets                            `yaml:"websockets"`
}

func DefaultHyperliquidExchange() HyperliquidExchange {
	return HyperliquidExchange{
		Accounts:   make(map[string]HyperliquidExchangeAccount),
		HttpClient: DefaultHttpClient(),
		Websockets: DefaultWebsockets(),
	}
}

func (h *HyperliquidExchange) UnmarshalYAML(node *yaml.Node) error {
	*h = DefaultHyperliquidExchange()
	type selfAlias HyperliquidExchange
	return node.Decode((*selfAlias)(h))
}

// HyperliquidExchangeAccount holds per-account config for a Hyperliquid trading account.
// MasterWalletAddress is always the master wallet and must be set.
// SubaccountAddress is optional: when set, info queries and vault routing target the subaccount
// address instead of the master wallet (the private key must be the master wallet key).
type HyperliquidExchangeAccount struct {
	PrivateKey          secrets.Secret `yaml:"private_key"`           // hex ECDSA private key
	MasterWalletAddress string         `yaml:"master_wallet_address"` // master account address
	SubaccountAddress   *string        `yaml:"subaccount_address"`    // optional; routes actions to this subaccount
	SettleCoin          string         `yaml:"settle_coin"`
	MinimumEquity       float64        `yaml:"minimum_equity"`
	MaxLeverage         *int           `yaml:"max_leverage"`
	MaxSymbolLeverage   map[string]int `yaml:"max_symbol_leverage"`
	MaxFractionAtRisk   float64        `yaml:"max_fraction_at_risk"`
	IsTestnet           bool           `yaml:"is_testnet"`
}

func DefaultHyperliquidExchangeAccount() HyperliquidExchangeAccount {
	return HyperliquidExchangeAccount{
		SettleCoin:        "USDC",
		MinimumEquity:     0,
		MaxLeverage:       nil,
		MaxSymbolLeverage: nil,
		MaxFractionAtRisk: 1,
		IsTestnet:         false,
	}
}

func (h *HyperliquidExchangeAccount) UnmarshalYAML(node *yaml.Node) error {
	*h = DefaultHyperliquidExchangeAccount()
	type selfAlias HyperliquidExchangeAccount
	return node.Decode((*selfAlias)(h))
}

func (h *HyperliquidExchangeAccount) ActiveWalletAddress() string {
	if h.SubaccountAddress != nil {
		return *h.SubaccountAddress
	}
	return h.MasterWalletAddress
}

func HyperliquidAccountID(isTestnet bool, activeAddress string) string {
	if isTestnet {
		return fmt.Sprintf("hyperliquid-testnet-%s", activeAddress)
	}
	return fmt.Sprintf("hyperliquid-%s", activeAddress)
}
