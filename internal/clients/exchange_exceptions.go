package clients

import (
	"errors"
	"fmt"
)

// Sentinel errors — use with errors.Is()
var (
	ErrNotEnoughMargin                     = errors.New("not enough margin")
	ErrSymbolNotExist                      = errors.New("symbol not exist")
	ErrMachineInternalClock                = errors.New("machine internal clock drifted, sync system time")
	ErrTradFiAgreementRequired             = errors.New("TradFi-Perps agreement required")
	ErrPositionExceedsMarginTier           = errors.New("position exceeds margin tier limit at current leverage")
	ErrSubaccountLeverageRestricted        = errors.New("subaccount leverage restricted")
	ErrNewFuturesAccountLeverageRestricted = errors.New("new futures account leverage restricted")
	ErrInvalidApiKeyOrPermissions          = errors.New("invalid API key, IP, or permissions")
	ErrApiKeyNotSet                        = errors.New("API key or secret is not set")
	ErrOpenInterestCap                     = errors.New("asset is at its open interest cap")
)

// ExchangeError wraps an exchange-specific error with a normalized kind.
type ExchangeError struct {
	Kind  error // one of the sentinel errors above
	Cause error // original error from the client
}

func (e *ExchangeError) Error() string {
	return fmt.Sprintf("[%v]: %v", e.Kind, e.Cause)
}

// Unwrap allows errors.Is() / errors.As() to walk both the kind and cause. see https://go.dev/blog/go1.13-errors
func (e *ExchangeError) Unwrap() []error {
	return []error{e.Kind, e.Cause}
}

func newExchangeError(kind, cause error) *ExchangeError {
	return &ExchangeError{Kind: kind, Cause: cause}
}
