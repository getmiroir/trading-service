package runners

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
)

// AccountPositionsShouldBeClosedError indicates that account positions should be closed
// Moved here to avoid import cycle (internal -> clients -> internal)
type AccountPositionsShouldBeClosedError struct {
	AccountName string
	Message     string
	Positions   []clients.Position // Exchange positions
	Immediate   bool
	ErrorCode   internal.SyncErrorCode
}

func (e *AccountPositionsShouldBeClosedError) Error() string {
	return fmt.Sprintf("Error syncing account %s: %s", e.AccountName, e.Message)
}

func GetAccountSyncErrorCode(err error) internal.SyncErrorCode {
	if accountSyncError, ok := err.(*internal.AccountSyncError); ok {
		return accountSyncError.ErrorCode
	}
	if accountPositionsShouldBeClosedError, ok := err.(*AccountPositionsShouldBeClosedError); ok {
		return accountPositionsShouldBeClosedError.ErrorCode
	}
	return internal.UnknownErrorCode
}

// AccountSync handles synchronization of an account
type AccountSync struct {
	copyLoop           *CopyLoop
	account            subsettings.ExchangeAccount
	accountName        string
	closeAllSignalTime *time.Time
	logger             *slog.Logger

	// Hyperliquid request-rate-limit usage is polled infrequently (every
	// Service.HyperliquidRateLimitPollSeconds) and cached here across loops so
	// each loop's metrics can carry the latest known value without re-polling.
	rateLimitMu      sync.Mutex
	cachedRateLimit  *clients.RateLimitInfo
	rateLimitFetched time.Time
}

// NewAccountSync creates a new AccountSync instance
func NewAccountSync(copyLoop *CopyLoop, accountName string, account subsettings.ExchangeAccount) *AccountSync {
	logger := logging.NewLogger("account-sync").With("account", accountName)
	return &AccountSync{
		copyLoop:           copyLoop,
		account:            account,
		accountName:        accountName,
		closeAllSignalTime: nil,
		logger:             logger,
	}
}

// Sync synchronizes the account with the given mapping
func (as *AccountSync) Sync(mapping clients.CopyMapping, cfg *subsettings.Service) (*metric_types.AccountMetrics, error) {
	receivedCloseAllSignal := false
	defer func() {
		if !receivedCloseAllSignal {
			as.closeAllSignalTime = nil
		}
	}()

	// Add a random sleep between 0 and account sync max sleep seconds
	sleepTime := rand.Float64() * cfg.AccountSyncMaxSleepSeconds
	time.Sleep(time.Duration(sleepTime * float64(time.Second)))

	accountSyncForMapping := NewAccountSyncForMapping(as, as.copyLoop, mapping, as.closeAllSignalTime, cfg)

	metrics, err := accountSyncForMapping.Sync()
	if err != nil {
		if closeErr, ok := err.(*AccountPositionsShouldBeClosedError); ok {
			receivedCloseAllSignal = true

			currentTime := time.Now()
			if as.closeAllSignalTime == nil {
				as.closeAllSignalTime = &currentTime
			}

			if closeErr.Immediate {
				as.logger.Error("Received exception signalling immediate full position close")
				if err := as.CloseAllPositions(closeErr.Positions); err != nil {
					return metrics, err
				}
				return metrics, closeErr
			}

			timeSinceSignal := currentTime.Sub(*as.closeAllSignalTime).Seconds()
			if timeSinceSignal > cfg.CloseAllMarginSeconds {
				as.logger.Error("Received exception signalling full position close",
					"time_since_signal", timeSinceSignal,
					"threshold", cfg.CloseAllMarginSeconds)
				if err := as.CloseAllPositions(closeErr.Positions); err != nil {
					return metrics, err
				}
				return metrics, closeErr
			}

			as.logger.Error("Received exception signalling full position close, waiting",
				"wait_remaining", cfg.CloseAllMarginSeconds-timeSinceSignal)
			return metrics, closeErr
		}
		return metrics, err
	}

	return metrics, nil
}

