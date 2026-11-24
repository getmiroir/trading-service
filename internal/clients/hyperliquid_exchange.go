package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

const hlMarketOrderSlippage = 0.05

// hlMissingOrderError is the exchange message for cancel error type MissingOrder.
// https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/error-responses
const hlMissingOrderError = "Order was never placed, already canceled, or filled."

// HyperliquidSupportedDexes are the perp dexes whose positions are read and mirrored:
// the default one ("") and the builder-deployed ones, whose coins are namespaced
// (e.g. xyz:MU, io:ANTH).
var HyperliquidSupportedDexes = []string{"", "xyz", "io"}

// hlSymbolCache caches the universe entry and the wire asset id for a coin.
// For the default perp dex the asset id is the plain universe index; for
// builder-deployed dexs it is offset (see hlWireAssetID).
type hlSymbolCache struct {
	Entry *HLUniverseEntry
	Index int
}

// HyperliquidExchange is the exchange client for Hyperliquid perpetuals.
// It is separate from the read-only Hyperliquid copy-source client.
type HyperliquidExchange struct {
	client           *http.Client
	websocketClients map[string]*HyperliquidExchangeWebsocket
	symbolCache      map[string]hlSymbolCache // coin → {universe entry, wire asset id}
	perpDexCache     map[string]int           // perp dex name → perp_dex_index (perpDexs array position)
	cacheMu          sync.RWMutex
	nonceSeqs        map[string]*hlNonceSeq // user address → monotonic exchange-action nonces
	nonceMu          sync.Mutex
	semaphore        chan struct{}
	rateLimiter      *RateLimiter
	cfg              subsettings.HyperliquidExchange
	creds            *secrets.SecretCache
	logApiResponses  bool
	logger           *slog.Logger
}

// NewHyperliquidExchange creates a new HyperliquidExchange client.
func NewHyperliquidExchange(cfg subsettings.HyperliquidExchange, logApiResponses bool) *HyperliquidExchange {
	client := &http.Client{
		Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HttpClient.PoolSize,
			MaxIdleConnsPerHost: cfg.HttpClient.PoolSize,
		},
	}

	limits := make([]RateLimit, len(cfg.HttpClient.RateLimiter.Limits))
	for i, l := range cfg.HttpClient.RateLimiter.Limits {
		limits[i] = RateLimit{Requests: l.Requests, Seconds: l.Seconds}
	}

	creds := secrets.NewSecretCache()

	h := &HyperliquidExchange{
		client:          client,
		symbolCache:     make(map[string]hlSymbolCache),
		perpDexCache:    make(map[string]int),
		nonceSeqs:       make(map[string]*hlNonceSeq),
		semaphore:       make(chan struct{}, cfg.HttpClient.MaxConcurrent),
		rateLimiter:     NewRateLimiter(limits),
		cfg:             cfg,
		creds:           creds,
		logApiResponses: logApiResponses,
		logger:          logging.NewLogger("hyperliquid-exchange-client"),
	}

	websocketClients := make(map[string]*HyperliquidExchangeWebsocket)
	for accountKey, account := range cfg.Accounts {
		acc := account
		websocketClients[accountKey] = NewHyperliquidExchangeWebsocket(
			acc,
			cfg.Websockets.PingInterval,
			cfg.Websockets.ReconnectDelay,
			cfg.Websockets.Timeout,
			creds,
			func() (int64, error) { return h.nextActionNonce(acc) },
		)
	}
	h.websocketClients = websocketClients

	return h
}

// ConnectAllWebsockets connects all websocket clients and starts supervisors.
func (h *HyperliquidExchange) ConnectAllWebsockets(ctx context.Context) {
	for accountKey, ws := range h.websocketClients {
		h.logger.Info("Connecting websocket", "account", accountKey)
		ws.ConnectWithSupervisor(ctx)
	}
}

// CloseAllWebsockets closes all websocket clients.
func (h *HyperliquidExchange) CloseAllWebsockets() {
	for accountKey, ws := range h.websocketClients {
		if err := ws.Close(); err != nil {
			h.logger.Warn("Failed to close websocket", "account", accountKey, "error", err)
		}
	}
}

