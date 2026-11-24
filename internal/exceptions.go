package internal

import (
	"fmt"
)

// Allowed error codes
type SyncErrorCode int

const (
	UnknownErrorCode                                     SyncErrorCode = -1
	DefaultErrorCode                                     SyncErrorCode = 1000
	LoopNewControlApiErrorCode                           SyncErrorCode = 2000
	LoopGetMappingsErrorCode                             SyncErrorCode = 2001
	LoopSyncAccountsErrorCode                            SyncErrorCode = 2002
	AccountPositionsRetrievalErrorCode                   SyncErrorCode = 3000
	AccountOpenStopOrdersRetrievalErrorCode              SyncErrorCode = 3001
	AccountApiKeyInfoRetrievalErrorCode                  SyncErrorCode = 3002
	AccountBalanceRetrievalErrorCode                     SyncErrorCode = 3003
	AccountBalanceInvalidAccountTypeErrorCode            SyncErrorCode = 3020
	AccountInvalidApiKeyOrPermissionsErrorCode           SyncErrorCode = 3021
	AccountApiKeyNotSetErrorCode                         SyncErrorCode = 3022
	AccountTargetPositionsRetrievalErrorCode             SyncErrorCode = 3004
	AccountInfoRetrievalErrorCode                        SyncErrorCode = 3005
	AccountNormalizationRetrievalErrorCode               SyncErrorCode = 3006
	AccountBalanceBelowMinimumEquityErrorCode            SyncErrorCode = 3007
	AccountBalanceAboveMaximumEquityErrorCode            SyncErrorCode = 3019
	AccountMarginModeErrorCode                           SyncErrorCode = 3008
	AccountPositionModeErrorCode                         SyncErrorCode = 3009
	AccountStatusErrorCode                               SyncErrorCode = 3010
	AccountSetLeverageErrorCode                          SyncErrorCode = 3011
	AccountMarginTypeErrorCode                           SyncErrorCode = 3012
	AccountSyncSymbolsErrorCode                          SyncErrorCode = 3013
	AccountEmergencyCloseErrorCode                       SyncErrorCode = 3014
	AccountVersionBlockedErrorCode                       SyncErrorCode = 3015
	AccountTargetAccountValueRetrievalErrorCode          SyncErrorCode = 3016
	AccountTargetUserAbstractionDexAbstractionErrorCode  SyncErrorCode = 3017
	AccountTargetUserAbstractionUnsupportedModeErrorCode SyncErrorCode = 3018
	SymbolNotEnoughMarginErrorCode                       SyncErrorCode = 4000
	SymbolClosePositionErrorCode                         SyncErrorCode = 4001
	SymbolComputePricePointsErrorCode                    SyncErrorCode = 4002
	SymbolSetPositionTpslErrorCode                       SyncErrorCode = 4003
	SymbolSymbolInfoErrorCode                            SyncErrorCode = 4004
	SymbolTickerErrorCode                                SyncErrorCode = 4005
	SymbolPlaceOrderErrorCode                            SyncErrorCode = 4006
	SymbolAdjustLeverageErrorCode                        SyncErrorCode = 4007
	SymbolNotExistErrorCode                              SyncErrorCode = 4008
	SymbolOrderSizeBelowMinErrorCode                     SyncErrorCode = 4009
	SymbolOrderValueBelowMinErrorCode                    SyncErrorCode = 4010
	SymbolMachineInternalClockErrorCode                  SyncErrorCode = 4011
	SymbolTradFiAgreementRequiredErrorCode               SyncErrorCode = 4012
	SymbolPositionExceedsMarginTierErrorCode             SyncErrorCode = 4013
	SymbolOrderNotPlaceableErrorCode                     SyncErrorCode = 4014
	SymbolLeverageExceedsMaxErrorCode                    SyncErrorCode = 4015
	SymbolSubaccountLeverageRestrictedErrorCode          SyncErrorCode = 4016
	SymbolNewFuturesAccountLeverageRestrictedErrorCode   SyncErrorCode = 4017
	SymbolOpenInterestCapErrorCode                       SyncErrorCode = 4018
	SymbolNotionalSafetyErrorCode                        SyncErrorCode = 4019
	SymbolDexCollisionErrorCode                          SyncErrorCode = 4020
)

