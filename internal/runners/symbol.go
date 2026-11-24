package runners

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
)

// SymbolSync handles synchronization of a single symbol
type SymbolSync struct {
	exchange        *clients.Exchange
	accountSync     *AccountSyncForMapping
	accountState    *AccountState
	mapping         clients.CopyMapping
	account         subsettings.ExchangeAccount
	target          clients.CopyTarget
	symbol          string
	currentPosition *clients.Position
	targetPosition  *clients.HyperliquidPosition // Hyperliquid position
	tpslMaxSleepSec float64
	logger          *slog.Logger
	cfg             *subsettings.Service
}

// NewSymbolSync creates a new SymbolSync instance
func NewSymbolSync(
	exchange *clients.Exchange,
	accountSync *AccountSyncForMapping,
	accountState *AccountState,
	symbol string,
	currentPosition *clients.Position,
	targetPosition *clients.HyperliquidPosition,
	cfg *subsettings.Service,
) *SymbolSync {
	logger := logging.NewLogger("symbol-sync").With(
		"account", accountSync.mapping.Name,
		"target", accountSync.mapping.Target.AddressShort(),
		"symbol", symbol,
	)

	return &SymbolSync{
		exchange:        exchange,
		accountSync:     accountSync,
		accountState:    accountState,
		mapping:         accountSync.mapping,
		account:         accountSync.mapping.Account,
		target:          accountSync.mapping.Target,
		symbol:          symbol,
		currentPosition: currentPosition,
		targetPosition:  targetPosition,
		tpslMaxSleepSec: cfg.SymbolSyncTpslMaxSleepSeconds,
		logger:          logger,
		cfg:             cfg,
	}
}

func GetSymbolSyncErrorCode(err error) internal.SyncErrorCode {
	if symbolSyncError, ok := err.(*internal.SymbolSyncError); ok {
		return symbolSyncError.ErrorCode
	}
	return internal.UnknownErrorCode
}

// SetSymbolMarginType sets the symbol margin type
func (s *SymbolSync) checkAndSetSymbolMarginTypeToIsolated() error {
	marginType := "ISOLATED"

	if s.currentPosition == nil {
		err := s.exchange.SetSymbolMarginType(s.account, s.symbol, marginType)
		if err != nil {
			return s.newSyncError(err, fmt.Sprintf("Error setting symbol margin type to isolated: %v", err), internal.DefaultErrorCode)
		}
	} else {
		// We cannot change margin type on open position
		symbolMarginType, err := s.exchange.SymbolMarginType(s.account, s.symbol)
		if err != nil {
			return s.newSyncError(err, fmt.Sprintf("Error getting symbol margin type: %v", err), internal.DefaultErrorCode)
		}
		// Exchange not allowing per symbol margin type return an empty string
		if symbolMarginType != "" && symbolMarginType != marginType {
			return &internal.SymbolSyncError{
				AccountName: s.mapping.Name,
				Symbol:      s.symbol,
				Message:     fmt.Sprintf("Symbol margin type is not correct, expected %s, got %s", marginType, symbolMarginType),
				FromError:   nil,
				ErrorCode:   internal.DefaultErrorCode,
			}
		}
	}

	return nil
}

// adjustLeverage sets the leverage for the symbol.
// When there is no open position, account sync already established that (currentPosition
// is nil), so we do not re-query Position — just set leverage before opening.
// Reject leverage above the exchange's per-symbol maximum before attempting the set
func (s *SymbolSync) adjustLeverage() error {
	if s.targetPosition == nil {
		return nil
	}
	symbolLeverage, err := s.mapping.ComputeLeverage(s.symbol, s.targetPosition)
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Invalid exchange type, cannot get leverage: %v", err), internal.DefaultErrorCode)
	}

	fromLeverage := 0.0
	if s.currentPosition != nil {
		fromLeverage = s.currentPosition.Leverage
		if math.Abs(fromLeverage-float64(symbolLeverage)) <= 1e-3 {
			return nil
		}
	}

	symbolInfo, err := s.exchange.SymbolInfo(s.account, s.symbol, true)
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Error getting symbol info: %v", err), internal.SymbolSymbolInfoErrorCode)
	}
	if symbolInfo.MaxLeverage > 0 && symbolLeverage > symbolInfo.MaxLeverage {
		leverageErr := fmt.Errorf("requested leverage %d exceeds the exchange maximum %d for %s", symbolLeverage, symbolInfo.MaxLeverage, s.symbol)
		return s.newSyncError(leverageErr, leverageErr.Error(), internal.SymbolLeverageExceedsMaxErrorCode)
	}

	s.logger.Info("Switching leverage",
		"from", fromLeverage,
		"to", symbolLeverage)
	if s.exchange.WebsocketsEnabled(s.mapping.Account) {
		err = s.exchange.SetLeverageWebsocket(s.mapping.Account, s.symbol, symbolLeverage)
	} else {
		err = s.exchange.SetLeverage(s.mapping.Account, s.symbol, symbolLeverage)
	}
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Error setting symbol leverage: %v", err), internal.SymbolAdjustLeverageErrorCode)
	}

	return nil
}