// PlaceMarketOrderWebsocket places a market order via the websocket connection.
// It fetches the mark price via HTTP (needed for the IOC slippage calculation) then posts the action via websocket.
func (h *HyperliquidExchange) PlaceMarketOrderWebsocket(
	accountKey string,
	coin, side, size string,
	takeProfit, stopLoss *float64,
	reduceOnly bool,
) (*HLOrderData, error) {
	account, ok := h.cfg.Accounts[accountKey]
	if !ok {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: account not found: %s", accountKey)
	}
	ws, ok := h.websocketClients[accountKey]
	if !ok || ws == nil {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: websocket client not found: %s", accountKey)
	}

	idx, entry, err := h.assetIndex(account, coin)
	if err != nil {
		return nil, err
	}

	ticker, err := h.Ticker(account, coin)
	if err != nil {
		return nil, err
	}
	markPrice := ticker.MarkPx.Float64()
	if markPrice == 0 {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: mark price is zero for %s", coin)
	}

	isBuy := side == "Buy"
	var limitPx float64
	if isBuy {
		limitPx = markPrice * (1 + hlMarketOrderSlippage)
	} else {
		limitPx = markPrice * (1 - hlMarketOrderSlippage)
	}

	sz, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: invalid size %q: %w", size, err)
	}

	orderMap := hlOrderedMap{
		{"a", idx},
		{"b", isBuy},
		{"p", hlPriceToWire(limitPx, entry.SzDecimals)},
		{"s", hlFloatToWire(sz)},
		{"r", reduceOnly},
		{"t", hlOrderedMap{{"limit", hlOrderedMap{{"tif", "Ioc"}}}}},
	}
	action := hlOrderedMap{
		{"type", "order"},
		{"orders", []interface{}{orderMap}},
		{"grouping", "na"},
	}

	resp, err := ws.postAction(action)
	if err != nil {
		return nil, err
	}

	var inner HLExchangeInner
	if err := json.Unmarshal(resp.Response, &inner); err != nil {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: parse response: %w", err)
	}

	var data HLOrderData
	if err := json.Unmarshal(inner.Data, &data); err != nil {
		return nil, fmt.Errorf("PlaceMarketOrderWebsocket: parse order data: %w", err)
	}

	for _, s := range data.Statuses {
		if s.Error != nil {
			return nil, &internal.HyperliquidExchangeError{Message: *s.Error}
		}
	}

	if !reduceOnly {
		if (takeProfit != nil && *takeProfit > 0) || (stopLoss != nil && *stopLoss > 0) {
			if err := h.placeTpSlOrdersWebsocket(ws, idx, entry.SzDecimals, coin, isBuy, sz, takeProfit, stopLoss, "na"); err != nil {
				h.logger.Warn("PlaceMarketOrderWebsocket: TP/SL placement failed", "coin", coin, "err", err)
			}
		}
	}

	return &data, nil
}

// ClosePositionWebsocket closes a position via the websocket connection.
func (h *HyperliquidExchange) ClosePositionWebsocket(accountKey, coin, side string, size float64) error {
	closeSide := "Sell"
	if side == "Sell" {
		closeSide = "Buy"
	}
	sizeStr := strconv.FormatFloat(size, 'f', -1, 64)
	_, err := h.PlaceMarketOrderWebsocket(accountKey, coin, closeSide, sizeStr, nil, nil, true)
	return err
}

// SetPositionTpslWebsocket sets take profit and stop loss for a position via
// the websocket connection. A nil takeProfit or stopLoss leg is not placed.
func (h *HyperliquidExchange) SetPositionTpslWebsocket(
	accountKey, coin, positionSide string,
	takeProfit, stopLoss *float64,
) error {
	account, ok := h.cfg.Accounts[accountKey]
	if !ok {
		return fmt.Errorf("SetPositionTpslWebsocket: account not found: %s", accountKey)
	}
	ws, ok := h.websocketClients[accountKey]
	if !ok || ws == nil {
		return fmt.Errorf("SetPositionTpslWebsocket: websocket client not found: %s", accountKey)
	}

	idx, entry, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}

	pos, _, err := h.Position(account, coin)
	if err != nil {
		return fmt.Errorf("SetPositionTpslWebsocket: fetch position: %w", err)
	}
	sz := 0.0
	if pos != nil {
		sz = math.Abs(pos.Szi.Float64())
	}

	isBuy := positionSide == "Buy"

	return h.placeTpSlOrdersWebsocket(ws, idx, entry.SzDecimals, coin, isBuy, sz, takeProfit, stopLoss, "positionTpsl")
}

// placeTpSlOrdersWebsocket places TP/SL trigger orders via the websocket connection.
func (h *HyperliquidExchange) placeTpSlOrdersWebsocket(
	ws *HyperliquidExchangeWebsocket,
	assetIdx int,
	szDecimals int,
	coin string,
	positionIsBuy bool,
	sz float64,
	takeProfit, stopLoss *float64,
	grouping string,
) error {
	triggerIsBuy := !positionIsBuy

	var orders []interface{}

	if takeProfit != nil && *takeProfit > 0 {
		orders = append(orders, hlOrderedMap{
			{"a", assetIdx},
			{"b", triggerIsBuy},
			{"p", hlPriceToWire(*takeProfit, szDecimals)},
			{"s", hlFloatToWire(sz)},
			{"r", true},
			{"t", hlOrderedMap{{"trigger", hlOrderedMap{
				{"isMarket", true},
				{"triggerPx", hlPriceToWire(*takeProfit, szDecimals)},
				{"tpsl", "tp"},
			}}}},
		})
	}

	if stopLoss != nil && *stopLoss > 0 {
		orders = append(orders, hlOrderedMap{
			{"a", assetIdx},
			{"b", triggerIsBuy},
			{"p", hlPriceToWire(*stopLoss, szDecimals)},
			{"s", hlFloatToWire(sz)},
			{"r", true},
			{"t", hlOrderedMap{{"trigger", hlOrderedMap{
				{"isMarket", true},
				{"triggerPx", hlPriceToWire(*stopLoss, szDecimals)},
				{"tpsl", "sl"},
			}}}},
		})
	}

	if len(orders) == 0 {
		return nil
	}

	action := hlOrderedMap{
		{"type", "order"},
		{"orders", orders},
		{"grouping", grouping},
	}

	resp, err := ws.postAction(action)
	if err != nil {
		return err
	}

	var inner HLExchangeInner
	if err := json.Unmarshal(resp.Response, &inner); err != nil {
		return fmt.Errorf("placeTpSlOrdersWebsocket(%s): parse response: %w", coin, err)
	}

	var data HLOrderData
	if err := json.Unmarshal(inner.Data, &data); err != nil {
		return fmt.Errorf("placeTpSlOrdersWebsocket(%s): parse order data: %w", coin, err)
	}

	for _, s := range data.Statuses {
		if s.Error != nil {
			return &internal.HyperliquidExchangeError{Message: *s.Error}
		}
	}
	return nil
}