// CloseAllPositions closes all positions
func (as *AccountSync) CloseAllPositions(positions []clients.Position) error {
	if len(positions) == 0 {
		as.logger.Info("No positions to close")
		return nil
	}

	symbols := make([]string, len(positions))
	for i, p := range positions {
		symbols[i] = p.Symbol
	}
	as.logger.Warn("Closing all positions", "symbols", symbols)

	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	for _, position := range positions {
		wg.Add(1)
		go func(pos clients.Position) {
			defer wg.Done()
			err := as.copyLoop.Exchange.ClosePosition(as.account, pos)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(position)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("errors closing positions: %v", errors)
	}

	as.logger.Warn("Closed all positions")
	return nil
}

// AccountSyncForMapping handles synchronization for a specific mapping
type AccountSyncForMapping struct {
	parent                 *AccountSync
	copyLoop               *CopyLoop
	mapping                clients.CopyMapping
	lastCloseAllSignalTime *time.Time
	logger                 *slog.Logger
	cfg                    *subsettings.Service
}

// fetchRateLimitThrottled returns the account's Hyperliquid request-rate-limit
// usage, refreshing it from the exchange at most once per
// Service.HyperliquidRateLimitPollSeconds. Non-Hyperliquid accounts always
// return nil. The last successfully fetched value is reused between refreshes
// and kept on transient fetch errors so a single failed poll doesn't clear it.
func (asm *AccountSyncForMapping) fetchRateLimitThrottled() *clients.RateLimitInfo {
	if !asm.mapping.Account.IsHyperliquid() {
		return nil
	}

	p := asm.parent
	p.rateLimitMu.Lock()
	defer p.rateLimitMu.Unlock()

	ttlSeconds := asm.cfg.HyperliquidRateLimitPollSeconds
	if ttlSeconds <= 0 {
		ttlSeconds = 300.0
	}
	ttl := time.Duration(ttlSeconds * float64(time.Second))
	if p.cachedRateLimit != nil && !p.rateLimitFetched.IsZero() && time.Since(p.rateLimitFetched) < ttl {
		return p.cachedRateLimit
	}

	info, err := asm.copyLoop.Exchange.UserRateLimit(asm.mapping.Account)
	if err != nil {
		asm.logger.Warn("Failed to fetch Hyperliquid rate limit", "error", err)
		return p.cachedRateLimit
	}
	if info != nil {
		p.cachedRateLimit = info
		p.rateLimitFetched = time.Now()
	}
	return p.cachedRateLimit
}

// NewAccountSyncForMapping creates a new AccountSyncForMapping instance
func NewAccountSyncForMapping(
	parent *AccountSync,
	copyLoop *CopyLoop,
	mapping clients.CopyMapping,
	lastCloseAllSignalTime *time.Time,
	cfg *subsettings.Service,
) *AccountSyncForMapping {
	addressShort := mapping.Target.AddressShort()
	logger := logging.NewLogger("account-sync-mapping").With(
		"account", mapping.Name,
		"target", addressShort,
	)
	return &AccountSyncForMapping{
		parent:                 parent,
		copyLoop:               copyLoop,
		mapping:                mapping,
		lastCloseAllSignalTime: lastCloseAllSignalTime,
		logger:                 logger,
		cfg:                    cfg,
	}
}

// Sync synchronizes the account for this mapping
func (asm *AccountSyncForMapping) Sync() (*metric_types.AccountMetrics, error) {
	startTime := time.Now()
	target := ""
	if asm.mapping.Target.Address != nil {
		target = *asm.mapping.Target.Address
	}

	exchange := ""
	if asm.mapping.Account.ExchangeType != nil {
		exchange = *asm.mapping.Account.ExchangeType
	} else {
		return nil, fmt.Errorf("failed to get ExchangeType")
	}

	isDemo, err := asm.mapping.Account.IsDemo()
	if err != nil {
		return nil, fmt.Errorf("failed to get IsDemo flag: %w", err)
	}

	settleCoin, err := asm.mapping.Account.SettleCoin()
	if err != nil {
		return nil, fmt.Errorf("failed to get SettleCoin: %w", err)
	}

	metrics := &metric_types.AccountMetrics{
		StartTimestamp: startTime.Unix(),
		Account:        asm.mapping.Name,
		Target:         target,
		Exchange:       exchange,
		TargetExchange: "hyperliquid",
		IsDemo:         isDemo,
		SettleCoin:     settleCoin,
	}

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in account sync: %v", r)
			asm.logger.Error("Panic in account sync", "error", err)
			// TODO: should we simply let the error propagate there?
			panic(r)
		}
	}()

	// Get account state
	state, errs := asm.GetState()

	// Fill metrics from state
	asm.fillMetricsFromState(metrics, state)

	if len(errs) > 0 {
		metrics.EndTimestamp = time.Now().Unix()
		for _, e := range errs {
			errorCode := GetAccountSyncErrorCode(e)
			metrics.Errors = append(metrics.Errors, metric_types.AccountError{Exception: e.Error(), ErrorCode: errorCode})
		}
		return metrics, errs[0]
	}

	// Minimum equity check
	if err := asm.TryUpdateOrRaiseOnBadState(state); err != nil {
		metrics.EndTimestamp = time.Now().Unix()
		errorCode := GetAccountSyncErrorCode(err)
		metrics.Errors = append(metrics.Errors, metric_types.AccountError{Exception: err.Error(), ErrorCode: errorCode})
		return metrics, err
	}

	// Get symbol sync objects
	symbolSyncs, unmappedMetrics := asm.BuildSymbolSyncs(state)

	// Fill tracking ratio
	metrics.TrackingRatio = float64Ptr(asm.computeTrackingRatio(state, symbolSyncs))

	// Sync all symbols concurrently
	symbolMetrics, err := asm.syncSymbols(symbolSyncs)
	metrics.Symbols = utils.MergeSymbolMetrics(symbolMetrics, unmappedMetrics)
	metrics.EndTimestamp = time.Now().Unix()

	if err != nil {
		errorCode := GetAccountSyncErrorCode(err)
		metrics.Errors = append(metrics.Errors, metric_types.AccountError{Exception: err.Error(), ErrorCode: errorCode})
	}

	return metrics, err
}

// fillMetricsFromState fills account metrics from the account state
func (asm *AccountSyncForMapping) fillMetricsFromState(metrics *metric_types.AccountMetrics, state *AccountState) {
	if state.CurrentBalance != nil {
		metrics.Equity = float64Ptr(state.CurrentBalance.Equity)
		metrics.Balance = float64Ptr(state.CurrentBalance.WalletBalance)
		metrics.PnL = float64Ptr(state.CurrentBalance.UnrealisedPnl)
		metrics.InitialMargin = float64Ptr(state.CurrentBalance.TotalPositionIM)
		metrics.MaintenanceMargin = float64Ptr(state.CurrentBalance.TotalPositionMM)
		metrics.AvailableMarginRatio = float64Ptr(state.AvailableMarginRatio)
	}

	minimumEquity, err := asm.mapping.MinimumEquity()
	if err != nil {
		asm.logger.Error("Failed to get minimum equity", "error", err)
		minimumEquity = 0
	}
	metrics.MinimumEquity = minimumEquity

	if state.NormalizationComputed {
		metrics.Normalization = float64Ptr(state.AccountNormalization)
		metrics.TargetEquity = float64Ptr(state.TargetAccountValue)
	}

	if state.HyperliquidRateLimit != nil {
		used := state.HyperliquidRateLimit.Used
		capValue := state.HyperliquidRateLimit.Cap
		metrics.HyperliquidRateLimitUsed = &used
		metrics.HyperliquidRateLimitCap = &capValue
	}

	if state.ApiKeyInfo != nil {
		if state.ApiKeyInfo.UserID != nil {
			userID := strconv.FormatInt(*state.ApiKeyInfo.UserID, 10)
			metrics.ExchangeUserID = &userID
		}

		if state.ApiKeyInfo.ExpiredAt != nil {
			expiredAt := *state.ApiKeyInfo.ExpiredAt
			metrics.APIKeyExpiredAt = &expiredAt
		}
	}

	metrics.Symbols = asm.buildDisplaySymbolMetrics(state)
}