// Sync synchronizes the symbol position
func (s *SymbolSync) Sync() (*metric_types.SymbolMetrics, error) {
	startTime := time.Now()
	settleCoin, err := s.mapping.Account.SettleCoin()
	if err != nil {
		settleCoin = ""
	}
	metrics := &metric_types.SymbolMetrics{
		StartTimestamp: startTime.Unix(),
		Symbol:         s.symbol,
		SettleCoin:     settleCoin,
	}
	if s.targetPosition != nil {
		metrics.TargetSymbol = s.targetPosition.Coin
	}

	sizes := s.ComputePositionSizes()
	tpslOrders := s.SymbolTpslOrders()
	s.fillMetrics(metrics, sizes, tpslOrders)

	if s.targetPosition == nil {
		// No target, close the position
		metrics.EndTimestamp = time.Now().Unix()
		err := s.closePosition()
		if err != nil {
			errorCode := GetSymbolSyncErrorCode(err)
			metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
		}
		return metrics, err
	}

	if err := s.checkAndSetSymbolMarginTypeToIsolated(); err != nil {
		errorCode := GetSymbolSyncErrorCode(err)
		metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
		return metrics, err
	}

	// Adjust SL of current position as needed, unless Hyperliquid TP/SL
	// adjustments are disabled for this account. In that case, close the existing TP/SL orders.
	if s.isTpslDisabled() {
		s.logger.Info("Hyperliquid TP/SL adjustments disabled, cancelling existing TP/SL orders")
		if err := s.closeExistingTpsl(); err != nil {
			// Log error but continue with position adjustment
			errorCode := GetSymbolSyncErrorCode(err)
			metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
			s.logger.Warn("Error cancelling TP/SL orders", "error", err)
		}
	} else {
		// Add a random sleep between 0 and max sleep seconds
		sleepTime := rand.Float64() * s.tpslMaxSleepSec
		time.Sleep(time.Duration(sleepTime * float64(time.Second)))

		if err := s.adjustSl(tpslOrders); err != nil {
			// Log error but continue with position adjustment
			errorCode := GetSymbolSyncErrorCode(err)
			metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
			s.logger.Warn("Error adjusting SL", "error", err)
		}
	}

	err = s.syncLeverageAndPosition(sizes, metrics)
	metrics.EndTimestamp = time.Now().Unix()
	return metrics, err
}

// syncLeverageAndPosition runs adjustLeverage and adjustPosition in the order
// required by positionFirstBeforeLeverage. SetLeverage is skipped when no trade
// will be placed and there is no open position to update leverage for.
func (s *SymbolSync) syncLeverageAndPosition(sizes PositionSizes, metrics *metric_types.SymbolMetrics) error {
	skipTrade := sizes.RelativeDelta() < s.cfg.MinRelativePositionDelta
	if skipTrade {
		s.logger.Debug("Relative delta below threshold, will not trade",
			"relative_delta_pct", sizes.RelativeDelta()*100,
			"threshold_pct", s.cfg.MinRelativePositionDelta*100)
	}

	var symbolInfo *clients.SymbolInfo
	var ticker *clients.Ticker
	if !skipTrade {
		var err error
		symbolInfo, ticker, err = s.fetchSymbolInfoAndTicker()
		if err != nil {
			errorCode := GetSymbolSyncErrorCode(err)
			metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
			return err
		}

		orderSize := math.Abs(sizes.Delta())
		if s.orderBelowExchangeMinimum(orderSize, ticker.LastPrice, symbolInfo) {
			s.logger.Debug("Delta below exchange minimum, will not trade",
				"order_size", orderSize,
				"order_value", orderSize*ticker.LastPrice,
				"rounded_order_size", roundDownToQtyGrid(math.Min(orderSize, symbolInfo.MaxMkOrderQty), symbolInfo),
				"min_qty", symbolInfo.MinOrderQty,
				"qty_step", symbolInfo.QtyStep,
				"min_value", s.minOrderNotionalValue(symbolInfo),
				"relative_delta_pct", sizes.RelativeDelta()*100,
			)
			if sizes.Current == 0 {
				// Opening would fail — report without calling SetLeverage first.
				err := s.orderBelowMinimumError(orderSize, ticker.LastPrice, symbolInfo)
				errorCode := GetSymbolSyncErrorCode(err)
				metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
				return err
			}
			// Existing position: tiny size adjust is a silent skip; leverage may
			// still need updating below.
			skipTrade = true
		}

		// Block risk-increasing trades whose exchange notional diverges from the
		// expected Hyperliquid notional (wrong map, or exchange ticker meaning drift).
		if !skipTrade && math.Abs(sizes.Target) > math.Abs(sizes.Current) {
			if err := s.notionalDivergenceError(sizes, ticker.LastPrice); err != nil {
				errorCode := GetSymbolSyncErrorCode(err)
				metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
				return err
			}
		}
	}

	if skipTrade && s.currentPosition == nil {
		// Nothing to trade and no open position whose leverage we manage.
		return nil
	}

	runLeverage := func() error {
		err := s.adjustLeverage()
		if err == nil {
			return nil
		}
		errorCode := GetSymbolSyncErrorCode(err)
		metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
		if s.currentPosition != nil && s.currentPosition.Size != 0 && errorCode == internal.SymbolNotEnoughMarginErrorCode {
			s.logger.Error("Error adjusting Leverage. Closing position", "error", err)
			if closeErr := s.closePosition(); closeErr != nil {
				metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: closeErr.Error(), ErrorCode: GetSymbolSyncErrorCode(closeErr)})
				s.logger.Warn("Position couldn't be closed", "error", closeErr)
			}
		}
		return err
	}

	runPosition := func() error {
		if skipTrade {
			return nil
		}
		err := s.adjustPosition(sizes, metrics, symbolInfo, ticker)
		if err == nil {
			return nil
		}
		errorCode := GetSymbolSyncErrorCode(err)
		metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: err.Error(), ErrorCode: errorCode})
		if errorCode == internal.SymbolOpenInterestCapErrorCode {
			s.logger.Error("Open interest at cap. Closing position", "error", err)
			if closeErr := s.closePosition(); closeErr != nil {
				metrics.Errors = append(metrics.Errors, metric_types.SymbolError{Exception: closeErr.Error(), ErrorCode: GetSymbolSyncErrorCode(closeErr)})
				s.logger.Warn("Position couldn't be closed", "error", closeErr)
			}
		}
		return err
	}

	if skipTrade {
		// Size is already close enough; still sync leverage for the open position.
		return runLeverage()
	}

	positionFirst := positionFirstBeforeLeverage(sizes, s.cfg.MinRelativePositionDelta)
	if positionFirst {
		s.logger.Info("Adjusting position before leverage")
		if err := runPosition(); err != nil {
			return err
		}
		return runLeverage()
	}
	if err := runLeverage(); err != nil {
		return err
	}
	return runPosition()
}