func (h *HyperliquidExchange) InvalidateCache() {
	h.cacheMu.Lock()
	h.symbolCache = make(map[string]hlSymbolCache)
	h.perpDexCache = make(map[string]int)
	h.cacheMu.Unlock()
}

// splitDexCoin splits a coin into its perp dex name and the coin itself. A coin
// on a builder-deployed dex is namespaced as "<dex>:<coin>" (e.g. "xyz:SPCX"),
// in which case the dex name is returned; otherwise the dex is empty (the
// default Hyperliquid perp dex). The coin is returned unchanged either way, as
// the dex universe and clearinghouseState responses use the namespaced name.
func splitDexCoin(coin string) (dex, name string) {
	if i := strings.IndexByte(coin, ':'); i >= 0 {
		return coin[:i], coin
	}
	return "", coin
}

// hlWireAssetID returns the asset id to send in the "a"/"asset" field of an
// exchange action. The default perp dex (perpDexIndex 0) uses the plain
// universe index; builder-deployed dexs are offset per the Hyperliquid spec:
// 100000 + perp_dex_index*10000 + index_in_meta.
func hlWireAssetID(perpDexIndex, indexInMeta int) int {
	if perpDexIndex == 0 {
		return indexInMeta
	}
	return 100000 + perpDexIndex*10000 + indexInMeta
}

func hlBaseURL(account subsettings.HyperliquidExchangeAccount) string {
	return resolveHyperliquidExchangeURL(account.IsTestnet)
}

// userAddress returns the address to use for info queries.
func (h *HyperliquidExchange) userAddress(account subsettings.HyperliquidExchangeAccount) (string, error) {
	return hlAccountUserAddress(account)
}

// privateKey retrieves the private key for the account from the secret cache.
func (h *HyperliquidExchange) privateKey(account subsettings.HyperliquidExchangeAccount) (string, error) {
	key, err := h.creds.Get(account.PrivateKey)
	if err != nil {
		return "", newExchangeError(ErrApiKeyNotSet, fmt.Errorf("private key not set: %w", err))
	}
	return key, nil
}

// -------------------------------------------------------------------
// Low-level HTTP helpers
// -------------------------------------------------------------------

// infoRequest POSTs to /info and unmarshals the response into result.
func (h *HyperliquidExchange) infoRequest(
	account subsettings.HyperliquidExchangeAccount,
	params map[string]interface{},
	result interface{},
) error {
	addr, err := h.userAddress(account)
	if err != nil {
		return err
	}

	h.semaphore <- struct{}{}
	defer func() { <-h.semaphore }()

	url := hlBaseURL(account) + "/info"
	body, err := h.post(url, addr, params)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("infoRequest: unmarshal: %w (body: %s)", err, string(body))
	}
	return nil
}

// exchangeAction signs and POSTs an action to /exchange.
func (h *HyperliquidExchange) exchangeAction(
	account subsettings.HyperliquidExchangeAccount,
	action hlOrderedMap,
) (*HLExchangeResponse, error) {
	privKey, err := h.privateKey(account)
	if err != nil {
		return nil, err
	}

	nonce, err := h.nextActionNonce(account)
	if err != nil {
		return nil, err
	}
	vault := account.SubaccountAddress
	isMainnet := !account.IsTestnet

	sig, err := hlSignL1Action(privKey, action, vault, nonce, isMainnet)
	if err != nil {
		return nil, fmt.Errorf("exchangeAction: sign: %w", err)
	}

	type requestBody struct {
		Action       hlOrderedMap `json:"action"`
		Nonce        int64        `json:"nonce"`
		Signature    HLSignature  `json:"signature"`
		VaultAddress *string      `json:"vaultAddress,omitempty"`
	}

	reqBody := requestBody{
		Action:       action,
		Nonce:        nonce,
		Signature:    *sig,
		VaultAddress: vault,
	}

	addr, err := h.userAddress(account)
	if err != nil {
		return nil, err
	}

	h.semaphore <- struct{}{}
	defer func() { <-h.semaphore }()

	url := hlBaseURL(account) + "/exchange"
	body, err := h.post(url, addr, reqBody)
	if err != nil {
		return nil, err
	}

	var resp HLExchangeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("exchangeAction: unmarshal: %w (body: %s)", err, string(body))
	}

	if msg, isErr := resp.ErrorMessage(); isErr {
		return nil, &internal.HyperliquidExchangeError{Message: msg}
	}

	return &resp, nil
}

