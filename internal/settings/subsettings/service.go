package subsettings

import (
	"time"

	"go.yaml.in/yaml/v4"
)

// DefaultHealthWriteIntervalSeconds is the fallback heartbeat interval used
// when the setting is unset or non-positive.
const DefaultHealthWriteIntervalSeconds = 0.2

// DefaultStopTimeoutSeconds is the fallback shutdown timeout used when the
// setting is unset or non-positive.
const DefaultStopTimeoutSeconds = 20.0

// MinMinNotionalSafetyFactor is the floor applied on load when the configured
// min_notional_safety_factor is below 1.0.
const MinMinNotionalSafetyFactor = 1.0

// MinMaxNotionalDivergenceRatio is the floor applied on load when the configured
// max_notional_divergence_ratio is below 1.0.
const MinMaxNotionalDivergenceRatio = 1.0

// Service settings
type Service struct {
	// Logging level.
	LogLevel string `yaml:"log_level"`
	// Wait time between syncs, in seconds.
	// Determines how often the service will sync the user account and the target account.
	// A lower value will result in more frequent syncs, but also more API calls, risking rate limiting.
	LoopSleepSeconds float64 `yaml:"loop_sleep_seconds"`
	// How much time to wait before closing all positions when the service finds recoverable errors
	// (e.g. if the Miroir API, exchange or Hyperliquid are temporarily unavailable).
	CloseAllMarginSeconds float64 `yaml:"close_all_margin_seconds"`
	// Minimum relative difference between user and target positions to place synchronization trades.
	MinRelativePositionDelta float64 `yaml:"min_relative_position_delta"`
	// MinNotionalSafetyFactor is applied to the exchange MinNotionalValue when sizing and
	// gating orders. We size in contract/asset quantity, not quote value: notional at
	// execution is qty * price. Between fetching the ticker and placing the order, price can
	// drop and push the notional below the exchange floor, causing rejection if we sized
	// right at the minimum. This factor adds a margin above that floor (default 1.10).
	// Values below 1.0 are clamped to 1.0 on load.
	MinNotionalSafetyFactor float64 `yaml:"min_notional_safety_factor"`
	// MaxNotionalDivergenceRatio caps how much larger the desired position notional at the
	// exchange price may be versus the expected Hyperliquid notional (HL positionValue ×
	// normalization). For a correct map this is ~exchangePrice/HL mark; default 1.5 (~50%)
	// blocks wrong-asset mappings while allowing cross-venue basis and illiquid dislocations.
	// Values below 1.0 are clamped to 1.0 on load. Only applied to risk-increasing trades.
	MaxNotionalDivergenceRatio float64 `yaml:"max_notional_divergence_ratio"`
	MinTpslRatio               float64 `yaml:"min_tpsl_ratio"`
	SlMinEquityFactor          float64 `yaml:"sl_min_equity_factor"`
	TpslMpReadjustDistance     float64 `yaml:"tpsl_mp_readjust_distance"`
	TpslReadjustDistance       float64 `yaml:"tpsl_readjust_distance"`
	TpslReadjustFraction       float64 `yaml:"tpsl_readjust_fraction"`
	// Optional YAML-level overrides applied on top of the default symbol mapping.
	// Shape: settleCoin → hyperSymbol → exchangeSymbol.
	// Example : { "USDT": { "TURBO": "1000TURBOUSDT" } }
	// Use null as the exchange symbol to mark a Hyperliquid coin as
	// explicitly unmapped on that settle coin (ErrSymbolNotExist).
	// Example : { "USDT": { "PURR": "" } }
	HyperToBybitOverrides   map[string]map[string]string `yaml:"hyper_to_bybit_overrides"`
	HyperToBinanceOverrides map[string]map[string]string `yaml:"hyper_to_binance_overrides"`
	HyperExcludedSymbols    []string                     `yaml:"hyper_excluded_symbols"`
	// TTL (in seconds) for the symbol-mapping cache fetched from the backend API.
	SymbolMappingTtlSeconds       float64 `yaml:"symbol_mapping_ttl_seconds"`
	MaxFractionedOrders           int     `yaml:"max_fractioned_orders"`
	AccountSyncMaxSleepSeconds    float64 `yaml:"account_sync_max_sleep_seconds"`
	SymbolSyncTpslMaxSleepSeconds float64 `yaml:"symbol_sync_tpsl_max_sleep_seconds"`
	LogApiResponses               bool    `yaml:"log_api_responses"`
	// Interval, in seconds, at which the service writes its health heartbeat
	// to the SQLite database the GUI polls. The GUI considers the service alive
	// when the latest heartbeat is newer than twice this interval.
	HealthWriteIntervalSeconds float64 `yaml:"health_write_interval_seconds"`
	// Minimum time, in seconds, between two exchange cache refreshes triggered on
	// sync errors. If refreshing the cache once didn't fix the issue, refreshing it
	// again immediately won't help either, so refreshes are throttled to this interval.
	MinCacheRefreshSeconds float64 `yaml:"min_cache_refresh_seconds"`
	// Interval, in seconds, at which the Hyperliquid request-rate-limit usage
	// (info userRateLimit) is polled per Hyperliquid destination account. The
	// usage is reported to the backend via loop metrics; polling is intentionally
	// infrequent to avoid spending rate-limit budget on the check itself.
	HyperliquidRateLimitPollSeconds float64 `yaml:"hyperliquid_rate_limit_poll_seconds"`
	// Minimum notional value (in quote currency) required to open a position on a
	// Hyperliquid growth-mode market while TP/SL placing is disabled for the
	// account. Opening such a position spends a fixed number of requests; requiring
	// this notional ensures the traded volume raises the account's rate-limit cap
	// by at least that many requests, keeping the account rate-limit-neutral.
	HyperliquidTpslDisabledMinNotionalValue float64 `yaml:"hyperliquid_tpsl_disabled_min_notional_value"`
	// Maximum time, in seconds, the service waits for the copy loop to shut down
	// cleanly when stopped. Past this delay the service exits anyway, so a stuck
	// request can never prevent it from stopping.
	StopTimeoutSeconds float64 `yaml:"stop_timeout_seconds"`
}

