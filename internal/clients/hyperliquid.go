package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/utils"
)

// Hyperliquid represents the Hyperliquid API client
type Hyperliquid struct {
	client          *http.Client
	url             string
	rateLimitKey    string
	bearerToken     *string
	rateLimiter     *RateLimiter
	httpClient      subsettings.HttpClient
	logApiResponses bool
	logger          *slog.Logger
}

// NewHyperliquid creates a new Hyperliquid client
func NewHyperliquid(cfg subsettings.HyperliquidSource, logApiResponses bool) *Hyperliquid {
	client := &http.Client{
		Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
	}

	limits := make([]RateLimit, len(cfg.HttpClient.RateLimiter.Limits))
	for i, l := range cfg.HttpClient.RateLimiter.Limits {
		limits[i] = RateLimit{Requests: l.Requests, Seconds: l.Seconds}
	}

	rateLimitKey := cfg.URL
	if parsed, err := url.Parse(cfg.URL); err == nil && parsed.Host != "" {
		rateLimitKey = parsed.Host
	}

	return &Hyperliquid{
		client:          client,
		url:             cfg.URL,
		rateLimitKey:    rateLimitKey,
		bearerToken:     cfg.BearerToken,
		rateLimiter:     NewRateLimiter(limits),
		httpClient:      cfg.HttpClient,
		logApiResponses: logApiResponses,
		logger:          logging.NewLogger("hyperliquid-client"),
	}
}

// Positions retrieves positions for a given copy target. An optional dex name can be
// passed to retrieve positions held on a specific DEX clearing house instead of the
// default Hyperliquid one.
func (h *Hyperliquid) Positions(target CopyTarget, dex ...string) (*PositionsResponse, error) {
	if target.Address == nil || *target.Address == "" {
		// Return empty positions response
		return &PositionsResponse{
			AssetPositions:     []AssetPosition{},
			MarginSummary:      MarginSummary{AccountValue: utils.Float64String(0), TotalMarginUsed: utils.Float64String(0)},
			CrossMarginSummary: CrossMarginSummary{AccountValue: utils.Float64String(0), TotalMarginUsed: utils.Float64String(0)},
		}, nil
	}

	params := map[string]interface{}{
		"type": "clearinghouseState",
		"user": *target.Address,
	}
	if len(dex) > 0 && dex[0] != "" {
		params["dex"] = dex[0]
	}

	var result PositionsResponse
	if err := h.request(&result, params); err != nil {
		return nil, err
	}
	return &result, nil
}

// UserAbstraction retrieves the user abstraction mode for a copy target.
func (h *Hyperliquid) UserAbstraction(target CopyTarget) (UserAbstractionMode, error) {
	if target.Address == nil || *target.Address == "" {
		return "", nil
	}

	params := map[string]interface{}{
		"type": "userAbstraction",
		"user": *target.Address,
	}

	var mode UserAbstractionMode
	if err := h.request(&mode, params); err != nil {
		return "", err
	}
	return mode, nil
}

// SpotClearinghouseState retrieves the raw spot token balances for a copy target.
// The response is forwarded as-is to the backend, which performs value extraction.
func (h *Hyperliquid) SpotClearinghouseState(target CopyTarget) (string, error) {
	if target.Address == nil || *target.Address == "" {
		return "", nil
	}

	params := map[string]interface{}{
		"type": "spotClearinghouseState",
		"user": *target.Address,
	}

	var spot json.RawMessage
	if err := h.request(&spot, params); err != nil {
		return "", err
	}
	return string(spot), nil
}

// SpotMetaAndAssetCtxs retrieves the raw global spot metadata and asset contexts.
// The response is forwarded as-is to the backend, which performs value extraction.
func (h *Hyperliquid) SpotMetaAndAssetCtxs() (string, error) {
	params := map[string]interface{}{
		"type": "spotMetaAndAssetCtxs",
	}

	var meta json.RawMessage
	if err := h.request(&meta, params); err != nil {
		return "", err
	}
	return string(meta), nil
}

