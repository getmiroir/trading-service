package clients

import (
	"errors"
	"fmt"
)

// HyperliquidDexAbstractionModeError is returned when the target uses legacy dexAbstraction mode.
type HyperliquidDexAbstractionModeError struct{}

func (e *HyperliquidDexAbstractionModeError) Error() string {
	return "hyperliquid user abstraction mode dexAbstraction is not supported"
}

// HyperliquidUnsupportedUserAbstractionModeError is returned for unknown user abstraction modes.
type HyperliquidUnsupportedUserAbstractionModeError struct {
	Mode string
}

func (e *HyperliquidUnsupportedUserAbstractionModeError) Error() string {
	return fmt.Sprintf("unsupported hyperliquid user abstraction mode: %s", e.Mode)
}

// HyperliquidPortfolioWindowError is returned when the required portfolio window or history is missing.
type HyperliquidPortfolioWindowError struct {
	Window string
}

func (e *HyperliquidPortfolioWindowError) Error() string {
	return fmt.Sprintf("portfolio window %q missing or has no account value history", e.Window)
}

// HyperliquidInvalidAccountTypeError is returned when the account abstraction type is not
// supported for balance retrieval (only unifiedAccount is supported).
type HyperliquidInvalidAccountTypeError struct {
	AccountType string
}

func (e *HyperliquidInvalidAccountTypeError) Error() string {
	return fmt.Sprintf("invalid hyperliquid account type for balance: %s", e.AccountType)
}

// IsHyperliquidInvalidAccountTypeError reports whether err is an invalid account type error.
func IsHyperliquidInvalidAccountTypeError(err error) bool {
	var target *HyperliquidInvalidAccountTypeError
	return errors.As(err, &target)
}

// IsHyperliquidDexAbstractionModeError reports whether err is a dexAbstraction mode error.
func IsHyperliquidDexAbstractionModeError(err error) bool {
	var target *HyperliquidDexAbstractionModeError
	return errors.As(err, &target)
}

// IsHyperliquidUnsupportedUserAbstractionModeError reports whether err is an unsupported mode error.
func IsHyperliquidUnsupportedUserAbstractionModeError(err error) bool {
	var target *HyperliquidUnsupportedUserAbstractionModeError
	return errors.As(err, &target)
}

// IsHyperliquidPortfolioWindowError reports whether err is a missing portfolio window error.
func IsHyperliquidPortfolioWindowError(err error) bool {
	var target *HyperliquidPortfolioWindowError
	return errors.As(err, &target)
}