// post sends a POST request and returns the raw response body.
func (h *HyperliquidExchange) post(url, rateLimitKey string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("post: marshal: %w", err)
	}

	maxRetries := h.cfg.HttpClient.RateLimiter.MaxRetries
	defaultRetry := h.cfg.HttpClient.RateLimiter.DefaultRetry

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		var shouldRetry *subsettings.RetrySettings
		var retryWarning string

		if err := h.rateLimiter.TryAcquire(rateLimitKey); err != nil {
			lastErr = err
			shouldRetry = &defaultRetry
			retryWarning = fmt.Sprintf("Hit rate limit on %s", rateLimitKey)
		} else {
			if h.logApiResponses {
				h.logger.Debug("Hyperliquid Exchange Request", "url", url, "body", string(jsonData))
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
			if err != nil {
				return nil, fmt.Errorf("post: create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := h.client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("post: request failed: %w", err)
			}

			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("post: read body: %w", readErr)
			}

			if h.logApiResponses {
				h.logger.Debug("Hyperliquid Exchange Response", "url", url, "status", resp.StatusCode, "body", string(body))
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return body, nil
			}

			statusErr := &internal.HyperliquidBadStatusError{Status: resp.StatusCode}
			lastErr = statusErr

			if retrySettings, ok := h.cfg.HttpClient.RateLimiter.RetryStatuscode[resp.StatusCode]; ok {
				shouldRetry = &retrySettings
				switch resp.StatusCode {
				case 418:
					retryWarning = fmt.Sprintf("IP has been auto-banned %d from %s", resp.StatusCode, url)
				case 429:
					retryWarning = fmt.Sprintf("Hit rate limit %d from %s", resp.StatusCode, url)
				default:
					retryWarning = fmt.Sprintf("Unexpected retry status code from %s", url)
				}
			} else {
				if h.logApiResponses {
					h.logger.Error("Hyperliquid Exchange bad HTTP status",
						"status", resp.StatusCode,
						"response", string(body),
						"url", url)
				}
				return nil, statusErr
			}
		}

		if shouldRetry != nil && attempt < maxRetries-1 {
			sleepTime := math.Min(
				shouldRetry.InitialRetryDelay*math.Pow(shouldRetry.RetryDelayFactor, float64(attempt)),
				shouldRetry.MaxRetryDelay,
			)
			jitter := 1.0 + rand.Float64()*shouldRetry.RetryDelayJitter
			sleepTime *= jitter

			h.logger.Warn("Rate limit hit, retrying",
				"warning", retryWarning,
				"retry_in_seconds", sleepTime)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &internal.LimiterException{
		Message: fmt.Sprintf("Max retries (%d) reached for %s", maxRetries, url),
	}
}

// -------------------------------------------------------------------
// Universe / symbol info
// -------------------------------------------------------------------

// fetchMeta fetches the metaAndAssetCtxs info response (always fresh for mark prices).
// An empty dex targets the default Hyperliquid perp dex; otherwise the universe
// and asset contexts for the named builder-deployed dex are returned.
func (h *HyperliquidExchange) fetchMetaAndAssetCtxs(account subsettings.HyperliquidExchangeAccount, dex string) (*HLMetaAndAssetCtxsResponse, error) {
	params := map[string]interface{}{"type": "metaAndAssetCtxs"}
	if dex != "" {
		params["dex"] = dex
	}
	var resp HLMetaAndAssetCtxsResponse
	if err := h.infoRequest(account, params, &resp); err != nil {
		return nil, fmt.Errorf("metaAndAssetCtxs: %w", err)
	}
	return &resp, nil
}

// perpDexIndex returns the perp_dex_index (position in the perpDexs array) for a
// dex name. The default dex (empty name) is index 0. Results are cached.
func (h *HyperliquidExchange) perpDexIndex(account subsettings.HyperliquidExchangeAccount, dex string) (int, error) {
	if dex == "" {
		return 0, nil
	}

	h.cacheMu.RLock()
	idx, ok := h.perpDexCache[dex]
	h.cacheMu.RUnlock()
	if ok {
		return idx, nil
	}

	var dexs []*HLPerpDex
	if err := h.infoRequest(account, map[string]interface{}{"type": "perpDexs"}, &dexs); err != nil {
		return 0, fmt.Errorf("perpDexs: %w", err)
	}

	found := -1
	h.cacheMu.Lock()
	for i, d := range dexs {
		if d != nil && d.Name != "" {
			h.perpDexCache[d.Name] = i
			if d.Name == dex {
				found = i
			}
		}
	}
	h.cacheMu.Unlock()

	if found < 0 {
		return 0, &internal.HyperliquidExchangeError{Message: fmt.Sprintf("perp dex not found: %s", dex)}
	}
	return found, nil
}

// assetIndex returns the wire asset id and universe entry for a coin name. The
// coin may be namespaced for a builder-deployed dex (e.g. "xyz:SPCX"), in which
// case that dex's universe is fetched and the asset id is offset accordingly.
// Results are cached; use InvalidateCache() to force a refresh.
func (h *HyperliquidExchange) assetIndex(account subsettings.HyperliquidExchangeAccount, coin string) (int, *HLUniverseEntry, error) {
	h.cacheMu.RLock()
	cached, ok := h.symbolCache[coin]
	h.cacheMu.RUnlock()
	if ok {
		return cached.Index, cached.Entry, nil
	}

	dex, _ := splitDexCoin(coin)
	perpDexIdx, err := h.perpDexIndex(account, dex)
	if err != nil {
		return 0, nil, err
	}

	resp, err := h.fetchMetaAndAssetCtxs(account, dex)
	if err != nil {
		return 0, nil, err
	}

	h.cacheMu.Lock()
	for i := range resp.Meta.Universe {
		e := &resp.Meta.Universe[i]
		h.symbolCache[e.Name] = hlSymbolCache{Entry: e, Index: hlWireAssetID(perpDexIdx, i)}
	}
	h.cacheMu.Unlock()

	h.cacheMu.RLock()
	cached, ok = h.symbolCache[coin]
	h.cacheMu.RUnlock()
	if !ok {
		return 0, nil, &internal.HyperliquidExchangeError{Message: fmt.Sprintf("coin not found in universe: %s", coin)}
	}
	return cached.Index, cached.Entry, nil
}

// -------------------------------------------------------------------
// Positions
// -------------------------------------------------------------------

func (h *HyperliquidExchange) fetchClearinghouseState(
	account subsettings.HyperliquidExchangeAccount,
	dex string,
) (HLClearinghouseState, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return HLClearinghouseState{}, err
	}
	params := map[string]interface{}{
		"type": "clearinghouseState",
		"user": addr,
	}
	if dex != "" {
		params["dex"] = dex
	}
	var state HLClearinghouseState
	if err := h.infoRequest(account, params, &state); err != nil {
		return HLClearinghouseState{}, err
	}
	return state, nil
}