func float64Ptr(v float64) *float64 {
	return &v
}

// buildDisplaySymbolMetrics builds display-only SymbolMetrics from the
// follower's own open positions.
func (asm *AccountSyncForMapping) buildDisplaySymbolMetrics(state *AccountState) []metric_types.SymbolMetrics {
	settleCoin, err := asm.mapping.Account.SettleCoin()
	if err != nil {
		settleCoin = ""
	}

	// Index stop-loss trigger prices by symbol (mirrors SymbolSync.SymbolTpslOrders).
	stopLossBySymbol := make(map[string]float64)
	for _, order := range state.OpenStopOrders {
		if order.OrderType == "SL" {
			stopLossBySymbol[order.Symbol] = order.TriggerPrice
		}
	}

	now := time.Now().Unix()
	symbols := make([]metric_types.SymbolMetrics, 0, len(state.CurrentPositions))
	for i := range state.CurrentPositions {
		pos := state.CurrentPositions[i]
		symbols = append(symbols, metric_types.SymbolMetrics{
			StartTimestamp: now,
			EndTimestamp:   now,
			Symbol:         pos.Symbol,
			SettleCoin:     settleCoin,
			Size:           pos.Size * float64(utils.SideToMultiplier(pos.Side)),
			Value:          pos.PositionValue,
			Price:          pos.MarkPrice,
			EntryPrice:     pos.EntryPrice,
			PnL:            pos.UnrealizedPnl,
			Leverage:       pos.Leverage,
			Liquidation:    pos.LiqPrice,
			StopLoss:       stopLossBySymbol[pos.Symbol],
		})
	}
	return symbols
}

// computeTrackingRatio computes the tracking ratio for the account
func (asm *AccountSyncForMapping) computeTrackingRatio(state *AccountState, symbolSyncs []*SymbolSync) float64 {
	totalTargetValue := 0.0
	totalCurrentValue := 0.0

	for _, ss := range symbolSyncs {
		if ss.targetPosition != nil {
			totalTargetValue += math.Abs(ss.targetPosition.PositionValue.Float64() * state.AccountNormalization)
		}
		if ss.currentPosition != nil {
			totalCurrentValue += math.Abs(ss.currentPosition.PositionValue)
		}
	}

	denominator := math.Max(totalCurrentValue, totalTargetValue)
	if denominator > 0 {
		return (totalCurrentValue - totalTargetValue) / denominator
	}
	return 0
}

// dexPositionsResult holds the copied trader's positions fetched from one perp dex.
type dexPositionsResult struct {
	Positions *clients.PositionsResponse
	Err       error
}