// SyncError is the base error for sync operations
type SyncError struct {
	Message   string
	ErrorCode SyncErrorCode
	FromError *error
}

func (e *SyncError) Error() string {
	return fmt.Sprintf("Error: %s (code: %d)", e.Message, e.ErrorCode)
}

// AccountSyncError represents an error during account synchronization
type AccountSyncError struct {
	AccountName string
	Message     string
	ErrorCode   SyncErrorCode
	FromError   *error
}

func (e *AccountSyncError) Error() string {
	return fmt.Sprintf("Error syncing account %s: %s", e.AccountName, e.Message)
}

// SymbolSyncError represents an error during symbol synchronization
type SymbolSyncError struct {
	AccountName string
	Symbol      string
	Message     string
	FromError   *error
	ErrorCode   SyncErrorCode
}

func (e *SymbolSyncError) Error() string {
	return fmt.Sprintf("Error syncing symbol %s for account %s: %s", e.Symbol, e.AccountName, e.Message)
}

func (e *SymbolSyncError) IsClosedSymbolError() bool {
	switch fromError := (*e.FromError).(type) {
	case *BybitBadReturnCodeError:
		if fromError.RetCode == 110074 {
			return true
		}
	case *BinanceBadStatusCodeError:
		if fromError.ErrorCode != nil && *fromError.ErrorCode == -4141 {
			return true
		}
	default:
		return false
	}
	return false
}

// BybitError represents a generic Bybit API error
type BybitError struct {
	Message string
}

func (e *BybitError) Error() string {
	return fmt.Sprintf("Bybit error: %s", e.Message)
}

// BybitBadReturnCodeError represents a Bybit API error with a return code
type BybitBadReturnCodeError struct {
	Message string
	RetCode int
}

func (e *BybitBadReturnCodeError) Error() string {
	return fmt.Sprintf("Bybit API bad return code %d: %s", e.RetCode, e.Message)
}

// BinanceError represents a generic Binance API error
type BinanceError struct {
	Message string
}

func (e *BinanceError) Error() string {
	return fmt.Sprintf("Binance error: %s", e.Message)
}

// BinanceBadStatusCodeError represents a Binance API error with a status code
type BinanceBadStatusCodeError struct {
	Message      string
	StatusCode   int
	ErrorCode    *int
	ErrorMessage *string
}

func (e *BinanceBadStatusCodeError) Error() string {
	if e.ErrorCode != nil && e.ErrorMessage != nil {
		return fmt.Sprintf("Binance API bad status code %d: %s\nError code: %d\nError message: %s",
			e.StatusCode, e.Message, *e.ErrorCode, *e.ErrorMessage)
	} else if e.ErrorCode != nil {
		return fmt.Sprintf("Binance API bad status code %d: %s\nError code: %d",
			e.StatusCode, e.Message, *e.ErrorCode)
	} else if e.ErrorMessage != nil {
		return fmt.Sprintf("Binance API bad status code %d: %s\nError message: %s",
			e.StatusCode, e.Message, *e.ErrorMessage)
	}
	return fmt.Sprintf("Binance API bad status code %d: %s", e.StatusCode, e.Message)
}

// HyperliquidBadStatusError represents a Hyperliquid API error with status code
type HyperliquidBadStatusError struct {
	Status int
}

func (e *HyperliquidBadStatusError) Error() string {
	return fmt.Sprintf("Hyperliquid API bad status: %d", e.Status)
}

// HyperliquidExchangeError represents a Hyperliquid exchange API error (status:"err")
type HyperliquidExchangeError struct {
	Message string
}

func (e *HyperliquidExchangeError) Error() string {
	return fmt.Sprintf("Hyperliquid exchange error: %s", e.Message)
}

// LimiterException represents a rate limiter exception
type LimiterException struct {
	Message string
}

func (e *LimiterException) Error() string {
	return e.Message
}

// GatheredErrorsError represents multiple errors gathered from concurrent operations
type GatheredErrorsError struct {
	Errors []error
}

func (e *GatheredErrorsError) Error() string {
	if len(e.Errors) == 0 {
		return "gathered errors: (no errors)"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("gathered error: %v", e.Errors[0])
	}
	msg := fmt.Sprintf("gathered %d errors:", len(e.Errors))
	for i, err := range e.Errors {
		msg += fmt.Sprintf("\n  [%d] %v", i, err)
	}
	return msg
}
