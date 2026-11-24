package exchange

import (
	"fmt"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

func probeBinanceAccount(key, secret string, demo bool) subsettings.BinanceAccount {
	account := subsettings.DefaultBinanceAccount()
	account.IsDemo = demo
	account.Key = secrets.Secret{Raw: &key}
	account.Secret = secrets.Secret{Raw: &secret}
	return account
}

// ProbeBinanceProd calls ApiKeyPermissions and SpotAccountInfo against the production Binance
// URLs. Demo keys will return an auth error; this asymmetry is used for the is-live consistency
// check.
func ProbeBinanceProd(key, secret string, binanceCfg subsettings.Binance) (*clients.BinanceApiKeyPermissions, *clients.BinanceSpotAccountInfo, error) {
	account := probeBinanceAccount(key, secret, false)
	c := clients.NewBinance(binanceCfg, false)
	perms, err := c.ApiKeyPermissions(account)
	if err != nil {
		return nil, nil, err
	}
	info, err := c.SpotAccountInfo(account, false)
	if err != nil {
		return nil, nil, err
	}
	return perms, info, nil
}

func ensureBinancePositionMode(client *clients.Binance, account subsettings.BinanceAccount) error {
	mode, err := client.AccountPositionMode(account)
	if err != nil {
		return formatAccountValidationError([]string{
			fmt.Sprintf("could not retrieve position mode: %v%s", err, tradingSettingsTutorial),
		})
	}

	if mode == expectedPositionMode {
		return nil
	}

	if setErr := client.SetAccountPositionMode(account, expectedPositionMode); setErr != nil {
		return formatAccountValidationError([]string{
			fmt.Sprintf(
				"position mode must be %s (currently %s); failed to switch automatically: %v; close open positions and orders first%s",
				expectedPositionMode, mode, setErr, tradingSettingsTutorial,
			),
		})
	}

	updatedMode, readErr := client.AccountPositionMode(account)
	if readErr != nil {
		return formatAccountValidationError([]string{
			fmt.Sprintf("could not verify position mode after switch: %v%s", readErr, tradingSettingsTutorial),
		})
	}
	if updatedMode != expectedPositionMode {
		return formatAccountValidationError([]string{
			fmt.Sprintf(
				"position mode must be %s (currently %s); close open positions and orders first%s",
				expectedPositionMode, updatedMode, tradingSettingsTutorial,
			),
		})
	}

	return nil
}

// ValidateBinanceAccount validates credentials against the correct URL for the given demo flag.
// For production accounts: calls ProbeBinanceProd and requires EnableFutures and EnableReading.
// For demo accounts: skips the prod-only permission endpoints.
// This is the single source of truth for Binance account validation.
func ValidateBinanceAccount(key, secret string, demo bool, binanceCfg subsettings.Binance) (*clients.ApiKeyInfo, error) {
	var spotApiKeyInfo *clients.BinanceApiKeyPermissions
	var spotAccountInfo *clients.BinanceSpotAccountInfo

	if !demo {
		var err error
		spotApiKeyInfo, spotAccountInfo, err = ProbeBinanceProd(key, secret, binanceCfg)
		if err != nil {
			return nil, fmt.Errorf("validating binance credentials: %w", err)
		}
		if !spotApiKeyInfo.EnableFutures {
			return nil, fmt.Errorf("futures trading is not enabled for this API key")
		}
		if !spotApiKeyInfo.EnableReading {
			return nil, fmt.Errorf("reading is not enabled for this API key")
		}
	}

	account := probeBinanceAccount(key, secret, demo)
	if err := ensureBinancePositionMode(clients.NewBinance(binanceCfg, false), account); err != nil {
		return nil, err
	}

	apiKeyInfo := clients.NewApiKeyInfoFromBinanceApiKeyInfo(spotApiKeyInfo, spotAccountInfo)
	return &apiKeyInfo, nil
}