// fetchSymbolInfoAndTicker loads symbol filters and the latest ticker concurrently.
func (s *SymbolSync) fetchSymbolInfoAndTicker() (*clients.SymbolInfo, *clients.Ticker, error) {
	var symbolInfo *clients.SymbolInfo
	var ticker *clients.Ticker
	var symbolInfoErr, tickerErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		symbolInfo, symbolInfoErr = s.exchange.SymbolInfo(s.account, s.symbol, true)
	}()

	go func() {
		defer wg.Done()
		ticker, tickerErr = s.exchange.Ticker(s.account, s.symbol)
	}()

	wg.Wait()

	if symbolInfoErr != nil {
		return nil, nil, s.newSyncError(symbolInfoErr, fmt.Sprintf("Error getting symbol info: %v", symbolInfoErr), internal.SymbolSymbolInfoErrorCode)
	}

	if tickerErr != nil {
		return nil, nil, s.newSyncError(tickerErr, fmt.Sprintf("Error getting ticker: %v", tickerErr), internal.SymbolTickerErrorCode)
	}

	if ticker == nil {
		return nil, nil, &internal.SymbolSyncError{
			AccountName: s.mapping.Name,
			Symbol:      s.symbol,
			Message:     "No ticker found, cannot order",
			FromError:   nil,
			ErrorCode:   internal.SymbolTickerErrorCode,
		}
	}

	return symbolInfo, ticker, nil
}

// orderBelowMinimumError builds the sync error used when an opening order is
// below exchange size/notional filters.
func (s *SymbolSync) orderBelowMinimumError(orderSize, lastPrice float64, symbolInfo *clients.SymbolInfo) error {
	minValue := s.minOrderNotionalValue(symbolInfo)
	if orderSize <= symbolInfo.MinOrderQty {
		return &internal.SymbolSyncError{
			AccountName: s.mapping.Name,
			Symbol:      s.symbol,
			Message:     fmt.Sprintf("Order size below min, will not buy: %v", orderSize),
			FromError:   nil,
			ErrorCode:   internal.SymbolOrderSizeBelowMinErrorCode,
		}
	}
	roundedSize := roundDownToQtyGrid(math.Min(orderSize, symbolInfo.MaxMkOrderQty), symbolInfo)
	return &internal.SymbolSyncError{
		AccountName: s.mapping.Name,
		Symbol:      s.symbol,
		Message: fmt.Sprintf(
			"Order value below min, will not buy: order value=%v (size %v rounded to step %v), required min=%v (tpsl_disabled=%v, growth_market=%v)",
			roundedSize*lastPrice,
			roundedSize,
			symbolInfo.QtyStep,
			minValue,
			s.isTpslDisabled(),
			symbolInfo.GrowthMode,
		),
		FromError: nil,
		ErrorCode: internal.SymbolOrderValueBelowMinErrorCode,
	}
}