func buildCoinToCtxMap(metaCtxs *HLMetaAndAssetCtxsResponse) map[string]*HLAssetCtx {
	coinToCtx := make(map[string]*HLAssetCtx, len(metaCtxs.Meta.Universe))
	for i, u := range metaCtxs.Meta.Universe {
		if i < len(metaCtxs.AssetCtxs) {
			coinToCtx[u.Name] = &metaCtxs.AssetCtxs[i]
		}
	}
	return coinToCtx
}

func appendHLPositions(
	state HLClearinghouseState,
	coinToCtx map[string]*HLAssetCtx,
	positions []HLAssetPosition,
	ctxs []HLAssetCtx,
) ([]HLAssetPosition, []HLAssetCtx) {
	for _, w := range state.AssetPositions {
		pos := w.Position
		if pos.Szi.Float64() == 0 {
			continue
		}
		positions = append(positions, pos)
		if ctx, ok := coinToCtx[pos.Coin]; ok {
			ctxs = append(ctxs, *ctx)
		} else {
			ctxs = append(ctxs, HLAssetCtx{})
		}
	}
	return positions, ctxs
}

// Positions returns all open positions for the account, with mark prices populated.
// Includes positions on every supported perp dex; the returned state is the default dex one.
func (h *HyperliquidExchange) Positions(account subsettings.HyperliquidExchangeAccount) ([]HLAssetPosition, []HLAssetCtx, *HLClearinghouseState, error) {
	positions := make([]HLAssetPosition, 0)
	ctxs := make([]HLAssetCtx, 0)
	var defaultState HLClearinghouseState
	for _, dex := range HyperliquidSupportedDexes {
		state, err := h.fetchClearinghouseState(account, dex)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("positions: clearinghouseState dex=%q: %w", dex, err)
		}

		metaCtxs, err := h.fetchMetaAndAssetCtxs(account, dex)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("positions: metaAndAssetCtxs dex=%q: %w", dex, err)
		}

		if dex == "" {
			defaultState = state
		}
		positions, ctxs = appendHLPositions(state, buildCoinToCtxMap(metaCtxs), positions, ctxs)
	}

	return positions, ctxs, &defaultState, nil
}

// Position returns the position for a single coin, or nil if none. The coin may
// be namespaced for a builder-deployed dex (e.g. "xyz:SPCX"), in which case that
// dex's clearinghouse state is queried.
func (h *HyperliquidExchange) Position(account subsettings.HyperliquidExchangeAccount, coin string) (*HLAssetPosition, *HLAssetCtx, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return nil, nil, err
	}

	dex, _ := splitDexCoin(coin)
	params := map[string]interface{}{
		"type": "clearinghouseState",
		"user": addr,
	}
	if dex != "" {
		params["dex"] = dex
	}

	var state HLClearinghouseState
	if err := h.infoRequest(account, params, &state); err != nil {
		return nil, nil, fmt.Errorf("position: clearinghouseState: %w", err)
	}

	for _, w := range state.AssetPositions {
		if w.Position.Coin != coin || w.Position.Szi.Float64() == 0 {
			continue
		}
		pos := w.Position
		ctx, err := h.Ticker(account, coin)
		if err != nil {
			return &pos, nil, nil
		}
		return &pos, ctx, nil
	}
	return nil, nil, nil
}

// ClearinghouseState returns the full clearinghouseState for the account.
func (h *HyperliquidExchange) ClearinghouseState(account subsettings.HyperliquidExchangeAccount) (*HLClearinghouseState, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return nil, err
	}
	var state HLClearinghouseState
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "clearinghouseState",
		"user": addr,
	}, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// UserRateLimit returns the account's current request-rate-limit usage and cap.
func (h *HyperliquidExchange) UserRateLimit(account subsettings.HyperliquidExchangeAccount) (*HLUserRateLimit, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return nil, err
	}
	var rateLimit HLUserRateLimit
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "userRateLimit",
		"user": addr,
	}, &rateLimit); err != nil {
		return nil, err
	}
	return &rateLimit, nil
}

// SpotClearinghouseState returns the spot wallet balances for the account.
func (h *HyperliquidExchange) SpotClearinghouseState(account subsettings.HyperliquidExchangeAccount) (*HLSpotClearinghouseState, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return nil, err
	}
	var state HLSpotClearinghouseState
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "spotClearinghouseState",
		"user": addr,
	}, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// UserAbstraction returns the account abstraction type for the account (e.g. "unifiedAccount", "default").