func DefaultService() Service {
	return Service{
		LogLevel:                                "INFO",
		LoopSleepSeconds:                        5.0,
		CloseAllMarginSeconds:                   43200.0,
		MinRelativePositionDelta:                0.02,
		MinNotionalSafetyFactor:                 1.1,
		MaxNotionalDivergenceRatio:              1.5,
		MinTpslRatio:                            0.15,
		SlMinEquityFactor:                       0.75,
		TpslMpReadjustDistance:                  0.2,
		TpslReadjustDistance:                    0.1,
		TpslReadjustFraction:                    0.2,
		HyperToBybitOverrides:                   map[string]map[string]string{},
		HyperToBinanceOverrides:                 map[string]map[string]string{},
		HyperExcludedSymbols:                    []string{},
		SymbolMappingTtlSeconds:                 600.0,
		MaxFractionedOrders:                     5,
		AccountSyncMaxSleepSeconds:              1.0,
		SymbolSyncTpslMaxSleepSeconds:           1.0,
		LogApiResponses:                         false,
		HealthWriteIntervalSeconds:              DefaultHealthWriteIntervalSeconds,
		MinCacheRefreshSeconds:                  300.0,
		HyperliquidRateLimitPollSeconds:         60.0,
		HyperliquidTpslDisabledMinNotionalValue: 20.0,
		StopTimeoutSeconds:                      DefaultStopTimeoutSeconds,
	}
}

// HealthInterval returns the heartbeat write/poll interval, falling back to the
// default when the configured value is unset or non-positive.
func (s *Service) HealthInterval() time.Duration {
	secs := s.HealthWriteIntervalSeconds
	if secs <= 0 {
		secs = DefaultHealthWriteIntervalSeconds
	}
	return time.Duration(secs * float64(time.Second))
}

// StopTimeout returns the maximum clean shutdown duration, falling back to the
// default when the configured value is unset or non-positive.
func (s *Service) StopTimeout() time.Duration {
	secs := s.StopTimeoutSeconds
	if secs <= 0 {
		secs = DefaultStopTimeoutSeconds
	}
	return time.Duration(secs * float64(time.Second))
}

func (s *Service) UnmarshalYAML(node *yaml.Node) error {
	*s = DefaultService()
	type selfAlias Service
	if err := node.Decode((*selfAlias)(s)); err != nil {
		return err
	}
	if s.MinNotionalSafetyFactor < MinMinNotionalSafetyFactor {
		s.MinNotionalSafetyFactor = MinMinNotionalSafetyFactor
	}
	if s.MaxNotionalDivergenceRatio < MinMaxNotionalDivergenceRatio {
		s.MaxNotionalDivergenceRatio = MinMaxNotionalDivergenceRatio
	}
	return nil
}