// adjustPosition places a trade to move current size toward target.
// Relative-delta and exchange-minimum gates are applied in syncLeverageAndPosition
// so SetLeverage is skipped when no trade will be placed.
func (s *SymbolSync) adjustPosition(sizes PositionSizes, metrics *metric_types.SymbolMetrics, symbolInfo *clients.SymbolInfo, ticker *clients.Ticker) error {
	var orderSide string
	var orderSize float64
	if sizes.Delta() >= 0 {
		orderSide = "Buy"
		orderSize = sizes.Delta()
	} else {
		orderSide = "Sell"
		orderSize = -sizes.Delta()
	}

	return s.placeOrder(ticker, symbolInfo, orderSize, orderSide, false, metrics)
}

// adjustSl adjusts the stop loss and removes any leftover take profit order
func (s *SymbolSync) adjustSl(tpslOrders TpslOrders) error {
	if s.currentPosition == nil {
		return nil
	}

	symbolInfo, err := s.exchange.SymbolInfo(s.account, s.symbol, true)
	if err != nil {
		return err
	}

	stopLoss, err := s.ComputeStopLoss(
		s.currentPosition.Side,
		s.currentPosition.Size,
		s.currentPosition.MarkPrice,
		symbolInfo,
	)
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Error computing stop loss: %v", err), internal.SymbolComputePricePointsErrorCode)
	}

	shouldAdjust := false
	var sl float64 = 0.0
	slOrder := tpslOrders.slOrder
	if slOrder == nil {
		// No SL set, should certainly set one
		s.logger.Warn("Position has no SL, will add")
		shouldAdjust = true
	} else {
		sl = slOrder.TriggerPrice
		mp := s.currentPosition.MarkPrice

		if mp > 0 {
			// Check if the mark price is too close to the SL limit,
			// or if the computed SL has moved too far from the current SL
			slMpDistance := math.Abs(mp-sl) / mp
			slOldDistance := math.Abs(stopLoss-sl) / stopLoss

			s.logger.Debug("SL status",
				"mark_price", mp,
				"stop_loss", sl,
				"sl_distance", slMpDistance)

			if slMpDistance < s.cfg.TpslMpReadjustDistance ||
				slOldDistance > s.cfg.TpslReadjustDistance {
				shouldAdjust = true
			}
		} else {
			s.logger.Warn("Mark price is somehow 0, will adjust")
			shouldAdjust = true
		}
	}

	// The service no longer manages take profit: setting the SL also cancels
	// any leftover TP order, so its presence forces an adjustment.
	if tpslOrders.tpOrder != nil {
		s.logger.Info("Position has a leftover TP order, will remove")
		shouldAdjust = true
	}

	if shouldAdjust {
		slChange := math.Inf(1)
		deadband := symbolInfo.TickSize
		if sl > 0 {
			slChange = math.Abs(stopLoss - sl)
			// Only re-push once the ideal SL has drifted by more than a fraction
			// of the stop's current distance from the mark price.
			stopDistance := math.Abs(s.currentPosition.MarkPrice - sl)
			deadband = math.Max(symbolInfo.TickSize, s.cfg.TpslReadjustFraction*stopDistance)
		}

		if slChange >= deadband || tpslOrders.tpOrder != nil {
			// Apply new SL
			s.logger.Info("Adjusting SL",
				"new_sl", stopLoss,
				"sl_change", slChange,
				"deadband", deadband,
				"tick_size", symbolInfo.TickSize)
			err := s.setPositionSl(stopLoss, tpslOrders)
			if err != nil {
				return s.newSyncError(err, fmt.Sprintf("Error setting SL: %v", err), internal.SymbolSetPositionTpslErrorCode)
			}
			return nil
		} else {
			s.logger.Debug("New SL too close to current to change",
				"new_sl", stopLoss,
				"sl_change", slChange,
				"deadband", deadband)
		}
	}

	return nil
}

// setPositionSl sets the stop loss via websocket when possible. Take profit is
// passed as nil so any leftover TP order gets canceled.
func (s *SymbolSync) setPositionSl(stopLoss float64, tpslOrders TpslOrders) error {
	if s.exchange.WebsocketsEnabled(s.account) {
		return s.exchange.SetPositionTpslWebsocket(
			s.account,
			s.symbol,
			s.currentPosition.Side,
			nil,
			&stopLoss,
			tpslOrders.tpOrder,
			tpslOrders.slOrder,
		)
	}

	s.logger.Info("Using REST for SL")
	return s.exchange.SetPositionTpsl(
		s.account,
		s.symbol,
		s.currentPosition.Side,
		nil,
		&stopLoss,
	)
}

// ComputePositionSizes computes current and target position sizes
func (s *SymbolSync) ComputePositionSizes() PositionSizes {
	var currentQuantity float64
	if s.currentPosition == nil {
		currentQuantity = 0.0
		s.logger.Debug("No current position found")
	} else {
		currentQuantity = s.currentPosition.Size * float64(utils.SideToMultiplier(s.currentPosition.Side))
		s.logger.Debug("Current position",
			"size", currentQuantity,
			"entry_price", s.currentPosition.EntryPrice)
	}

	var targetQuantity float64
	if s.targetPosition == nil {
		targetQuantity = 0.0
		s.logger.Debug("No target position found")
	} else {
		targetQuantityRaw := s.targetPosition.Szi.Float64()
		targetQuantity = targetQuantityRaw * s.accountState.AccountNormalization
		s.logger.Debug("Target quantity",
			"normalized", targetQuantity,
			"actual", targetQuantityRaw,
			"entry_price", s.targetPosition.EntryPx.Float64())
	}

	return PositionSizes{
		Current: currentQuantity,
		Target:  targetQuantity,
	}
}

