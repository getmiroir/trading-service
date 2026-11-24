package exchange

import (
	"fmt"
	"slices"
	"strings"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

const bybitExpectedMarginMode = "ISOLATED_MARGIN"

func probeBybitAccount(key, secret string, demo, testnet bool) subsettings.BybitAccount {
	account := subsettings.DefaultBybitAccount()
	account.IsDemo = demo
	account.IsTestnet = testnet
	account.Key = secrets.Secret{Raw: &key}
	account.Secret = secrets.Secret{Raw: &secret}
	return account
}

// ProbeBybit calls ApiKeyInfo on the prod, demo, or testnet Bybit URL and returns the raw response.
// A non-nil error means the credentials were rejected or a network error occurred.
func ProbeBybit(key, secret string, demo, testnet bool, bybitCfg subsettings.Bybit) (*clients.BybitApiKeyInfoResponse, error) {
	account := probeBybitAccount(key, secret, demo, testnet)
	return clients.NewBybit(bybitCfg, false).ApiKeyInfo(account)
}

func ensureBybitTradingSettings(client *clients.Bybit, account subsettings.BybitAccount) error {
	var violations []string

	info, err := client.AccountInfo(account)
	if err != nil {
		violations = append(violations, fmt.Sprintf("could not retrieve account margin mode: %v%s", err, tradingSettingsTutorial))
	} else if info.MarginMode != bybitExpectedMarginMode {
		currentMode := info.MarginMode
		if setErr := client.SetAccountMarginMode(account, bybitExpectedMarginMode); setErr != nil {
			violations = append(violations, fmt.Sprintf(
				"margin mode must be %s (currently %s); failed to switch automatically: %v%s",
				bybitExpectedMarginMode, currentMode, setErr, tradingSettingsTutorial,
			))
		} else {
			updatedInfo, readErr := client.AccountInfo(account)
			if readErr != nil {
				violations = append(violations, fmt.Sprintf("could not verify margin mode after switch: %v%s", readErr, tradingSettingsTutorial))
			} else if updatedInfo.MarginMode != bybitExpectedMarginMode {
				violations = append(violations, fmt.Sprintf(
					"margin mode must be %s (currently %s); close open positions and orders first%s",
					bybitExpectedMarginMode, updatedInfo.MarginMode, tradingSettingsTutorial,
				))
			}
		}
	}

	if err := client.SetAccountPositionMode(account, expectedPositionMode); err != nil {
		violations = append(violations, fmt.Sprintf(
			"failed to set position mode to %s: %v; close open positions and orders first%s",
			expectedPositionMode, err, tradingSettingsTutorial,
		))
	}

	if len(violations) > 0 {
		return formatAccountValidationError(violations)
	}
	return nil
}

// ValidateBybitAccount probes the correct URL (selected by the demo and testnet flags), checks that
// ReadOnly == 0 and that ContractTrade includes "Order" and "Position", and returns the
// normalised ApiKeyInfo. This is the single source of truth for Bybit account validation.
func ValidateBybitAccount(key, secret string, demo, testnet bool, bybitCfg subsettings.Bybit) (*clients.ApiKeyInfo, error) {
	raw, err := ProbeBybit(key, secret, demo, testnet, bybitCfg)
	if err != nil {
		return nil, fmt.Errorf("fetching API key info: %w", err)
	}

	var violations []string
	if raw.ReadOnly != 0 {
		violations = append(violations, "API key must not be read-only")
	}
	requiredContractTrade := []string{"Order", "Position"}
	var missing []string
	for _, perm := range requiredContractTrade {
		if !slices.Contains(raw.Permissions.ContractTrade, perm) {
			missing = append(missing, perm)
		}
	}
	if len(missing) > 0 {
		violations = append(violations, fmt.Sprintf("API key missing required ContractTrade permissions: %s", strings.Join(missing, ", ")))
	}
	if len(violations) > 0 {
		return nil, formatAccountValidationError(violations)
	}

	apiKeyInfo, err := clients.NewApiKeyInfoFromBybitApiKeyInfo(*raw)
	if err != nil {
		return nil, fmt.Errorf("creating API key info: %w", err)
	}

	account := probeBybitAccount(key, secret, demo, testnet)
	if err := ensureBybitTradingSettings(clients.NewBybit(bybitCfg, false), account); err != nil {
		return nil, err
	}

	return apiKeyInfo, nil
}
