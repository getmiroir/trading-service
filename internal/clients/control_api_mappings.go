package clients

import (
	"fmt"
	"math"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// MappingStatus represents the status of a copy mapping
type MappingStatus string

const (
	MappingStatusOK             MappingStatus = "ok"
	MappingStatusError          MappingStatus = "error"
	MappingStatusEmergencyClose MappingStatus = "emergency_close"
	MappingStatusVersionBlocked MappingStatus = "version_blocked"
)

// CopyMapping represents a mapping between an Exchange account and a copy target
type CopyMapping struct {
	Name    string
	Account subsettings.ExchangeAccount
	Target  CopyTarget
	Status  MappingStatus
}

// MinimumEquity returns the maximum of target and account minimum equity
func (cm *CopyMapping) MinimumEquity() (float64, error) {
	exchangeAccount := cm.Account
	var minimumEquity float64
	switch *exchangeAccount.ExchangeType {
	case "bybit":
		minimumEquity = exchangeAccount.Account.(*subsettings.BybitAccount).MinimumEquity
	case "binance":
		minimumEquity = exchangeAccount.Account.(*subsettings.BinanceAccount).MinimumEquity
	case "hyperliquid":
		minimumEquity = exchangeAccount.Account.(*subsettings.HyperliquidExchangeAccount).MinimumEquity
	default:
		return 0, fmt.Errorf("invalid exchange type: %s", *exchangeAccount.ExchangeType)
	}

	targetMin := 0.0
	if cm.Target.MinimumEquity != nil {
		targetMin = *cm.Target.MinimumEquity
	}
	if targetMin > minimumEquity {
		return targetMin, nil
	}
	return minimumEquity, nil
}

// ComputeLeverage returns the leverage for a given coin, taking into account overrides.
// If the account has no ComputeLeverage set (nil), the target leverage is returned uncapped.
// Account LeverageOverrides only affect the cap; target LeverageOverrides are always applied.
func (cm *CopyMapping) ComputeLeverage(symbol string, targetPosition *HyperliquidPosition) (int, error) {

	// Override always take priority
	if override, ok := cm.Target.LeverageOverrides[targetPosition.Coin]; ok {
		return override, nil
	}

	exchangeAccount := cm.Account
	var accountMaxLeverage *int
	var accountMaxLeverageOverrides map[string]int
	switch *exchangeAccount.ExchangeType {
	case "bybit":
		accountMaxLeverage = exchangeAccount.Account.(*subsettings.BybitAccount).MaxLeverage
		accountMaxLeverageOverrides = exchangeAccount.Account.(*subsettings.BybitAccount).MaxSymbolLeverage
	case "binance":
		accountMaxLeverage = exchangeAccount.Account.(*subsettings.BinanceAccount).MaxLeverage
		accountMaxLeverageOverrides = exchangeAccount.Account.(*subsettings.BinanceAccount).MaxSymbolLeverage
	case "hyperliquid":
		accountMaxLeverage = exchangeAccount.Account.(*subsettings.HyperliquidExchangeAccount).MaxLeverage
		accountMaxLeverageOverrides = exchangeAccount.Account.(*subsettings.HyperliquidExchangeAccount).MaxSymbolLeverage
	default:
		return 1, fmt.Errorf("invalid exchange type: %s", *exchangeAccount.ExchangeType)
	}

	// Compute the effective account cap (only when account leverage is configured).
	var accountCap *int
	if accountMaxLeverage != nil {
		cap := *accountMaxLeverage
		if accountMaxLeverageOverrides != nil {
			if override, ok := accountMaxLeverageOverrides[symbol]; ok {
				cap = override
			}
		}
		accountCap = &cap
	}

	effectiveLeverage := int(math.Ceil(targetPosition.Leverage.Value * cm.Target.ExposureFactor))

	if accountCap != nil && *accountCap < effectiveLeverage {
		effectiveLeverage = *accountCap
	}
	if cm.Target.MaximumLeverage != nil && *cm.Target.MaximumLeverage < effectiveLeverage {
		effectiveLeverage = *cm.Target.MaximumLeverage
	}
	if effectiveLeverage < 1 {
		return 1, nil
	}
	return effectiveLeverage, nil
}