// GetState retrieves the current account state
func (asm *AccountSyncForMapping) GetState() (*AccountState, []error) {
	var currentPositions []clients.Position
	var currentPositionsErr error
	var currentBalance *clients.AccountBalanceCoin
	var currentBalanceErr error
	targetPositionsByDex := make(map[string]*dexPositionsResult, len(clients.HyperliquidSupportedDexes))
	for _, dex := range clients.HyperliquidSupportedDexes {
		targetPositionsByDex[dex] = &dexPositionsResult{}
	}
	var targetAccountValueInputs *TargetAccountValueInputs
	var targetAccountValueInputsErr error
	var accountInfo *clients.AccountInfo
	var accountInfoErr error
	var openStopOrders []clients.OpenStopOrder
	var openStopOrdersErr error
	var apiKeyInfo *clients.ApiKeyInfo
	var apiKeyInfoErr error
	var hyperliquidRateLimit *clients.RateLimitInfo

	var wg sync.WaitGroup
	var errors []error

	// Each goroutine writes to its own result/error variables, so no mutex is
	// needed; wg.Wait() below establishes the happens-before barrier for reads.

	// Get current positions
	wg.Add(1)
	go func() {
		defer wg.Done()
		currentPositions, currentPositionsErr = asm.copyLoop.Exchange.Positions(asm.mapping.Account)
	}()

	// Get current open stop orders
	wg.Add(1)
	go func() {
		defer wg.Done()
		openStopOrders, openStopOrdersErr = asm.copyLoop.Exchange.OpenStopOrders(asm.mapping.Account)
	}()

	// Get current balance
	wg.Add(1)
	go func() {
		defer wg.Done()
		currentBalance, currentBalanceErr = asm.copyLoop.Exchange.SettleCoinBalance(asm.mapping.Account)
	}()

	// Get target positions on every supported dex clearing house
	for dex, result := range targetPositionsByDex {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.Positions, result.Err = asm.copyLoop.Hyperliquid.Positions(asm.mapping.Target, dex)
		}()
	}

	// Get Hyperliquid inputs for target account value (computed by the backend)
	wg.Add(1)
	go func() {
		defer wg.Done()
		targetAccountValueInputs, targetAccountValueInputsErr = asm.fetchTargetAccountValueInputs()
	}()

	// Get account info
	wg.Add(1)
	go func() {
		defer wg.Done()
		accountInfo, accountInfoErr = asm.copyLoop.Exchange.AccountInfo(asm.mapping.Account)
	}()

	// Get Hyperliquid request-rate-limit usage (throttled; nil for other exchanges)
	wg.Add(1)
	go func() {
		defer wg.Done()
		hyperliquidRateLimit = asm.fetchRateLimitThrottled()
	}()

	// Get API key and user information
	isDemo, _ := asm.mapping.Account.IsDemo()
	wg.Add(1)
	go func() {
		defer wg.Done()
		apiKeyInfo, apiKeyInfoErr = asm.copyLoop.Exchange.GetApiKeyAndUserInformation(asm.mapping.Account, !isDemo)
	}()

	wg.Wait()

	// Extract target positions from the responses of every supported dex
	var hyperPositions []clients.HyperliquidPosition
	for _, dex := range clients.HyperliquidSupportedDexes {
		if positions := targetPositionsByDex[dex].Positions; positions != nil {
			for _, ap := range positions.AssetPositions {
				hyperPositions = append(hyperPositions, ap.Position)
			}
		}
	}

	// create account state before raising any of the errors
	state, stateErr := NewAccountState(asm, accountInfo, currentBalance, targetPositionsByDex[""].Positions, targetAccountValueInputs, currentPositions, hyperPositions, openStopOrders, apiKeyInfo, asm.cfg)
	state.HyperliquidRateLimit = hyperliquidRateLimit

	// If we cannot connect to the exchange due to API key issues, return early
	exchangeClientErrs := []error{
		currentPositionsErr,
		openStopOrdersErr,
		currentBalanceErr,
		accountInfoErr,
		apiKeyInfoErr,
	}
	if cause := firstApiKeyNotSetError(exchangeClientErrs); cause != nil {
		asm.logger.Error("Exchange API key or secret is not set", "error", cause)
		return state, []error{&internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message: fmt.Sprintf(
				"API key or secret is not set for %s: %v",
				*asm.mapping.Account.ExchangeType,
				cause,
			),
			ErrorCode: internal.AccountApiKeyNotSetErrorCode,
			FromError: &cause,
		}}
	}

	if cause := firstInvalidApiKeyOrPermissionsError(exchangeClientErrs); cause != nil {
		asm.logger.Error("Exchange rejected API key, IP, or permissions", "error", cause)
		return state, []error{&internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message: fmt.Sprintf(
				"Invalid API key, IP, or permissions for %s: %v",
				*asm.mapping.Account.ExchangeType,
				cause,
			),
			ErrorCode: internal.AccountInvalidApiKeyOrPermissionsErrorCode,
			FromError: &cause,
		}}
	}

	// msgOr returns the underlying fetch error's message or a fallback
	msgOr := func(err error, fallback string) string {
		if err != nil {
			return err.Error()
		}
		return fallback
	}

	// Check we could retrieve the current positions
	if currentPositionsErr != nil || currentPositions == nil {
		asm.logger.Error("Failed to retrieve positions", "error", currentPositionsErr)
		errors = append(errors, &internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message:     msgOr(currentPositionsErr, fmt.Sprintf("Could not retrieve current positions from %s", *asm.mapping.Account.ExchangeType)),
			ErrorCode:   internal.AccountPositionsRetrievalErrorCode,
		})
	}

	// Check we could retrieve the current open stop orders
	if openStopOrdersErr != nil || openStopOrders == nil {
		asm.logger.Error("Failed to retrieve open stop orders", "error", openStopOrdersErr)
		errors = append(errors, &internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message:     msgOr(openStopOrdersErr, fmt.Sprintf("Could not retrieve current open stop orders from %s", *asm.mapping.Account.ExchangeType)),
			ErrorCode:   internal.AccountOpenStopOrdersRetrievalErrorCode,
		})
	}

	// Check we could retrieve the current balance
	if currentBalanceErr != nil || currentBalance == nil {
		asm.logger.Warn("Could not retrieve balance", "exchange", *asm.mapping.Account.ExchangeType)
		message := msgOr(currentBalanceErr, fmt.Sprintf("Could not retrieve balance from %s", *asm.mapping.Account.ExchangeType))
		errorCode := internal.AccountBalanceRetrievalErrorCode
		if clients.IsHyperliquidInvalidAccountTypeError(currentBalanceErr) {
			message = "Hyperliquid account must be a unified (perps) account to retrieve its balance"
			errorCode = internal.AccountBalanceInvalidAccountTypeErrorCode
		}
		errors = append(errors, &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     message,
			Positions:   currentPositions,
			Immediate:   false,
			ErrorCode:   errorCode,
		})
	}

	// Check we could retrieve the target positions on every supported dex
	for _, dex := range clients.HyperliquidSupportedDexes {
		result := targetPositionsByDex[dex]
		if result.Err == nil && result.Positions != nil {
			continue
		}
		source := "hyperliquid"
		if dex != "" {
			source = fmt.Sprintf("%s dex", dex)
		}
		asm.logger.Warn("Could not retrieve target positions", "dex", dex, "error", result.Err)
		errors = append(errors, &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     msgOr(result.Err, fmt.Sprintf("Could not retrieve target positions from %s", source)),
			Positions:   currentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountTargetPositionsRetrievalErrorCode,
		})
	}

	// Check we could retrieve Hyperliquid data for target account value
	if asm.mapping.Target.Address != nil && *asm.mapping.Target.Address != "" && targetAccountValueInputsErr != nil {
		asm.logger.Warn("Could not retrieve target account value", "error", targetAccountValueInputsErr)
		errors = append(errors, &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     targetAccountValueInputsErr.Error(),
			Positions:   currentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountTargetAccountValueRetrievalErrorCode,
		})
	}

	// Check we could retrieve account info
	if accountInfoErr != nil || accountInfo == nil {
		asm.logger.Warn("Could not retrieve account info", "exchange", *asm.mapping.Account.ExchangeType)
		errors = append(errors, &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     msgOr(accountInfoErr, fmt.Sprintf("Could not retrieve account info from %s", *asm.mapping.Account.ExchangeType)),
			Positions:   currentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountInfoRetrievalErrorCode,
		})
	}

	// Check we could retrieve API key and user information
	if apiKeyInfoErr != nil || apiKeyInfo == nil {
		asm.logger.Warn("Could not retrieve API key and user information", "error", apiKeyInfoErr)
		errors = append(errors, &internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message:     msgOr(apiKeyInfoErr, fmt.Sprintf("Could not retrieve API key and user information from %s", *asm.mapping.Account.ExchangeType)),
			ErrorCode:   internal.AccountApiKeyInfoRetrievalErrorCode,
		})
	}

	// raise stateError if any
	if stateErr != nil {
		errors = append(errors, stateErr)
	}

	if len(errors) > 0 {
		return state, errors
	}
	return state, nil
}

