package subsettings

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// Exchange client
type Exchange struct {
	Bybit       Bybit               `yaml:"bybit"`
	Binance     Binance             `yaml:"binance"`
	Hyperliquid HyperliquidExchange `yaml:"hyperliquid"`
}

func DefaultExchange() Exchange {
	return Exchange{
		Bybit:       DefaultBybit(),
		Binance:     DefaultBinance(),
		Hyperliquid: DefaultHyperliquidExchange(),
	}
}

func (e *Exchange) Accounts() map[string]ExchangeAccount {
	accounts := make(map[string]ExchangeAccount)
	for accountName, account := range e.Bybit.Accounts {
		accountType := "bybit"
		accounts[accountName] = NewExchangeAccount(accountName, &account, &accountType)
	}
	for accountName, account := range e.Binance.Accounts {
		accountType := "binance"
		accounts[accountName] = NewExchangeAccount(accountName, &account, &accountType)
	}
	for accountName, account := range e.Hyperliquid.Accounts {
		accountType := "hyperliquid"
		accounts[accountName] = NewExchangeAccount(accountName, &account, &accountType)
	}
	return accounts
}

func (e *Exchange) UnmarshalYAML(node *yaml.Node) error {
	*e = DefaultExchange()
	type selfAlias Exchange
	return node.Decode((*selfAlias)(e))
}

type ExchangeAccount struct {
	Name         string      `yaml:"-"`
	ExchangeType *string     `yaml:"exchange_type"` // bybit, binance
	Account      interface{} `yaml:"account"`
}

func NewExchangeAccount(accountName string, account interface{}, exchangeType *string) ExchangeAccount {
	return ExchangeAccount{
		Name:         accountName,
		ExchangeType: exchangeType,
		Account:      account,
	}
}

func (e *ExchangeAccount) MaxFractionAtRisk() (float64, error) {
	switch *e.ExchangeType {
	case "bybit":
		return e.Account.(*BybitAccount).MaxFractionAtRisk, nil
	case "binance":
		return e.Account.(*BinanceAccount).MaxFractionAtRisk, nil
	case "hyperliquid":
		return e.Account.(*HyperliquidExchangeAccount).MaxFractionAtRisk, nil
	default:
		return 0, fmt.Errorf("invalid exchange type: %s", *e.ExchangeType)
	}
}

func (e *ExchangeAccount) IsHyperliquid() bool {
	return e.ExchangeType != nil && *e.ExchangeType == "hyperliquid"
}

func (e *ExchangeAccount) IsDemo() (bool, error) {
	switch *e.ExchangeType {
	case "bybit":
		return e.Account.(*BybitAccount).IsDemo, nil
	case "binance":
		return e.Account.(*BinanceAccount).IsDemo, nil
	case "hyperliquid":
		return e.Account.(*HyperliquidExchangeAccount).IsTestnet, nil
	default:
		return false, fmt.Errorf("invalid exchange type: %s", *e.ExchangeType)
	}
}

func (e *ExchangeAccount) SettleCoin() (string, error) {
	switch *e.ExchangeType {
	case "bybit":
		return e.Account.(*BybitAccount).SettleCoin, nil
	case "binance":
		return e.Account.(*BinanceAccount).SettleCoin, nil
	case "hyperliquid":
		return e.Account.(*HyperliquidExchangeAccount).SettleCoin, nil
	default:
		return "", fmt.Errorf("invalid exchange type: %s", *e.ExchangeType)
	}
}