func (h *HyperliquidExchange) UserAbstraction(account subsettings.HyperliquidExchangeAccount) (string, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return "", err
	}
	var accountType string
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "userAbstraction",
		"user": addr,
	}, &accountType); err != nil {
		return "", err
	}
	return accountType, nil
}

// -------------------------------------------------------------------
// Ticker / symbol info
// -------------------------------------------------------------------

// Ticker returns the current market context (including mark price) for a coin.
func (h *HyperliquidExchange) Ticker(account subsettings.HyperliquidExchangeAccount, coin string) (*HLAssetCtx, error) {
	dex, _ := splitDexCoin(coin)
	resp, err := h.fetchMetaAndAssetCtxs(account, dex)
	if err != nil {
		return nil, err
	}
	for i, u := range resp.Meta.Universe {
		if u.Name == coin {
			if i < len(resp.AssetCtxs) {
				return &resp.AssetCtxs[i], nil
			}
		}
	}
	return nil, &internal.HyperliquidExchangeError{Message: fmt.Sprintf("coin not found: %s", coin)}
}

// SymbolInfo returns the universe metadata for a coin (lot size, max leverage, etc.).
// Results are cached; pass cached=false or call InvalidateCache() to force a refresh.
func (h *HyperliquidExchange) SymbolInfo(account subsettings.HyperliquidExchangeAccount, coin string, cached bool) (*HLUniverseEntry, error) {
	if cached {
		h.cacheMu.RLock()
		c, ok := h.symbolCache[coin]
		h.cacheMu.RUnlock()
		if ok {
			return c.Entry, nil
		}
	}

	_, entry, err := h.assetIndex(account, coin)
	return entry, err
}

// -------------------------------------------------------------------
// Orders
// -------------------------------------------------------------------

// PlaceMarketOrder places a market order as an IOC limit order with 5% slippage.
// If takeProfit or stopLoss are non-nil and non-zero, separate trigger orders are placed.
func (h *HyperliquidExchange) PlaceMarketOrder(
	account subsettings.HyperliquidExchangeAccount,
	coin, side, size string,
	takeProfit, stopLoss *float64,
	reduceOnly, _ bool,
) (*HLOrderData, error) {
	idx, entry, err := h.assetIndex(account, coin)
	if err != nil {
		return nil, err
	}

	ctx, err := h.Ticker(account, coin)
	if err != nil {
		return nil, err
	}
	markPrice := ctx.MarkPx.Float64()
	if markPrice == 0 {
		return nil, fmt.Errorf("PlaceMarketOrder: mark price is zero for %s", coin)
	}

	isBuy := side == "Buy"
	var limitPx float64
	if isBuy {
		limitPx = markPrice * (1 + hlMarketOrderSlippage)
	} else {
		limitPx = markPrice * (1 - hlMarketOrderSlippage)
	}

	sz, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return nil, fmt.Errorf("PlaceMarketOrder: invalid size %q: %w", size, err)
	}

	orderMap := hlOrderedMap{
		{"a", idx},
		{"b", isBuy},
		{"p", hlPriceToWire(limitPx, entry.SzDecimals)},
		{"s", hlFloatToWire(sz)},
		{"r", reduceOnly},
		{"t", hlOrderedMap{{"limit", hlOrderedMap{{"tif", "Ioc"}}}}},
	}

	action := hlOrderedMap{
		{"type", "order"},
		{"orders", []interface{}{orderMap}},
		{"grouping", "na"},
	}

	resp, err := h.exchangeAction(account, action)
	if err != nil {
		return nil, err
	}

	var inner HLExchangeInner
	if err := json.Unmarshal(resp.Response, &inner); err != nil {
		return nil, fmt.Errorf("PlaceMarketOrder: parse response: %w", err)
	}

	var data HLOrderData
	if err := json.Unmarshal(inner.Data, &data); err != nil {
		return nil, fmt.Errorf("PlaceMarketOrder: parse order data: %w", err)
	}

	// Check per-order errors
	for _, s := range data.Statuses {
		if s.Error != nil {
			return nil, &internal.HyperliquidExchangeError{Message: *s.Error}
		}
	}

	// Place TP/SL trigger orders if requested
	if (takeProfit != nil && *takeProfit > 0) || (stopLoss != nil && *stopLoss > 0) {
		if err := h.placeTpSlOrders(account, idx, entry.SzDecimals, coin, isBuy, sz, takeProfit, stopLoss, "na"); err != nil {
			h.logger.Warn("PlaceMarketOrder: TP/SL placement failed", "coin", coin, "err", err)
		}
	}

	return &data, nil
}

// SetLeverage updates the leverage for a coin (isolated margin).
func (h *HyperliquidExchange) SetLeverage(account subsettings.HyperliquidExchangeAccount, coin string, leverage int) error {
	idx, _, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}
	_, err = h.exchangeAction(account, hlUpdateLeverageAction(idx, leverage))
	return err
}

// SetLeverageWebsocket updates leverage via the websocket connection.
func (h *HyperliquidExchange) SetLeverageWebsocket(accountKey, coin string, leverage int) error {
	account, ok := h.cfg.Accounts[accountKey]
	if !ok {
		return fmt.Errorf("SetLeverageWebsocket: account not found: %s", accountKey)
	}
	ws, ok := h.websocketClients[accountKey]
	if !ok || ws == nil {
		return fmt.Errorf("SetLeverageWebsocket: websocket client not found: %s", accountKey)
	}
	idx, _, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}
	_, err = ws.postAction(hlUpdateLeverageAction(idx, leverage))
	return err
}