// TryUpdateOrRaiseOnBadState checks if the account state is valid. Tries to update it
// and raises on failure
func (asm *AccountSyncForMapping) TryUpdateOrRaiseOnBadState(state *AccountState) error {
	// Check minimum equity
	minimumEquity, err := asm.mapping.MinimumEquity()
	if err != nil {
		asm.logger.Error("Failed to get minimum equity", "error", err)
		return err
	}

	if state.CurrentBalance.Equity < minimumEquity {
		asm.logger.Error("Current balance below minimum equity")
		return &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message: fmt.Sprintf("Current balance %.3fk is below minimum equity %.2fk",
				state.CurrentBalance.Equity*1e-3, minimumEquity*1e-3),
			Positions: state.CurrentPositions,
			Immediate: true,
			ErrorCode: internal.AccountBalanceBelowMinimumEquityErrorCode,
		}
	}

	// check account margin mode. Try to update it
	if state.AccountInfo.MarginMode == nil || *state.AccountInfo.MarginMode != "ISOLATED_MARGIN" {
		marginMode := "ISOLATED_MARGIN"
		err := asm.copyLoop.Exchange.SetAccountMarginMode(asm.mapping.Account, &marginMode)
		if err != nil {
			asm.logger.Error("Account is not in isolated margin mode")
			return &AccountPositionsShouldBeClosedError{
				AccountName: asm.mapping.Name,
				Message:     fmt.Sprintf("Failed to set account margin mode: %s", err.Error()),
				Positions:   state.CurrentPositions,
				Immediate:   false,
				ErrorCode:   internal.AccountMarginModeErrorCode,
			}
		}
	}

	// check account position mode. Try to update it
	err = asm.copyLoop.Exchange.SetAccountPositionMode(asm.mapping.Account, "one-way")
	if err != nil {
		asm.logger.Error("Account is not in one-way position mode")
		return &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     fmt.Sprintf("Failed to set account in one-way position mode: %s", err.Error()),
			Positions:   state.CurrentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountPositionModeErrorCode,
		}
	}

	// check mapping status
	if asm.mapping.Status == clients.MappingStatusEmergencyClose {
		asm.logger.Error("Closing of positions required by server, applying immediately")
		return &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     "Closing of positions required by server",
			Positions:   state.CurrentPositions,
			Immediate:   true,
			ErrorCode:   internal.AccountEmergencyCloseErrorCode,
		}
	}
	if asm.mapping.Status == clients.MappingStatusVersionBlocked {
		asm.logger.Error("Client version blocked by server")
		return &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     "Client version blocked by server - update required",
			Positions:   state.CurrentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountVersionBlockedErrorCode,
		}
	}
	if asm.mapping.Status != clients.MappingStatusOK {
		asm.logger.Error("Error happened during mapping retrieval")
		return &AccountPositionsShouldBeClosedError{
			AccountName: asm.mapping.Name,
			Message:     "Error happened during mapping retrieval",
			Positions:   state.CurrentPositions,
			Immediate:   false,
			ErrorCode:   internal.AccountStatusErrorCode,
		}
	}

	return nil
}

