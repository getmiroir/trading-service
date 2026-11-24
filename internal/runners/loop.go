package runners

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/metrics/metrics_queue"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/google/uuid"
)

func GetLoopSyncErrorCode(err error) internal.SyncErrorCode {
	if syncError, ok := err.(*internal.SyncError); ok {
		return syncError.ErrorCode
	}
	return internal.UnknownErrorCode
}

// CopyLoop represents the main copy trading loop
type CopyLoop struct {
	Exchange         *clients.Exchange
	Hyperliquid      *clients.Hyperliquid
	ControlApi       *clients.ControlApi
	MetricsQueue     metrics_queue.MetricsQueue
	SymbolsInfo      map[string]*clients.SymbolInfo
	StartupTime      time.Time
	LastSyncTime     time.Time
	lastCacheRefresh time.Time
	cfg              *settings.CopyTradingSettings
	logger           *slog.Logger
}

// invalidateCacheThrottled invalidates the exchange cache at most once every
// Service.MinCacheRefreshSeconds, even when called on every errored sync.
func (cl *CopyLoop) invalidateCacheThrottled() {
	minInterval := time.Duration(cl.cfg.Service.MinCacheRefreshSeconds * float64(time.Second))
	now := time.Now()
	if !cl.lastCacheRefresh.IsZero() && now.Sub(cl.lastCacheRefresh) < minInterval {
		return
	}
	cl.logger.Debug("Cache refresh")
	cl.lastCacheRefresh = now
	cl.Exchange.InvalidateCache()
}

// NewCopyLoop creates a new CopyLoop instance. Clients are created when Run(cfg) is called.
func NewCopyLoop() *CopyLoop {
	return &CopyLoop{
		SymbolsInfo: make(map[string]*clients.SymbolInfo),
		logger:      logging.NewLogger("copy-loop"),
	}
}

// Run runs the main copy loop with the given settings. Creates clients from cfg at start.
func (cl *CopyLoop) Run(ctx context.Context, cfg *settings.CopyTradingSettings) error {
	cl.cfg = cfg
	accounts := cfg.Exchange.Accounts()
	cl.Hyperliquid = clients.NewHyperliquid(cfg.Source.Hyperliquid, cfg.Service.LogApiResponses)
	controlApi, err := clients.NewControlApi(cfg.ControlApi, accounts)
	if err != nil {
		return &internal.SyncError{
			Message:   err.Error(),
			ErrorCode: internal.LoopNewControlApiErrorCode,
			FromError: &err,
		}
	}
	cl.ControlApi = controlApi
	cl.Exchange = clients.NewExchange(cfg.Exchange, cfg.Service.LogApiResponses, clients.NewSymbolMapper(controlApi, cfg.Service))
	cl.Exchange.ConnectWebsockets(ctx)
	defer cl.Exchange.CloseWebsockets()
	cl.MetricsQueue = metrics_queue.NewInMemoryMetricsQueue(controlApi, cfg.Metrics)
	cl.StartupTime = time.Now()
	cl.LastSyncTime = time.Now()

	cl.logger.Info("Starting copy loop")

	// Start the metrics queue background worker
	cl.MetricsQueue.Start()
	defer cl.MetricsQueue.Stop()

	// Create account syncs for all accounts (bybit and binance)
	accountSyncs := make(map[string]*AccountSync)
	for accountName, account := range accounts {
		accountSyncs[accountName] = NewAccountSync(cl, accountName, account)
	}

	sleepDuration := time.Duration(cl.cfg.Service.LoopSleepSeconds * float64(time.Second))
	for {
		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			cl.logger.Info("Copy loop cancelled, shutting down")
			return ctx.Err()
		case <-timer.C:
		}

		// Sync accounts
		metrics, err := cl.SyncAccounts(accountSyncs)
		if err != nil {
			cl.logger.Error("Error in sync", "error", err)
			cl.invalidateCacheThrottled()
		}

		// Push metrics to the queue
		cl.MetricsQueue.Push(metrics)

		cl.LastSyncTime = time.Now()
	}
}

// SyncAccounts synchronizes all accounts
func (cl *CopyLoop) SyncAccounts(accountSyncs map[string]*AccountSync) (*metric_types.LoopMetrics, error) {
	stepStart := time.Now()
	metrics := &metric_types.LoopMetrics{
		ID:             uuid.New().String(),
		StartTimestamp: stepStart.Unix(),
		Version:        version.Version,
		OS:             runtime.GOOS,
		Platform:       version.Platform,
	}

	cl.logger.Debug("Starting sync",
		"timestamp", float64(stepStart.UnixNano())/1e9,
		"uptime", stepStart.Sub(cl.StartupTime).Seconds())

	cl.logger.Debug("Retrieving mappings")
	mappings, err := cl.ControlApi.GetMappings()
	if err != nil {
		metrics.EndTimestamp = time.Now().Unix()
		metrics.Errors = append(metrics.Errors, metric_types.LoopError{Exception: err.Error(), ErrorCode: internal.LoopGetMappingsErrorCode})
		return metrics, &internal.SyncError{
			Message:   err.Error(),
			ErrorCode: internal.LoopGetMappingsErrorCode,
			FromError: &err,
		}
	}

	cl.logger.Debug("Syncing accounts")

	// Sync all accounts concurrently
	var wg sync.WaitGroup
	var errors []error
	var accountMetrics []metric_types.AccountMetrics
	var mu sync.Mutex

	for _, mapping := range mappings {
		wg.Add(1)
		go func(m clients.CopyMapping) {
			defer wg.Done()
			if accountSync, ok := accountSyncs[m.Name]; ok {
				am, err := accountSync.Sync(m, &cl.cfg.Service)
				mu.Lock()
				if am != nil {
					accountMetrics = append(accountMetrics, *am)
				}
				if err != nil {
					errors = append(errors, err)
				}
				mu.Unlock()
			}
		}(mapping)
	}

	wg.Wait()

	metrics.Accounts = accountMetrics
	metrics.EndTimestamp = time.Now().Unix()

	if len(errors) > 0 {
		// Add errors to metrics
		for _, err := range errors {
			metrics.Errors = append(metrics.Errors, metric_types.LoopError{Exception: err.Error(), ErrorCode: internal.LoopSyncAccountsErrorCode})
		}

		if len(errors) == 1 {
			return metrics, &internal.SyncError{
				Message:   errors[0].Error(),
				ErrorCode: internal.LoopSyncAccountsErrorCode,
				FromError: &errors[0],
			}
		}
		return metrics, &internal.GatheredErrorsError{Errors: errors}
	}

	stepEnd := time.Now()
	cl.logger.Debug("Sync complete",
		"timestamp", float64(stepEnd.UnixNano())/1e9,
		"uptime", stepEnd.Sub(cl.StartupTime).Seconds(),
		"step_duration", stepEnd.Sub(stepStart).Seconds())

	return metrics, nil
}