// ComputeStopLoss computes the stop loss price point
func (s *SymbolSync) ComputeStopLoss(side string, size float64, price float64, info *clients.SymbolInfo) (float64, error) {
	equity := s.accountState.CurrentBalance.Equity
	minEquity, err := s.mapping.MinimumEquity()
	if err != nil {
		return 0, err
	}
	totalRiskFraction, err := s.account.MaxFractionAtRisk()
	if err != nil {
		return 0, err
	}

	// Calculate position weights
	var otherPositionWeights []float64
	for _, p := range s.accountState.CurrentPositions {
		if p.Symbol != s.symbol {
			otherPositionWeights = append(otherPositionWeights, math.Sqrt(math.Abs(p.Size*p.MarkPrice)))
		}
	}
	currentPositionWeight := math.Sqrt(math.Abs(size * price))

	totalWeight := currentPositionWeight
	for _, w := range otherPositionWeights {
		totalWeight += w
	}

	positionRiskProportion := currentPositionWeight / totalWeight

	minEquityReference := s.cfg.SlMinEquityFactor * minEquity
	allowedRisk := math.Max(0,
		totalRiskFraction*positionRiskProportion*(equity-minEquityReference))
	riskInPriceDistance := allowedRisk / size

	minPrice := info.MinPrice
	maxPrice := info.MaxPrice
	if info.TickSize <= 0 {
		return 0, fmt.Errorf("invalid tick size %v for symbol %s", info.TickSize, s.symbol)
	}
	tickSizePrecision := int(math.Max(0, math.Ceil(-math.Log10(info.TickSize))))

	var stopLoss float64
	if side == "Buy" {
		stopLoss = math.Max(minPrice,
			math.Max(price*s.cfg.MinTpslRatio, price-riskInPriceDistance))
	} else {
		stopLoss = math.Min(maxPrice, price+riskInPriceDistance)
	}
	stopLoss = math.Round(stopLoss*math.Pow10(tickSizePrecision)) / math.Pow10(tickSizePrecision)
	return stopLoss, nil
}

// closePosition closes the current position
func (s *SymbolSync) closePosition() error {
	if s.currentPosition == nil {
		s.logger.Debug("No current position, nothing to close")
		return nil
	}

	s.logger.Info("Placing full close order",
		"side", s.currentPosition.Side,
		"size", s.currentPosition.Size,
		"markPrice", s.currentPosition.MarkPrice,
		"symbol", s.currentPosition.Symbol,
	)
	err := s.exchange.ClosePosition(s.account, *s.currentPosition)
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Error closing position: %v", err), internal.SymbolClosePositionErrorCode)
	}

	s.logger.Debug("Closed position")
	return nil
}

func (s *SymbolSync) isTpslDisabled() bool {
	return !s.target.IsTpslSafetyEnabled()
}

func (s *SymbolSync) closeExistingTpsl() error {
	orders := s.symbolTpSlOrders()
	if len(orders) == 0 {
		return nil
	}

	err := s.exchange.CancelStopOrders(s.account, orders)
	if err != nil {
		return s.newSyncError(
			err,
			fmt.Sprintf("Error cancelling TP/SL orders: %v", err),
			internal.SymbolSetPositionTpslErrorCode,
		)
	}
	return nil
}

func (s *SymbolSync) symbolTpSlOrders() []clients.OpenStopOrder {
	orders := make([]clients.OpenStopOrder, 0, 2)
	for _, order := range s.accountState.OpenStopOrders {
		if order.Symbol != s.symbol {
			continue
		}
		if order.OrderType != "TP" && order.OrderType != "SL" {
			continue
		}
		orders = append(orders, order)
	}
	return orders
}

// minOrderNotionalValue returns the minimum notional value an order must reach.
// On Hyperliquid growth-mode markets, while TP/SL adjustments are disabled, it
// is raised to HyperliquidTpslDisabledMinNotionalValue so that opening a
// position generates enough traded volume to raise the rate-limit cap by at
// least the requests it consumes.
func (s *SymbolSync) minOrderNotionalValue(symbolInfo *clients.SymbolInfo) float64 {
	minValue := s.cfg.MinNotionalSafetyFactor * symbolInfo.MinNotionalValue
	if s.isTpslDisabled() && symbolInfo.GrowthMode {
		minValue = math.Max(minValue, s.cfg.HyperliquidTpslDisabledMinNotionalValue)
	}
	return minValue
}