// BuildSymbolSyncs builds symbol sync objects for all symbols. It also
// returns pre-built metric entries for target symbols that could not be
// mapped to an exchange symbol, or that are traded on several dexes at once —
// these surface as SymbolNotExistErrorCode / SymbolDexCollisionErrorCode in the
// loop metrics rather than silently disappearing.
func (asm *AccountSyncForMapping) BuildSymbolSyncs(state *AccountState) ([]*SymbolSync, []metric_types.SymbolMetrics) {
	settleCoin, err := asm.mapping.Account.SettleCoin()
	if err != nil {
		asm.logger.Error("Failed to get SettleCoin", "error", err)
		settleCoin = ""
	}

	// Get position lists on both sides
	currentPositionsBySymbol := make(map[string]*clients.Position)
	for i := range state.CurrentPositions {
		pos := state.CurrentPositions[i]
		currentPositionsBySymbol[pos.Symbol] = &pos
	}

	// Collisions are detected on the Hyperliquid coins, after the exclusions and
	// before resolution, which would merge xyz:DRAM and io:DRAM into one symbol.
	targetPositions := asm.nonExcludedTargetPositions(state.TargetPositions)
	targetPositions, unmappedMetrics := asm.withoutDexCollisions(targetPositions, settleCoin)

	resolvedBySymbol := make(map[string][]*clients.HyperliquidPosition)
	for _, pos := range targetPositions {
		exchangeSymbol, err := asm.copyLoop.Exchange.HyperToExchangeSymbol(asm.mapping.Account, pos.Coin)
		if err != nil {
			code := internal.UnknownErrorCode
			if errors.Is(err, clients.ErrSymbolNotExist) {
				code = internal.SymbolNotExistErrorCode
			}
			asm.logger.Error("Failed to convert hyper symbol to exchange symbol",
				"symbol", pos.Coin, "error", err)
			now := time.Now().Unix()
			unmappedMetrics = append(unmappedMetrics, metric_types.SymbolMetrics{
				StartTimestamp: now,
				EndTimestamp:   now,
				Symbol:         pos.Coin,
				TargetSymbol:   pos.Coin,
				SettleCoin:     settleCoin,
				Errors: []metric_types.SymbolError{{
					Exception: err.Error(),
					ErrorCode: code,
				}},
			})
			continue
		}
		resolvedBySymbol[exchangeSymbol] = append(resolvedBySymbol[exchangeSymbol], pos)
	}
	tradeableTargetPositionsBySymbol, resolutionCollisionMetrics := asm.withoutResolutionCollisions(resolvedBySymbol, settleCoin)
	unmappedMetrics = append(unmappedMetrics, resolutionCollisionMetrics...)

	// Get all symbols on both sides
	allSymbolsMap := make(map[string]bool)
	for symbol := range currentPositionsBySymbol {
		allSymbolsMap[symbol] = true
	}
	for symbol := range tradeableTargetPositionsBySymbol {
		allSymbolsMap[symbol] = true
	}

	allSymbols := make([]string, 0, len(allSymbolsMap))
	for symbol := range allSymbolsMap {
		allSymbols = append(allSymbols, symbol)
	}
	sort.Strings(allSymbols)

	// Compute tracking ratio for observability
	totalTargetValue := 0.0
	totalCurrentValue := 0.0
	for _, symbol := range allSymbols {
		if pos, ok := tradeableTargetPositionsBySymbol[symbol]; ok {
			totalTargetValue += math.Abs(pos.PositionValue.Float64() * state.AccountNormalization)
		}
		if pos, ok := currentPositionsBySymbol[symbol]; ok {
			totalCurrentValue += math.Abs(pos.PositionValue)
		}
	}
	trackingRatioDenominator := math.Max(totalCurrentValue, totalTargetValue)
	trackingRatio := 0.0
	if trackingRatioDenominator > 0 {
		trackingRatio = (totalCurrentValue - totalTargetValue) / trackingRatioDenominator
	}
	_ = trackingRatio // Stub metrics

	// Create the result list of SymbolSync objects
	result := make([]*SymbolSync, 0, len(allSymbols))
	for _, symbol := range allSymbols {
		var currentPos *clients.Position
		if pos, ok := currentPositionsBySymbol[symbol]; ok {
			currentPos = pos
		}
		var targetPos *clients.HyperliquidPosition
		if pos, ok := tradeableTargetPositionsBySymbol[symbol]; ok {
			targetPos = pos
		}

		result = append(result, NewSymbolSync(
			asm.copyLoop.Exchange,
			asm,
			state,
			symbol,
			currentPos,
			targetPos,
			asm.cfg,
		))
	}

	return result, unmappedMetrics
}

// nonExcludedTargetPositions returns the target positions whose coin is excluded
// neither globally in the settings nor specifically for this copy target.
func (asm *AccountSyncForMapping) nonExcludedTargetPositions(positions []clients.HyperliquidPosition) []*clients.HyperliquidPosition {
	excluded := make(map[string]bool, len(asm.cfg.HyperExcludedSymbols)+len(asm.mapping.Target.ExcludedSymbols))
	for _, symbol := range asm.cfg.HyperExcludedSymbols {
		excluded[symbol] = true
	}
	for _, symbol := range asm.mapping.Target.ExcludedSymbols {
		excluded[symbol] = true
	}

	kept := make([]*clients.HyperliquidPosition, 0, len(positions))
	for i := range positions {
		if !excluded[positions[i].Coin] {
			kept = append(kept, &positions[i])
		}
	}
	return kept
}

// withoutDexCollisions drops the coins traded on several perp dexes at once and
// returns a SymbolDexCollisionErrorCode metric for each of them. Having no target,
// the user's position on such a symbol gets closed.
func (asm *AccountSyncForMapping) withoutDexCollisions(
	positions []*clients.HyperliquidPosition,
	settleCoin string,
) ([]*clients.HyperliquidPosition, []metric_types.SymbolMetrics) {
	coins := make([]string, len(positions))
	for i, pos := range positions {
		coins[i] = pos.Coin
	}
	collisions := utils.DexCollisions(coins)
	if len(collisions) == 0 {
		return positions, nil
	}

	kept := make([]*clients.HyperliquidPosition, 0, len(positions))
	var collisionMetrics []metric_types.SymbolMetrics
	for _, pos := range positions {
		colliding, ok := collisions[utils.BareCoin(pos.Coin)]
		if !ok {
			kept = append(kept, pos)
			continue
		}
		asm.logger.Warn("Symbol traded on several dexes, not copied",
			"symbol", pos.Coin, "colliding", colliding)
		collisionMetrics = append(collisionMetrics, dexCollisionMetric(pos.Coin, colliding, settleCoin))
	}
	return kept, collisionMetrics
}

// withoutResolutionCollisions keeps the exchange symbols resolved from a single coin.
// Differently named coins resolving to the same exchange symbol (e.g. through an
// override) would otherwise overwrite each other: they are all dropped with a
// SymbolDexCollisionErrorCode metric.
//
// On a Hyperliquid account resolution is the identity, so this failsafe never
// triggers there: two differently named coins for the same asset (e.g. xyz:GOLD and
// io:XAU) are both copied on Hyperliquid while they are excluded on Binance and Bybit.
func (asm *AccountSyncForMapping) withoutResolutionCollisions(
	resolvedBySymbol map[string][]*clients.HyperliquidPosition,
	settleCoin string,
) (map[string]*clients.HyperliquidPosition, []metric_types.SymbolMetrics) {
	tradeable := make(map[string]*clients.HyperliquidPosition, len(resolvedBySymbol))
	var collisionMetrics []metric_types.SymbolMetrics
	for symbol, positions := range resolvedBySymbol {
		if len(positions) == 1 {
			tradeable[symbol] = positions[0]
			continue
		}
		colliding := make([]string, len(positions))
		for i, pos := range positions {
			colliding[i] = pos.Coin
		}
		sort.Strings(colliding)
		asm.logger.Warn("Symbols resolve to the same exchange symbol, not copied",
			"symbol", symbol, "colliding", colliding)
		for _, coin := range colliding {
			collisionMetrics = append(collisionMetrics, dexCollisionMetric(coin, colliding, settleCoin))
		}
	}
	return tradeable, collisionMetrics
}