// Portfolio retrieves the raw portfolio history for a copy target.
// The response is forwarded as-is to the backend, which performs value extraction.
func (h *Hyperliquid) Portfolio(target CopyTarget) (string, error) {
	if target.Address == nil || *target.Address == "" {
		return "", nil
	}

	params := map[string]interface{}{
		"type": "portfolio",
		"user": *target.Address,
	}

	var portfolio json.RawMessage
	if err := h.request(&portfolio, params); err != nil {
		return "", err
	}
	return string(portfolio), nil
}

// request makes a POST request to the Hyperliquid API
func (h *Hyperliquid) request(responseModel interface{}, params map[string]interface{}) error {
	jsonData, err := json.Marshal(params)
	if err != nil {
		if h.logApiResponses {
			h.logger.Error("Failed to marshal request", "error", err, "params", params)
		}
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	body, err := h.postWithRetry(jsonData)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, responseModel); err != nil {
		if h.logApiResponses {
			h.logger.Error("Failed to unmarshal response",
				"error", err,
				"body", string(body),
				"url", h.url)
		}
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if h.logApiResponses {
		h.logger.Debug("Hyperliquid API Request successful", "url", h.url)
	}

	return nil
}

func (h *Hyperliquid) postWithRetry(jsonData []byte) ([]byte, error) {
	maxRetries := h.httpClient.RateLimiter.MaxRetries
	defaultRetry := h.httpClient.RateLimiter.DefaultRetry

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		var shouldRetry *subsettings.RetrySettings
		var retryWarning string

		if err := h.rateLimiter.TryAcquire(h.rateLimitKey); err != nil {
			lastErr = err
			shouldRetry = &defaultRetry
			retryWarning = fmt.Sprintf("Hit rate limit on %s", h.rateLimitKey)
		} else {
			if h.logApiResponses {
				h.logger.Debug("Hyperliquid API Request", "method", "POST", "url", h.url)
				h.logger.Debug("Hyperliquid API Request body", "body", string(jsonData))
			}

			req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewBuffer(jsonData))
			if err != nil {
				if h.logApiResponses {
					h.logger.Error("Failed to create request", "error", err, "url", h.url)
				}
				return nil, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if h.bearerToken != nil {
				req.Header.Set("Authorization", "Bearer "+*h.bearerToken)
			}

			resp, err := h.client.Do(req)
			if err != nil {
				if h.logApiResponses {
					h.logger.Error("Request failed", "error", err, "url", h.url)
				}
				return nil, fmt.Errorf("request failed: %w", err)
			}

			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				if h.logApiResponses {
					h.logger.Error("Failed to read response body", "error", readErr, "url", h.url)
				}
				return nil, fmt.Errorf("failed to read response: %w", readErr)
			}

			if h.logApiResponses {
				h.logger.Debug("Hyperliquid API Response",
					"status", resp.StatusCode,
					"url", h.url,
					"body", string(body))
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return body, nil
			}

			statusErr := &internal.HyperliquidBadStatusError{Status: resp.StatusCode}
			lastErr = statusErr

			if retrySettings, ok := h.httpClient.RateLimiter.RetryStatuscode[resp.StatusCode]; ok {
				shouldRetry = &retrySettings
				switch resp.StatusCode {
				case 418:
					retryWarning = fmt.Sprintf("IP has been auto-banned %d from %s", resp.StatusCode, h.url)
				case 429:
					retryWarning = fmt.Sprintf("Hit rate limit %d from %s", resp.StatusCode, h.url)
				default:
					retryWarning = fmt.Sprintf("Unexpected retry status code from %s", h.url)
				}
			} else {
				if h.logApiResponses {
					h.logger.Error("Bad HTTP status",
						"status", resp.StatusCode,
						"response", string(body))
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
		Message: fmt.Sprintf("Max retries (%d) reached for %s", maxRetries, h.url),
	}
}
