package subsettings

import (
	"fmt"

	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"go.yaml.in/yaml/v4"
)

// Bybit client
type Bybit struct {
	Accounts        map[string]BybitAccount `yaml:"accounts"`
	ReceiveWindowMs int                     `yaml:"receive_window_ms"`
	HttpClient      HttpClient              `yaml:"http_client"`
	Websockets      Websockets              `yaml:"websockets"`
}

func DefaultBybit() Bybit {
	return Bybit{
		Accounts:        make(map[string]BybitAccount),
		ReceiveWindowMs: 5000,
		HttpClient:      DefaultHttpClient(),
		Websockets:      DefaultWebsockets(),
	}
}

func (b *Bybit) UnmarshalYAML(node *yaml.Node) error {
	*b = DefaultBybit()
	type selfAlias Bybit
	return node.Decode((*selfAlias)(b))
}

// Bybit users in Turkey and Kazakhstan must use regional websocket endpoints
// https://bybit-exchange.github.io/docs/v5/websocket/trade/guideline
type BybitRegion string

const (
	BybitRegionGlobal     BybitRegion = ""
	BybitRegionTurkey     BybitRegion = "tr"
	BybitRegionKazakhstan BybitRegion = "kz"
)

func (r BybitRegion) Valid() bool {
	switch r {
	case BybitRegionGlobal, BybitRegionTurkey, BybitRegionKazakhstan:
		return true
	default:
		return false
	}
}

// Bybit account
type BybitAccount struct {
	Key               secrets.Secret `yaml:"key"`
	Secret            secrets.Secret `yaml:"secret"`
	SettleCoin        string         `yaml:"settle_coin"`
	MinimumEquity     float64        `yaml:"minimum_equity"`
	MaxLeverage       *int           `yaml:"max_leverage"`
	MaxSymbolLeverage map[string]int `yaml:"max_symbol_leverage"`
	MaxFractionAtRisk float64        `yaml:"max_fraction_at_risk"`
	IsDemo            bool           `yaml:"is_demo"`
	IsTestnet         bool           `yaml:"is_testnet"`
	Region            BybitRegion    `yaml:"region"`
}

func DefaultBybitAccount() BybitAccount {
	return BybitAccount{
		SettleCoin:        "USDT",
		MinimumEquity:     0,
		MaxLeverage:       nil,
		MaxSymbolLeverage: nil,
		MaxFractionAtRisk: 1,
		IsDemo:            false,
	}
}

func (b *BybitAccount) UnmarshalYAML(node *yaml.Node) error {
	*b = DefaultBybitAccount()
	type selfAlias BybitAccount
	if err := node.Decode((*selfAlias)(b)); err != nil {
		return err
	}
	if !b.Region.Valid() {
		return fmt.Errorf("invalid bybit region %q (supported: tr, kz)", b.Region)
	}
	if b.IsTestnet {
		b.IsDemo = true
	}
	return nil
}
