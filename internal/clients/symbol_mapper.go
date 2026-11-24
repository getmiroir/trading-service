package clients

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

const (
	symbolMapperSourceHyper   = "hyperliquid"
	symbolMapperTargetBybit   = "bybit"
	symbolMapperTargetBinance = "binance"
)

// Lookup precedence: YAML overrides → backend API cache.
// Empty string means explicit unmapped
type SymbolMapper struct {
	api       *ControlApi
	overrides map[string]map[string]map[string]string // target → settleCoin → hyperSymbol → exchangeSymbol
	cache     map[string]map[string]map[string]string
	loadedAt  time.Time
	ttl       time.Duration
	mu        sync.RWMutex
	logger    *slog.Logger
}

func NewSymbolMapper(api *ControlApi, service subsettings.Service) *SymbolMapper {
	overrides := map[string]map[string]map[string]string{
		symbolMapperTargetBybit:   service.HyperToBybitOverrides,
		symbolMapperTargetBinance: service.HyperToBinanceOverrides,
	}
	return &SymbolMapper{
		api:       api,
		overrides: overrides,
		cache:     make(map[string]map[string]map[string]string),
		ttl:       time.Duration(service.SymbolMappingTtlSeconds * float64(time.Second)),
		logger:    logging.NewLogger("symbol-mapper"),
	}
}

func (m *SymbolMapper) HyperToBybitSymbol(settleCoin, hyperSymbol string) (string, error) {
	return m.resolve(symbolMapperTargetBybit, settleCoin, hyperSymbol)
}

func (m *SymbolMapper) HyperToBinanceSymbol(settleCoin, hyperSymbol string) (string, error) {
	return m.resolve(symbolMapperTargetBinance, settleCoin, hyperSymbol)
}

// InvalidateCache clears the API-backed cache. The next lookup will refetch.
func (m *SymbolMapper) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]map[string]map[string]string)
	m.loadedAt = time.Time{}
}

func (m *SymbolMapper) resolve(target, settleCoin, hyperSymbol string) (string, error) {
	if sym, ok := lookup(m.overrides, target, settleCoin, hyperSymbol); ok {
		if sym == "" {
			return "", fmt.Errorf("symbol %s explicitly unmapped on %s/%s: %w", hyperSymbol, target, settleCoin, ErrSymbolNotExist)
		}
		return sym, nil
	}
	if err := m.ensureLoaded(); err != nil {
		return "", err
	}
	m.mu.RLock()
	sym, ok := lookup(m.cache, target, settleCoin, hyperSymbol)
	m.mu.RUnlock()
	if ok {
		return sym, nil
	}
	return "", fmt.Errorf("no symbol mapping for %s on %s/%s: %w", hyperSymbol, target, settleCoin, ErrSymbolNotExist)
}

func lookup(table map[string]map[string]map[string]string, target, settleCoin, hyperSymbol string) (string, bool) {
	byTarget, ok := table[target]
	if !ok {
		return "", false
	}
	bySettle, ok := byTarget[settleCoin]
	if !ok {
		return "", false
	}
	sym, ok := bySettle[hyperSymbol]
	return sym, ok
}

func (m *SymbolMapper) ensureLoaded() error {
	m.mu.RLock()
	if m.cacheFresh() {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cacheFresh() {
		return nil
	}
	entries, err := m.api.GetSymbolMappings()
	m.logger.Debug("Loading new mapping", "count", len(entries))
	if err != nil {
		m.logger.Error("Failed to load symbol mappings", "error", err)
		return err
	}
	cache := make(map[string]map[string]map[string]string)
	for _, e := range entries {
		if e.Source != symbolMapperSourceHyper {
			continue
		}
		byTarget, ok := cache[e.Target]
		if !ok {
			byTarget = make(map[string]map[string]string)
			cache[e.Target] = byTarget
		}
		bySettle, ok := byTarget[e.SettleCoin]
		if !ok {
			bySettle = make(map[string]string)
			byTarget[e.SettleCoin] = bySettle
		}
		bySettle[e.SourceSymbol] = e.TargetSymbol
	}
	m.cache = cache
	m.loadedAt = time.Now()
	m.logger.Info("Loaded symbol mappings", "entries", len(entries))
	return nil
}

// cacheFresh reports whether the cache was loaded within TTL. Caller holds m.mu (R or W).
func (m *SymbolMapper) cacheFresh() bool {
	if m.loadedAt.IsZero() {
		return false
	}
	if m.ttl <= 0 {
		return true
	}
	return time.Since(m.loadedAt) < m.ttl
}