func dexCollisionMetric(coin string, colliding []string, settleCoin string) metric_types.SymbolMetrics {
	now := time.Now().Unix()
	return metric_types.SymbolMetrics{
		StartTimestamp: now,
		EndTimestamp:   now,
		Symbol:         coin,
		TargetSymbol:   coin,
		SettleCoin:     settleCoin,
		Errors: []metric_types.SymbolError{{
			Exception: fmt.Sprintf("symbols colliding across dexes, not copied: %s", strings.Join(colliding, ", ")),
			ErrorCode: internal.SymbolDexCollisionErrorCode,
		}},
	}
}

// SyncSymbols synchronizes all symbols (public, without metrics)
func (asm *AccountSyncForMapping) SyncSymbols(symbols []*SymbolSync) error {
	_, err := asm.syncSymbols(symbols)
	return err
}

// syncSymbols synchronizes all symbols and returns their metrics
func (asm *AccountSyncForMapping) syncSymbols(symbols []*SymbolSync) ([]metric_types.SymbolMetrics, error) {
	symbolNames := make([]string, len(symbols))
	for i, s := range symbols {
		symbolNames[i] = s.symbol
	}
	asm.logger.Info("Syncing symbols", "symbols", symbolNames)

	var wg sync.WaitGroup
	var errors []error
	var symbolMetrics []metric_types.SymbolMetrics
	var mu sync.Mutex

	for _, symbolSync := range symbols {
		wg.Add(1)
		go func(ss *SymbolSync) {
			defer wg.Done()
			metrics, err := ss.Sync()
			mu.Lock()
			if metrics != nil {
				symbolMetrics = append(symbolMetrics, *metrics)
			}
			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(symbolSync)
	}

	wg.Wait()

	if len(errors) > 0 {
		asm.logger.Debug("Sync complete with errors")
		return symbolMetrics, &internal.AccountSyncError{
			AccountName: asm.mapping.Name,
			Message:     fmt.Sprintf("errors syncing symbols: %v", errors),
			ErrorCode:   internal.AccountSyncSymbolsErrorCode,
		}
	}

	asm.logger.Debug("Sync complete")
	return symbolMetrics, nil
}

// TargetAccountValueInputs holds raw Hyperliquid responses used to compute target account value.
type TargetAccountValueInputs struct {
	UserAbstraction      clients.UserAbstractionMode
	Portfolio            string
	SpotClearinghouse    string
	SpotMetaAndAssetCtxs string
}

// fetchTargetAccountValueInputs fetches Hyperliquid data forwarded to the backend for account value extraction.
func (asm *AccountSyncForMapping) fetchTargetAccountValueInputs() (*TargetAccountValueInputs, error) {
	target := asm.mapping.Target
	if target.Address == nil || *target.Address == "" {
		return nil, nil
	}

	var (
		mode         clients.UserAbstractionMode
		portfolio    string
		modeErr      error
		portfolioErr error
		wg           sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		mode, modeErr = asm.copyLoop.Hyperliquid.UserAbstraction(target)
	}()
	go func() {
		defer wg.Done()
		portfolio, portfolioErr = asm.copyLoop.Hyperliquid.Portfolio(target)
	}()
	wg.Wait()

	if modeErr != nil {
		return nil, modeErr
	}
	if portfolioErr != nil {
		return nil, portfolioErr
	}

	inputs := &TargetAccountValueInputs{
		UserAbstraction: mode,
		Portfolio:       portfolio,
	}

	if mode == clients.UserAbstractionDexAbstraction {
		var (
			spot     string
			spotMeta string
			spotErr  error
			metaErr  error
			spotWG   sync.WaitGroup
		)

		spotWG.Add(2)
		go func() {
			defer spotWG.Done()
			spot, spotErr = asm.copyLoop.Hyperliquid.SpotClearinghouseState(target)
		}()
		go func() {
			defer spotWG.Done()
			spotMeta, metaErr = asm.copyLoop.Hyperliquid.SpotMetaAndAssetCtxs()
		}()
		spotWG.Wait()

		if spotErr != nil {
			return nil, spotErr
		}
		if metaErr != nil {
			return nil, metaErr
		}

		inputs.SpotClearinghouse = spot
		inputs.SpotMetaAndAssetCtxs = spotMeta
	}

	return inputs, nil
}

func targetAccountValueErrorCode(err error) internal.SyncErrorCode {
	if clients.IsHyperliquidDexAbstractionModeError(err) {
		return internal.AccountTargetUserAbstractionDexAbstractionErrorCode
	}
	if clients.IsHyperliquidUnsupportedUserAbstractionModeError(err) {
		return internal.AccountTargetUserAbstractionUnsupportedModeErrorCode
	}
	return internal.AccountTargetAccountValueRetrievalErrorCode
}

func firstInvalidApiKeyOrPermissionsError(errs []error) error {
	for _, err := range errs {
		if errors.Is(err, clients.ErrInvalidApiKeyOrPermissions) {
			return err
		}
	}
	return nil
}

func firstApiKeyNotSetError(errs []error) error {
	for _, err := range errs {
		if errors.Is(err, clients.ErrApiKeyNotSet) {
			return err
		}
	}
	return nil
}

// AccountState represents the state of an account
type AccountState struct {
	AccountSync              *AccountSyncForMapping
	AccountInfo              *clients.AccountInfo
	CurrentBalance           *clients.AccountBalanceCoin
	TargetBalance            *clients.PositionsResponse
	TargetAccountValueInputs *TargetAccountValueInputs
	TargetAccountValue       float64
	CurrentPositions         []clients.Position
	TargetPositions          []clients.HyperliquidPosition
	OpenStopOrders           []clients.OpenStopOrder
	ApiKeyInfo               *clients.ApiKeyInfo
	AccountNormalization     float64
	NormalizationComputed    bool
	AvailableMarginRatio     float64
	HyperliquidRateLimit     *clients.RateLimitInfo
	cfg                      *subsettings.Service
}

// NewAccountState creates a new AccountState
func NewAccountState(
	accountSync *AccountSyncForMapping,
	accountInfo *clients.AccountInfo,
	currentBalance *clients.AccountBalanceCoin,
	targetBalance *clients.PositionsResponse,
	targetAccountValueInputs *TargetAccountValueInputs,
	currentPositions []clients.Position,
	targetPositions []clients.HyperliquidPosition,
	openStopOrders []clients.OpenStopOrder,
	apiKeyInfo *clients.ApiKeyInfo,
	cfg *subsettings.Service,
) (*AccountState, *internal.AccountSyncError) {
	state := &AccountState{
		AccountSync:              accountSync,
		AccountInfo:              accountInfo,
		CurrentBalance:           currentBalance,
		TargetBalance:            targetBalance,
		TargetAccountValueInputs: targetAccountValueInputs,
		CurrentPositions:         currentPositions,
		TargetPositions:          targetPositions,
		OpenStopOrders:           openStopOrders,
		ApiKeyInfo:               apiKeyInfo,
		cfg:                      cfg,
	}

	state.AvailableMarginRatio = state.computeAvailableMarginRatio()

	normalization, err := state.computeAccountNormalization()
	if err != nil {
		state.AccountNormalization = 0
		errorCode := internal.AccountNormalizationRetrievalErrorCode
		if isTargetAccountValueError(err) {
			errorCode = targetAccountValueErrorCode(err)
		}
		return state, &internal.AccountSyncError{
			AccountName: accountSync.mapping.Name,
			Message:     err.Error(),
			ErrorCode:   errorCode,
			FromError:   &err,
		}
	}
	state.AccountNormalization = normalization
	state.NormalizationComputed = true

	return state, nil
}

// computeAvailableMarginRatio computes the available margin ratio
func (as *AccountState) computeAvailableMarginRatio() float64 {
	if as.CurrentBalance == nil {
		return 0.0
	}
	walletBalance := as.CurrentBalance.WalletBalance
	if walletBalance > 0 {
		return as.CurrentBalance.AvailableBalance / walletBalance
	}
	return 0.0
}

func isTargetAccountValueError(err error) bool {
	return clients.IsHyperliquidDexAbstractionModeError(err) ||
		clients.IsHyperliquidUnsupportedUserAbstractionModeError(err) ||
		clients.IsHyperliquidPortfolioWindowError(err)
}

func normalizationApiErrorToError(apiErr *clients.NormalizationApiError) error {
	switch apiErr.Detail.Code {
	case "dex_abstraction_missing_spot_data":
		return &clients.HyperliquidDexAbstractionModeError{}
	case "unsupported_user_abstraction_mode":
		return &clients.HyperliquidUnsupportedUserAbstractionModeError{Mode: apiErr.Detail.Mode}
	case "portfolio_window_missing":
		return &clients.HyperliquidPortfolioWindowError{Window: apiErr.Detail.Window}
	default:
		return apiErr
	}
}

// computeAccountNormalization computes the account normalization via the control API
func (as *AccountState) computeAccountNormalization() (float64, error) {
	if as.CurrentBalance == nil {
		return 0, fmt.Errorf("no balance available")
	}
	target := as.AccountSync.mapping.Target
	if !target.Enabled || target.Address == nil || *target.Address == "" {
		return 0, nil
	}
	if as.TargetAccountValueInputs == nil {
		return 0, fmt.Errorf("no target account value inputs available")
	}

	inputs := as.TargetAccountValueInputs
	currentValue := as.CurrentBalance.Equity

	resp, err := as.AccountSync.copyLoop.ControlApi.ComputeNormalization(clients.ComputeNormalizationRequest{
		User: clients.ComputeNormalizationUser{
			Account:      as.AccountSync.mapping.Name,
			CurrentValue: currentValue,
		},
		Target: clients.ComputeNormalizationTarget{
			UserAbstraction:        string(inputs.UserAbstraction),
			Portfolio:              inputs.Portfolio,
			SpotClearinghouseState: inputs.SpotClearinghouse,
			SpotMetaAndAssetCtxs:   inputs.SpotMetaAndAssetCtxs,
		},
	})
	if err != nil {
		var apiErr *clients.NormalizationApiError
		if errors.As(err, &apiErr) {
			return 0, normalizationApiErrorToError(apiErr)
		}
		return 0, fmt.Errorf("computing account normalization: %w", err)
	}

	chosenNormalization := resp.Normalization
	as.TargetAccountValue = resp.TargetValue
	targetValue := resp.TargetValue

	// Render the normalization as a string for logging
	var chosenNormalizationStr string
	if chosenNormalization == 0 {
		chosenNormalizationStr = "0x"
	} else if chosenNormalization < 0.25 {
		chosenNormalizationStr = fmt.Sprintf("1/%.3fx", 1/chosenNormalization)
	} else {
		chosenNormalizationStr = fmt.Sprintf("%.3fx", chosenNormalization)
	}

	as.AccountSync.logger.Debug("Using account normalization",
		"normalization", chosenNormalizationStr,
		"target_k", targetValue*1e-3,
		"current_k", currentValue*1e-3)

	return chosenNormalization, nil
}
