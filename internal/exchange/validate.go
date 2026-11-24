package exchange

import (
	"fmt"

	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// ValidateAccounts checks every configured exchange account before the copy loop starts.
// For each account it:
//  1. Runs the full credential + permission validation against the URL matching is_demo
//  2. For demo accounts, additionally probes the prod URL and asserts it fails — this
//     catches the case where production credentials are misconfigured with is_demo: true.
func ValidateAccounts(cfg *settings.Settings) error {
	for name, acc := range cfg.CopyTrading.Exchange.Bybit.Accounts {
		if err := checkBybitAccount(name, acc, cfg.CopyTrading.Exchange.Bybit); err != nil {
			return err
		}
	}
	for name, acc := range cfg.CopyTrading.Exchange.Binance.Accounts {
		if err := checkBinanceAccount(name, acc, cfg.CopyTrading.Exchange.Binance); err != nil {
			return err
		}
	}
	for name, acc := range cfg.CopyTrading.Exchange.Hyperliquid.Accounts {
		if err := checkHyperliquidExchangeAccount(name, acc, cfg.CopyTrading.Exchange.Hyperliquid); err != nil {
			return err
		}
	}
	return nil
}

func checkBybitAccount(name string, acc subsettings.BybitAccount, bybitCfg subsettings.Bybit) error {
	key, err := acc.Key.Value()
	if err != nil {
		return fmt.Errorf("bybit/%s: resolving API key: %w", name, err)
	}
	secret, err := acc.Secret.Value()
	if err != nil {
		return fmt.Errorf("bybit/%s: resolving API secret: %w", name, err)
	}

	// Step 1 — full credential + permission validation on the correct URL.
	if _, err := ValidateBybitAccount(key, secret, acc.IsDemo, acc.IsTestnet, bybitCfg); err != nil {
		return fmt.Errorf("bybit/%s: %w", name, err)
	}

	// Step 2 — is-live consistency check for demo accounts.
	// The prod probe must fail: a demo key is rejected by api.bybit.com.
	// For prod accounts (is_demo=false) this is already implicit in step 1 succeeding.
	if acc.IsDemo || acc.IsTestnet {
		if _, prodErr := ProbeBybit(key, secret, false, false, bybitCfg); prodErr == nil {
			return fmt.Errorf("bybit/%s: is_demo=true but credentials succeed on prod URL — prod key configured as demo?", name)
		}
	}

	return nil
}

func checkHyperliquidExchangeAccount(name string, acc subsettings.HyperliquidExchangeAccount, hlCfg subsettings.HyperliquidExchange) error {
	if _, err := acc.PrivateKey.Value(); err != nil {
		return fmt.Errorf("hyperliquid/%s: resolving private key: %w", name, err)
	}
	// There are no validation for isDemo in hyperliquid. Anyway, the isTestNet FORCE the testnet url.
	if err := ValidateHyperliquidExchangeAccount(acc, hlCfg); err != nil {
		return fmt.Errorf("hyperliquid/%s: %w", name, err)
	}

	return nil
}

func checkBinanceAccount(name string, acc subsettings.BinanceAccount, binanceCfg subsettings.Binance) error {
	key, err := acc.Key.Value()
	if err != nil {
		return fmt.Errorf("binance/%s: resolving API key: %w", name, err)
	}
	secret, err := acc.Secret.Value()
	if err != nil {
		return fmt.Errorf("binance/%s: resolving API secret: %w", name, err)
	}

	// Step 1 — full credential + permission validation on the correct URL.
	if _, err := ValidateBinanceAccount(key, secret, acc.IsDemo, binanceCfg); err != nil {
		return fmt.Errorf("binance/%s: %w", name, err)
	}

	// Step 2 — is-live consistency check for demo accounts.
	// ProbeBinanceProd always hits the production endpoints; demo keys will be rejected.
	// For prod accounts (is_demo=false) this is already implicit in step 1 succeeding.
	if acc.IsDemo {
		if _, _, prodErr := ProbeBinanceProd(key, secret, binanceCfg); prodErr == nil {
			return fmt.Errorf("binance/%s: is_demo=true but credentials succeed on prod URL — prod key configured as demo?", name)
		}
	}

	return nil
}