// orderBelowExchangeMinimum reports whether placeOrder would find no valid order
// for orderSize. It runs the same fractioning as placeOrder, because a raw size
// above the minimums can fall below them once rounded down to the quantity step.
func (s *SymbolSync) orderBelowExchangeMinimum(orderSize, lastPrice float64, symbolInfo *clients.SymbolInfo) bool {
	return len(s.fractionOrderSizes(orderSize, lastPrice, symbolInfo)) == 0
}

// roundDownToQtyGrid rounds size down onto the exchange quantity grid
// (minQty + k*step), the size the exchange actually receives.
func roundDownToQtyGrid(size float64, symbolInfo *clients.SymbolInfo) float64 {
	minQty := symbolInfo.MinOrderQty
	return minQty + math.Floor((size-minQty)/symbolInfo.QtyStep)*symbolInfo.QtyStep
}

func (s *SymbolSync) fractionOrderSizes(orderSize, lastPrice float64, symbolInfo *clients.SymbolInfo) []string {
	// Compute limit quantities from exchange info
	minQty := symbolInfo.MinOrderQty
	minQtyByValue := s.minOrderNotionalValue(symbolInfo) / lastPrice
	stepPrecision := int(math.Max(0, math.Ceil(-math.Log10(symbolInfo.QtyStep))))

	// Fraction the order into multiple batches
	var sizes []string
	remaining := orderSize

	for range s.cfg.MaxFractionedOrders {
		if remaining <= minQty {
			break
		}

		// The next fractioned order is what remains, or at most max_qty
		raw := math.Min(remaining, symbolInfo.MaxMkOrderQty)
		// Round down according to the step size
		rounded := roundDownToQtyGrid(raw, symbolInfo)

		// Format it as Bybit expects a string
		roundedStr := fmt.Sprintf("%.*f", stepPrecision, rounded)

		// Check that the order is not too small in terms of quote value
		if rounded > minQtyByValue {
			sizes = append(sizes, roundedStr)
		}

		// Decrease the remaining order size
		remaining -= rounded
	}
	return sizes
}

func checkNotionalDivergence(
	sizes PositionSizes,
	expectedNotional float64,
	exchangeLastPrice float64,
	maxRatio float64,
) (ok bool, desiredNotional float64, ratio float64) {
	desiredNotional = math.Abs(sizes.Target) * exchangeLastPrice
	if expectedNotional <= 0 {
		if math.Abs(sizes.Target) > 0 {
			return false, desiredNotional, math.Inf(1)
		}
		return true, desiredNotional, 0
	}
	ratio = desiredNotional / expectedNotional
	return ratio <= maxRatio, desiredNotional, ratio
}

func (s *SymbolSync) notionalDivergenceError(sizes PositionSizes, exchangeLastPrice float64) error {
	normalization := s.accountState.AccountNormalization
	expected := 0.0
	if s.targetPosition != nil {
		expected = math.Abs(s.targetPosition.PositionValue.Float64()) * normalization
	}
	ok, desired, ratio := checkNotionalDivergence(
		sizes,
		expected,
		exchangeLastPrice,
		s.cfg.MaxNotionalDivergenceRatio,
	)
	if ok {
		return nil
	}

	hlCoin := ""
	if s.targetPosition != nil {
		hlCoin = s.targetPosition.Coin
	}
	s.logger.Error("Notional divergence safety gate blocked trade",
		"hl_coin", hlCoin,
		"exchange_symbol", s.symbol,
		"expected_notional", expected,
		"desired_notional", desired,
		"ratio", ratio,
		"max_ratio", s.cfg.MaxNotionalDivergenceRatio,
		"exchange_price", exchangeLastPrice,
		"normalization", normalization,
	)

	return &internal.SymbolSyncError{
		AccountName: s.mapping.Name,
		Symbol:      s.symbol,
		Message: fmt.Sprintf(
			"Desired notional %.4g diverges from expected Hyperliquid notional %.4g (ratio=%g, max=%g); blocking risk-increasing trade for mapped symbol %s (hl_coin=%s)",
			desired,
			expected,
			ratio,
			s.cfg.MaxNotionalDivergenceRatio,
			s.symbol,
			hlCoin,
		),
		FromError: nil,
		ErrorCode: internal.SymbolNotionalSafetyErrorCode,
	}
}

// PlaceOrder places an order with fractioning logic (public, without metrics)
func (s *SymbolSync) PlaceOrder(ticker *clients.Ticker, symbolInfo *clients.SymbolInfo, orderSize float64, orderSide string, reduceOnly bool) error {
	return s.placeOrder(ticker, symbolInfo, orderSize, orderSide, reduceOnly, nil)
}