// hlUpdateLeverageAction builds an updateLeverage action for isolated margin.
func hlUpdateLeverageAction(assetIdx int, leverage int) hlOrderedMap {
	return hlOrderedMap{
		{"type", "updateLeverage"},
		{"asset", assetIdx},
		{"isCross", false},
		{"leverage", leverage},
	}
}

// SetPositionTpsl places TP and SL trigger orders for an existing position.
// A nil takeProfit or stopLoss leg is not placed. Existing TP/SL orders are
// NOT replaced (the positionTpsl grouping only ties orders to the position):
// callers must cancel them via CancelOrders or the triggers accumulate.
func (h *HyperliquidExchange) SetPositionTpsl(
	account subsettings.HyperliquidExchangeAccount,
	coin, positionSide string,
	takeProfit, stopLoss *float64,
) error {
	idx, entry, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}

	// Fetch current position size
	pos, _, err := h.Position(account, coin)
	if err != nil {
		return fmt.Errorf("SetPositionTpsl: fetch position: %w", err)
	}
	sz := 0.0
	if pos != nil {
		sz = math.Abs(pos.Szi.Float64())
	}

	isBuy := positionSide == "Buy"

	return h.placeTpSlOrders(account, idx, entry.SzDecimals, coin, isBuy, sz, takeProfit, stopLoss, "positionTpsl")
}

// placeTpSlOrders places TP and SL trigger orders in a single exchange action.
// isBuy is the direction of the POSITION (not the trigger order direction).
func (h *HyperliquidExchange) placeTpSlOrders(
	account subsettings.HyperliquidExchangeAccount,
	assetIdx int,
	szDecimals int,
	coin string,
	positionIsBuy bool,
	sz float64,
	takeProfit, stopLoss *float64,
	grouping string,
) error {
	// Trigger orders close the position, so they go in the opposite direction
	triggerIsBuy := !positionIsBuy

	var orders []interface{}

	if takeProfit != nil && *takeProfit > 0 {
		orders = append(orders, hlOrderedMap{
			{"a", assetIdx},
			{"b", triggerIsBuy},
			{"p", hlPriceToWire(*takeProfit, szDecimals)},
			{"s", hlFloatToWire(sz)},
			{"r", true},
			{"t", hlOrderedMap{{"trigger", hlOrderedMap{
				{"isMarket", true},
				{"triggerPx", hlPriceToWire(*takeProfit, szDecimals)},
				{"tpsl", "tp"},
			}}}},
		})
	}

	if stopLoss != nil && *stopLoss > 0 {
		orders = append(orders, hlOrderedMap{
			{"a", assetIdx},
			{"b", triggerIsBuy},
			{"p", hlPriceToWire(*stopLoss, szDecimals)},
			{"s", hlFloatToWire(sz)},
			{"r", true},
			{"t", hlOrderedMap{{"trigger", hlOrderedMap{
				{"isMarket", true},
				{"triggerPx", hlPriceToWire(*stopLoss, szDecimals)},
				{"tpsl", "sl"},
			}}}},
		})
	}

	if len(orders) == 0 {
		return nil
	}

	action := hlOrderedMap{
		{"type", "order"},
		{"orders", orders},
		{"grouping", grouping},
	}

	resp, err := h.exchangeAction(account, action)
	if err != nil {
		return err
	}

	var inner HLExchangeInner
	if err := json.Unmarshal(resp.Response, &inner); err != nil {
		return fmt.Errorf("placeTpSlOrders(%s): parse response: %w", coin, err)
	}

	var data HLOrderData
	if err := json.Unmarshal(inner.Data, &data); err != nil {
		return fmt.Errorf("placeTpSlOrders(%s): parse order data: %w", coin, err)
	}

	for _, s := range data.Statuses {
		if s.Error != nil {
			return &internal.HyperliquidExchangeError{Message: *s.Error}
		}
	}
	return nil
}

// CancelOrders cancels open orders on a coin by order ID. Per-order failures
// (e.g. an order that already triggered or was canceled) are reported inside
// the response statuses and are deliberately ignored; only request-level
// errors are returned.
func (h *HyperliquidExchange) CancelOrders(account subsettings.HyperliquidExchangeAccount, coin string, oids []int64) error {
	if len(oids) == 0 {
		return nil
	}
	idx, _, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}
	_, err = h.exchangeAction(account, hlCancelAction(idx, oids))
	return err
}

// CancelOrdersWebsocket cancels open orders on a coin by order ID via the
// websocket connection. Per-order failures are ignored like in CancelOrders.
func (h *HyperliquidExchange) CancelOrdersWebsocket(accountKey, coin string, oids []int64) error {
	if len(oids) == 0 {
		return nil
	}
	account, ok := h.cfg.Accounts[accountKey]
	if !ok {
		return fmt.Errorf("CancelOrdersWebsocket: account not found: %s", accountKey)
	}
	ws, ok := h.websocketClients[accountKey]
	if !ok || ws == nil {
		return fmt.Errorf("CancelOrdersWebsocket: websocket client not found: %s", accountKey)
	}
	idx, _, err := h.assetIndex(account, coin)
	if err != nil {
		return err
	}
	_, err = ws.postAction(hlCancelAction(idx, oids))
	return err
}

