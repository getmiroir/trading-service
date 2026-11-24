package subsettings

import (
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"go.yaml.in/yaml/v4"
)

// Binance client
type Binance struct {
	Accounts        map[string]BinanceAccount `yaml:"accounts"`
	ReceiveWindowMs int                       `yaml:"receive_window_ms"`
	HttpClient      HttpClient                `yaml:"http_client"`
	Websockets      Websockets                `yaml:"websockets"`
}

func DefaultBinance() Binance {
	return Binance{
		Accounts:        make(map[string]BinanceAccount),
		ReceiveWindowMs: 5000,
		HttpClient:      DefaultHttpClient(),
		Websockets:      DefaultWebsockets(),
	}
}

func (b *Binance) UnmarshalYAML(node *yaml.Node) error {
	*b = DefaultBinance()
	type selfAlias Binance
	return node.Decode((*selfAlias)(b))
}

// Binance account
type BinanceAccount struct {
	Key               secrets.Secret `yaml:"key"`
	Secret            secrets.Secret `yaml:"secret"`
	SettleCoin        string         `yaml:"settle_coin"`
	MinimumEquity     float64        `yaml:"minimum_equity"`
	MaxLeverage       *int           `yaml:"max_leverage"`
	MaxSymbolLeverage map[string]int `yaml:"max_symbol_leverage"`
	MaxFractionAtRisk float64        `yaml:"max_fraction_at_risk"`
	IsDemo            bool           `yaml:"is_demo"`
}

func DefaultBinanceAccount() BinanceAccount {
	return BinanceAccount{
		SettleCoin:        "USDT",
		MinimumEquity:     0,
		MaxLeverage:       nil,
		MaxSymbolLeverage: nil,
		MaxFractionAtRisk: 1,
		IsDemo:            false,
	}
}

func (b *BinanceAccount) UnmarshalYAML(node *yaml.Node) error {
	*b = DefaultBinanceAccount()
	type selfAlias BinanceAccount
	return node.Decode((*selfAlias)(b))
}