// placeOrder places an order with fractioning logic
func (s *SymbolSync) placeOrder(ticker *clients.Ticker, symbolInfo *clients.SymbolInfo, orderSize float64, orderSide string, reduceOnly bool, metrics *metric_types.SymbolMetrics) error {
	// Compute limit quantities from exchange info
	minQty := symbolInfo.MinOrderQty
	minValue := s.minOrderNotionalValue(symbolInfo)

	// Fraction the order into multiple batches
	fractionedOrderSizes := s.fractionOrderSizes(orderSize, ticker.LastPrice, symbolInfo)

	// Check if there was actually any order to place
	if len(fractionedOrderSizes) == 0 {
		if orderSize <= minQty {
			return &internal.SymbolSyncError{
				AccountName: s.mapping.Name,
				Symbol:      s.symbol,
				Message:     fmt.Sprintf("Order size below min, will not buy: %v", orderSize),
				FromError:   nil,
				ErrorCode:   internal.SymbolOrderSizeBelowMinErrorCode,
			}
		} else if orderSize*ticker.LastPrice < minValue {
			// When Hyperliquid TP/SL placement is disabled this also covers
			// the artificial growth-market minimum notional: we reuse
			// SymbolOrderValueBelowMinErrorCode (4010) so no new UI surface is
			// needed.
			return &internal.SymbolSyncError{
				AccountName: s.mapping.Name,
				Symbol:      s.symbol,
				Message: fmt.Sprintf(
					"Order value below min, will not buy: order value=%v, required min=%v (tpsl_disabled=%v, growth_market=%v)",
					orderSize*ticker.LastPrice,
					minValue,
					s.isTpslDisabled(),
					symbolInfo.GrowthMode,
				),
				FromError: nil,
				ErrorCode: internal.SymbolOrderValueBelowMinErrorCode,
			}
		} else {
			return &internal.SymbolSyncError{
				AccountName: s.mapping.Name,
				Symbol:      s.symbol,
				Message: fmt.Sprintf(
					"Order cannot be placed: size=%v meets exchange minimums but cannot be split into valid fractions (minQty=%v, step=%v, minNotional=%v, price=%v)",
					orderSize,
					symbolInfo.MinOrderQty,
					symbolInfo.QtyStep,
					minValue,
					ticker.LastPrice,
				),
				FromError: nil,
				ErrorCode: internal.SymbolOrderNotPlaceableErrorCode,
			}
		}
	}

	stopLoss, err := s.ComputeStopLoss(orderSide, orderSize, ticker.LastPrice, symbolInfo)
	if err != nil {
		return s.newSyncError(err, fmt.Sprintf("Error computing stop loss: %v", err), internal.SymbolComputePricePointsErrorCode)
	}

	s.logger.Info("Placing order",
		"size", orderSize,
		"side", orderSide,
		"fractions", fractionedOrderSizes)

	var stopLossPtr *float64
	if !s.isTpslDisabled() {
		stopLossPtr = &stopLoss
	}

	// Place all fractioned orders concurrently
	var wg sync.WaitGroup
	var ordersErrors []error
	var trades []metric_types.TradeMetrics
	var mu sync.Mutex

	for _, size := range fractionedOrderSizes {
		wg.Add(1)
		go func(sizeStr string) {
			defer wg.Done()
			tradeStart := time.Now()
			err := s.PlaceOrderIndividual(ticker, orderSide, sizeStr, reduceOnly, stopLossPtr)
			tradeEnd := time.Now()

			mu.Lock()
			if err != nil {
				ordersErrors = append(ordersErrors, s.newSyncError(err, fmt.Sprintf("Error placing order: %v", err), internal.SymbolPlaceOrderErrorCode))
			} else {
				// Record successful trade
				sizeFloat, _ := strconv.ParseFloat(sizeStr, 64)
				trades = append(trades, metric_types.TradeMetrics{
					StartTimestamp: tradeStart.Unix(),
					EndTimestamp:   tradeEnd.Unix(),
					Side:           orderSide,
					Size:           sizeFloat,
					Value:          sizeFloat * ticker.LastPrice,
				})
			}
			mu.Unlock()
		}(size)
	}

	wg.Wait()

	// Add trades to metrics
	if metrics != nil {
		metrics.Trades = append(metrics.Trades, trades...)
	}

	if len(ordersErrors) > 0 {
		// prioritized errors: return the first occurrence of these specific codes
		for _, err := range ordersErrors {
			if symbolSyncError, ok := err.(*internal.SymbolSyncError); ok {
				switch symbolSyncError.ErrorCode {
				case internal.SymbolNotEnoughMarginErrorCode,
					internal.SymbolTradFiAgreementRequiredErrorCode,
					internal.SymbolPositionExceedsMarginTierErrorCode,
					internal.SymbolOpenInterestCapErrorCode:
					return symbolSyncError
				}
			}
		}

		// else we simply say that the place order failed
		return &internal.SymbolSyncError{
			AccountName: s.mapping.Name,
			Symbol:      s.symbol,
			Message:     fmt.Sprintf("errors placing orders: %v", ordersErrors),
			FromError:   nil,
			ErrorCode:   internal.SymbolPlaceOrderErrorCode,
		}
	}

	s.logger.Debug("Placed order", "side", orderSide)
	return nil
}

// PlaceOrderIndividual places a single order with a stop loss and no take profit
func (s *SymbolSync) PlaceOrderIndividual(ticker *clients.Ticker, orderSide string, orderSize string, reduceOnly bool, stopLoss *float64) error {
	_, err := s.exchange.PlaceOrder(
		s.account,
		s.symbol,
		orderSide,
		orderSize,
		nil,
		stopLoss,
		reduceOnly,
		false,
	)
	return err
}