// hlCancelAction builds a cancel action for the given asset and order IDs.
func hlCancelAction(assetIdx int, oids []int64) hlOrderedMap {
	cancels := make([]interface{}, len(oids))
	for i, oid := range oids {
		cancels[i] = hlOrderedMap{{"a", assetIdx}, {"o", oid}}
	}
	return hlOrderedMap{
		{"type", "cancel"},
		{"cancels", cancels},
	}
}

// OpenStopOrders returns all open TP and SL trigger orders for the account.
func (h *HyperliquidExchange) OpenStopOrders(account subsettings.HyperliquidExchangeAccount) ([]HLOpenOrder, error) {
	addr, err := h.userAddress(account)
	if err != nil {
		return nil, err
	}

	// frontendOpenOrders is scoped per perp dex; need to query all supported ones.
	var allOrders []HLOpenOrder
	for _, dex := range HyperliquidSupportedDexes {
		params := map[string]interface{}{
			"type": "frontendOpenOrders",
			"user": addr,
		}
		if dex != "" {
			params["dex"] = dex
		}
		var orders []HLOpenOrder
		if err := h.infoRequest(account, params, &orders); err != nil {
			return nil, err
		}
		allOrders = append(allOrders, orders...)
	}

	// Filter to only TP/SL trigger orders
	var stopOrders []HLOpenOrder
	for _, o := range allOrders {
		if kind := hlTpslKind(o); o.IsTrigger && (kind == "TP" || kind == "SL") {
			stopOrders = append(stopOrders, o)
		}
	}
	return stopOrders, nil
}

// CancelStopOrders cancels open TP/SL trigger orders by order id.
func (h *HyperliquidExchange) CancelStopOrders(
	account subsettings.HyperliquidExchangeAccount,
	orders []OpenStopOrder,
) error {
	if len(orders) == 0 {
		return nil
	}

	cancels := make([]interface{}, 0, len(orders))
	for _, order := range orders {
		if order.OrderID == nil {
			continue
		}
		oid, err := strconv.ParseInt(*order.OrderID, 10, 64)
		if err != nil {
			return fmt.Errorf("CancelStopOrders: parse order id %q: %w", *order.OrderID, err)
		}
		assetIdx, _, err := h.assetIndex(account, order.Symbol)
		if err != nil {
			return err
		}
		cancels = append(cancels, hlOrderedMap{{"a", assetIdx}, {"o", oid}})
	}

	if len(cancels) == 0 {
		return nil
	}

	action := hlOrderedMap{
		{"type", "cancel"},
		{"cancels", cancels},
	}

	resp, err := h.exchangeAction(account, action)
	if err != nil {
		return err
	}

	var inner HLExchangeInner
	if err := json.Unmarshal(resp.Response, &inner); err != nil {
		return fmt.Errorf("CancelStopOrders: parse response: %w", err)
	}

	var data HLOrderData
	if err := json.Unmarshal(inner.Data, &data); err != nil {
		return fmt.Errorf("CancelStopOrders: parse cancel data: %w", err)
	}

	for _, status := range data.Statuses {
		if status.Error == nil || *status.Error == hlMissingOrderError {
			// ignore missing orders
			continue
		}
		return &internal.HyperliquidExchangeError{Message: *status.Error}
	}
	return nil
}

// ApiKeyInfo returns the ExtraAgent entry for the account's private key by deriving its address
// and matching it against the master wallet's authorized agents.
func (h *HyperliquidExchange) ApiKeyInfo(account subsettings.HyperliquidExchangeAccount) (*HLExtraAgent, error) {
	privKey, err := h.privateKey(account)
	if err != nil {
		return nil, err
	}
	derivedAddr, err := DeriveHLAddress(privKey)
	if err != nil {
		return nil, fmt.Errorf("ApiKeyInfo: derive address: %w", err)
	}
	agents, err := h.ExtraAgents(account)
	if err != nil {
		return nil, fmt.Errorf("ApiKeyInfo: fetch extra agents: %w", err)
	}
	for i := range agents {
		if strings.EqualFold(agents[i].Address, derivedAddr) {
			return &agents[i], nil
		}
	}
	return nil, fmt.Errorf("ApiKeyInfo: derived address %s not found in extra agents", derivedAddr)
}

// ExtraAgents returns the extra agent wallets (API wallets) authorized for the master wallet address.
func (h *HyperliquidExchange) ExtraAgents(account subsettings.HyperliquidExchangeAccount) ([]HLExtraAgent, error) {
	if account.MasterWalletAddress == "" {
		return nil, fmt.Errorf("ExtraAgents: master_wallet_address is not set")
	}
	var agents []HLExtraAgent
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "extraAgents",
		"user": account.MasterWalletAddress,
	}, &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

// Subaccounts returns the subaccounts for the master wallet address.
func (h *HyperliquidExchange) Subaccounts(account subsettings.HyperliquidExchangeAccount) ([]HLSubaccount, error) {
	if account.MasterWalletAddress == "" {
		return nil, fmt.Errorf("Subaccounts: master_wallet_address is not set")
	}
	var subaccounts []HLSubaccount
	if err := h.infoRequest(account, map[string]interface{}{
		"type": "subAccounts",
		"user": account.MasterWalletAddress,
	}, &subaccounts); err != nil {
		return nil, err
	}
	return subaccounts, nil
}
