package clients

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getmiroir/trading-service/internal/utils"
)

// HLClearinghouseState is the response from the clearinghouseState info request.
type HLClearinghouseState struct {
	AssetPositions             []HLAssetPositionWrapper `json:"assetPositions"`
	MarginSummary              HLMarginSummary          `json:"marginSummary"`
	CrossMarginSummary         HLMarginSummary          `json:"crossMarginSummary"`
	CrossMaintenanceMarginUsed utils.Float64String      `json:"crossMaintenanceMarginUsed"`
}

type HLAssetPositionWrapper struct {
	Position HLAssetPosition `json:"position"`
}

type HLAssetPosition struct {
	Coin          string              `json:"coin"`
	EntryPx       utils.Float64String `json:"entryPx"`
	Leverage      HLLeverage          `json:"leverage"`
	LiquidationPx utils.Float64String `json:"liquidationPx"`
	MarginUsed    utils.Float64String `json:"marginUsed"`
	PositionValue utils.Float64String `json:"positionValue"`
	Szi           utils.Float64String `json:"szi"` // positive = long, negative = short
	UnrealizedPnl utils.Float64String `json:"unrealizedPnl"`
}

type HLLeverage struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type HLMarginSummary struct {
	AccountValue    utils.Float64String `json:"accountValue"`
	TotalMarginUsed utils.Float64String `json:"totalMarginUsed"`
	TotalNtlPos     utils.Float64String `json:"totalNtlPos"`
	TotalRawUsd     utils.Float64String `json:"totalRawUsd"`
}

// HLSpotBalance is a single token entry from the spotClearinghouseState response.
type HLSpotBalance struct {
	Coin     string              `json:"coin"`
	Hold     utils.Float64String `json:"hold"`
	Total    utils.Float64String `json:"total"`
	EntryNtl utils.Float64String `json:"entryNtl"`
}

// HLSpotClearinghouseState is the response from the spotClearinghouseState info request.
type HLSpotClearinghouseState struct {
	Balances []HLSpotBalance `json:"balances"`
}

// HLUniverseMeta is the universe section of the meta info response.
type HLUniverseMeta struct {
	Universe []HLUniverseEntry `json:"universe"`
}

// HLUniverseEntry holds per-asset metadata from the meta info request.
type HLUniverseEntry struct {
	Name         string `json:"name"`
	SzDecimals   int    `json:"szDecimals"`
	MaxLeverage  int    `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated"`
	GrowthMode   string `json:"growthMode"`
}

type HLUserRateLimit struct {
	CumVlm        utils.Float64String `json:"cumVlm"`
	NRequestsUsed int64               `json:"nRequestsUsed"`
	NRequestsCap  int64               `json:"nRequestsCap"`
}

// HLAssetCtx holds per-asset market context from the metaAndAssetCtxs info request.
type HLAssetCtx struct {
	DayNtlVlm    utils.Float64String `json:"dayNtlVlm"`
	Funding      utils.Float64String `json:"funding"`
	MarkPx       utils.Float64String `json:"markPx"`
	MidPx        utils.Float64String `json:"midPx"`
	OpenInterest utils.Float64String `json:"openInterest"`
	OraclePx     utils.Float64String `json:"oraclePx"`
	PrevDayPx    utils.Float64String `json:"prevDayPx"`
}

// HLMetaAndAssetCtxsResponse is the response from the metaAndAssetCtxs info request.
// The API returns a 2-element JSON array: [meta, assetCtxs].
type HLMetaAndAssetCtxsResponse struct {
	Meta      HLUniverseMeta
	AssetCtxs []HLAssetCtx
}

func (r *HLMetaAndAssetCtxsResponse) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) != 2 {
		return fmt.Errorf("hlMetaAndAssetCtxs: expected 2-element array, got %d", len(raw))
	}
	if err := json.Unmarshal(raw[0], &r.Meta); err != nil {
		return fmt.Errorf("hlMetaAndAssetCtxs: meta: %w", err)
	}
	return json.Unmarshal(raw[1], &r.AssetCtxs)
}