// fillMetrics fills the symbol metrics with current data
func (s *SymbolSync) fillMetrics(metrics *metric_types.SymbolMetrics, sizes PositionSizes, tpslOrders TpslOrders) {
	if metrics == nil {
		return
	}

	metrics.TargetSize = sizes.Target

	if s.currentPosition != nil {
		metrics.Size = s.currentPosition.Size * float64(utils.SideToMultiplier(s.currentPosition.Side))
		metrics.Value = s.currentPosition.PositionValue
		metrics.Price = s.currentPosition.MarkPrice
		metrics.EntryPrice = s.currentPosition.EntryPrice
		metrics.PnL = s.currentPosition.UnrealizedPnl
		metrics.Leverage = s.currentPosition.Leverage
		metrics.Liquidation = s.currentPosition.LiqPrice
	}

	if tpslOrders.slOrder != nil {
		metrics.StopLoss = tpslOrders.slOrder.TriggerPrice
	}
}

// TpslOrders represents take profit and stop loss orders. The tpOrder is only
// tracked so leftover take profit orders can be canceled: the service no
// longer sets take profit.
type TpslOrders struct {
	tpOrder *clients.OpenStopOrder
	slOrder *clients.OpenStopOrder
}

// SymbolOpenStopOrders returns the open stop orders for the symbol
func (s *SymbolSync) SymbolTpslOrders() TpslOrders {
	openStopOrders := []clients.OpenStopOrder{}
	for _, order := range s.accountState.OpenStopOrders {
		if order.Symbol == s.symbol {
			openStopOrders = append(openStopOrders, order)
		}
	}
	var tpOrder *clients.OpenStopOrder
	var slOrder *clients.OpenStopOrder
	for _, order := range openStopOrders {
		switch order.OrderType {
		case "TP":
			tpOrder = &order
		case "SL":
			slOrder = &order
		}
	}

	return TpslOrders{
		tpOrder: tpOrder,
		slOrder: slOrder,
	}
}

// newSyncError creates a SymbolSyncError, mapping known exchange sentinel errors to their codes.
func (s *SymbolSync) newSyncError(err error, message string, defaultCode internal.SyncErrorCode) *internal.SymbolSyncError {
	code := defaultCode
	if errors.Is(err, clients.ErrNotEnoughMargin) {
		code = internal.SymbolNotEnoughMarginErrorCode
	} else if errors.Is(err, clients.ErrSymbolNotExist) {
		code = internal.SymbolNotExistErrorCode
	} else if errors.Is(err, clients.ErrMachineInternalClock) {
		code = internal.SymbolMachineInternalClockErrorCode
	} else if errors.Is(err, clients.ErrTradFiAgreementRequired) {
		code = internal.SymbolTradFiAgreementRequiredErrorCode
	} else if errors.Is(err, clients.ErrPositionExceedsMarginTier) {
		code = internal.SymbolPositionExceedsMarginTierErrorCode
	} else if errors.Is(err, clients.ErrSubaccountLeverageRestricted) {
		code = internal.SymbolSubaccountLeverageRestrictedErrorCode
	} else if errors.Is(err, clients.ErrNewFuturesAccountLeverageRestricted) {
		code = internal.SymbolNewFuturesAccountLeverageRestrictedErrorCode
	} else if errors.Is(err, clients.ErrOpenInterestCap) {
		code = internal.SymbolOpenInterestCapErrorCode
	} else if errors.Is(err, clients.ErrInvalidApiKeyOrPermissions) {
		code = internal.AccountInvalidApiKeyOrPermissionsErrorCode
	} else if errors.Is(err, clients.ErrApiKeyNotSet) {
		code = internal.AccountApiKeyNotSetErrorCode
	}

	return &internal.SymbolSyncError{
		AccountName: s.mapping.Name,
		Symbol:      s.symbol,
		Message:     message,
		FromError:   &err,
		ErrorCode:   code,
	}
}

// PositionSizes represents current and target position sizes
type PositionSizes struct {
	Current float64
	Target  float64
}

// Delta returns the difference between target and current
func (ps PositionSizes) Delta() float64 {
	return ps.Target - ps.Current
}

// RelativeDelta returns the relative delta
func (ps PositionSizes) RelativeDelta() float64 {
	norm := math.Max(math.Abs(ps.Current), math.Abs(ps.Target))
	if norm > 0 {
		return math.Abs(ps.Delta()) / norm
	}
	return 0
}

// When adjusting size and leverage for a position we have to be careful with the order of
// operations in two cases: the ones that adjust position size and leverage in the same direction.
// In those cases margin can go up or down in the intermediate state. We make sure it goes down to avoid running out of required margin.
// When size and leverage decrease, adjust size first. And viceversa.
func positionFirstBeforeLeverage(sizes PositionSizes, minRelativeDelta float64) bool {
	if sizes.RelativeDelta() < minRelativeDelta {
		return false
	}
	return math.Abs(sizes.Target) < math.Abs(sizes.Current)
}