// HLPerpDex is a single entry from the perpDexs info request. The array's first
// element is null (the default perp dex); its array position is the perp_dex_index.
type HLPerpDex struct {
	Name string `json:"name"`
}

// HLExtraAgent is a single entry from the extraAgents info request.
type HLExtraAgent struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	ValidUntil int64  `json:"validUntil"`
}

// HLSubaccount is a single entry from the subaccounts info request.
type HLSubaccount struct {
	Name           string `json:"name"`
	SubaccountUser string `json:"subaccountUser"`
	Master         string `json:"master"`
}

// HLOpenOrder is a single open order from the frontendOpenOrders info request.
// The plain openOrders request returns only coin/limitPx/oid/side/sz/timestamp;
// the trigger-related fields below are exclusive to frontendOpenOrders.
type HLOpenOrder struct {
	Coin             string              `json:"coin"`
	IsPositionTpsl   bool                `json:"isPositionTpsl"`
	IsTrigger        bool                `json:"isTrigger"`
	LimitPx          utils.Float64String `json:"limitPx"`
	Oid              int64               `json:"oid"`
	OrderType        string              `json:"orderType"` // e.g. "Limit", "Take Profit Market", "Stop Market"
	OrigSz           utils.Float64String `json:"origSz"`
	ReduceOnly       bool                `json:"reduceOnly"`
	Side             string              `json:"side"` // "A" = ask/sell, "B" = bid/buy
	Sz               utils.Float64String `json:"sz"`
	TriggerCondition string              `json:"triggerCondition"` // human-readable, e.g. "N/A" or "Price above 30000"
	TriggerPx        utils.Float64String `json:"triggerPx"`
}

// hlTpslKind classifies an open order as "TP", "SL", or "" (not a TP/SL order)
// from its orderType ("Take Profit Market", "Stop Limit", ...). The
// triggerCondition field is a human-readable description, not a tp/sl marker.
func hlTpslKind(o HLOpenOrder) string {
	switch {
	case strings.Contains(o.OrderType, "Take Profit"):
		return "TP"
	case strings.Contains(o.OrderType, "Stop"):
		return "SL"
	}
	return ""
}

// HLExchangeResponse is the top-level response from the /exchange endpoint.
// On success: {"status":"ok","response":{"type":"...","data":{...}}}
// On error:   {"status":"err","response":"error message"}
type HLExchangeResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

// ErrorMessage returns the error string if Status == "err", else "".
func (r *HLExchangeResponse) ErrorMessage() (string, bool) {
	if r.Status != "err" {
		return "", false
	}
	var msg string
	if err := json.Unmarshal(r.Response, &msg); err != nil {
		return string(r.Response), true
	}
	return msg, true
}

// HLExchangeInner is the structured response body when Status == "ok".
type HLExchangeInner struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// HLOrderData is the data field for a successful order response.
type HLOrderData struct {
	Statuses []HLOrderStatus `json:"statuses"`
}

type HLOrderStatus struct {
	Resting *HLRestingOrder `json:"resting,omitempty"`
	Filled  *HLFilledOrder  `json:"filled,omitempty"`
	Error   *string         `json:"error,omitempty"`
}

// UnmarshalJSON handles both object form {"resting":...} and plain string statuses.
// Known non-error strings: "success" (filled/resting), "waitingForTrigger" (trigger order placed).
func (s *HLOrderStatus) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var msg string
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if msg != "success" && msg != "waitingForTrigger" {
			s.Error = &msg
		}
		return nil
	}
	type alias HLOrderStatus
	return json.Unmarshal(data, (*alias)(s))
}

type HLRestingOrder struct {
	Oid int64 `json:"oid"`
}

type HLFilledOrder struct {
	Oid     int64  `json:"oid"`
	TotalSz string `json:"totalSz"`
	AvgPx   string `json:"avgPx"`
}
