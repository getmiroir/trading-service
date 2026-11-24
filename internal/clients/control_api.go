package clients

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/metrics/metric_types"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"github.com/getmiroir/trading-service/internal/version"
)

const (
	ContentTypeHeader        string = "Content-Type"
	AuthorizationHeader      string = "Authorization"
	ApiKeyIdHeader           string = "X-Api-Key-Id"
	TimestampHeader          string = "X-Timestamp"
	SignatureHeader          string = "X-Signature"
	ClientVersionHeader      string = "X-Client-Version"
	ClientPlatformHeader     string = "X-Client-Platform"
	ClientOsHeader           string = "X-Client-Os"
	EmergencyCloseHeader     string = "X-Emergency-Close"
	UnsupportedVersionHeader string = "X-Unsupported-Version"
)

// HTTPStatusError is returned by do() for non-2xx responses.
type HTTPStatusError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

// VersionControlError indicates a version-related rejection from the server
type VersionControlError struct {
	ImmediateClose bool
}

func (e *VersionControlError) Error() string {
	if e.ImmediateClose {
		return "server requires emergency position close"
	}
	return "client version blocked by server"
}

// ControlApi represents the Control API client
type ControlApi struct {
	client      *http.Client
	url         string
	keyId       string
	privateKey  ed25519.PrivateKey
	bearerToken *string
	accounts    map[string]subsettings.ExchangeAccount
	logger      *slog.Logger

	// archivedAccounts is the archived list from the last successful /control/targets fetch.
	// This is needed to avoid syncing archived accounts while the control API is unreachable.
	archivedMu       sync.Mutex
	archivedAccounts []string
}

// NewControlApi creates a new Control API client.
// The API key must be in the format "key_id.base64url(ed25519_private_key_seed)".
func NewControlApi(cfg subsettings.ControlApi, accounts map[string]subsettings.ExchangeAccount) (*ControlApi, error) {
	apiKey, err := cfg.ApiKey.Value()
	if err != nil {
		return nil, fmt.Errorf("internal authentication token is not set: %w", err)
	}
	parts := strings.SplitN(apiKey, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid API key format: expected key_id.private_key")
	}
	seedBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}
	return &ControlApi{
		client: &http.Client{
			Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		},

		url:        cfg.URL,
		keyId:      parts[0],
		privateKey: ed25519.NewKeyFromSeed(seedBytes),
		accounts:   accounts,
		logger:     logging.NewLogger("control-api"),
	}, nil
}

// NewControlApiWithBearerToken creates a new Control API client using a bearer token for authentication.
func NewControlApiWithBearerToken(cfg subsettings.ControlApi, bearerToken string, accounts map[string]subsettings.ExchangeAccount) *ControlApi {
	return &ControlApi{
		client: &http.Client{
			Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		},
		url:         cfg.URL,
		bearerToken: &bearerToken,
		accounts:    accounts,
		logger:      logging.NewLogger("control-api"),
	}
}

// SymbolMappingEntry mirrors one row of the backend /symbol-mappings/ response.
type SymbolMappingEntry struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	SettleCoin   string `json:"settleCoin"`
	SourceSymbol string `json:"sourceSymbol"`
	TargetSymbol string `json:"targetSymbol"`
	Valid        bool   `json:"valid"`
}

type symbolMappingsResponse struct {
	Mappings []SymbolMappingEntry `json:"mappings"`
}

// GetSymbolMappings retrieves the source→target symbol mapping table from the control API.
func (c *ControlApi) GetSymbolMappings() ([]SymbolMappingEntry, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}
	var resp symbolMappingsResponse

	if _, err := c.do("GET", "/symbol-mappings/?validOnly=true", nil, &resp); err != nil {
		return nil, fmt.Errorf("fetching symbol mappings: %w", err)
	}
	return resp.Mappings, nil
}

// GetMappings retrieves copy mappings from the control API
func (c *ControlApi) GetMappings() ([]CopyMapping, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	targets, status := c.fetchCopyTargets()
	return c.buildMappings(targets, status), nil
}

// CreateApiKeyResponse represents the response from creating an API key
type CreateApiKeyResponse struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// CreateApiKey creates a new API key with the given name.
// Generates an Ed25519 key pair locally, registers the public key with the server,
// and returns a composite "key_id.private_key_seed" suitable for storage.
func (c *ControlApi) CreateApiKey(name string) (*CreateApiKeyResponse, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	reqBody := map[string]string{
		"name":       name,
		"public_key": base64.RawURLEncoding.EncodeToString(pubKey),
	}

	var resp CreateApiKeyResponse
	if _, err := c.do("POST", "/api-keys/create", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("creating API key: %w", err)
	}

	resp.Key = resp.Key + "." + base64.RawURLEncoding.EncodeToString(privKey.Seed())
	return &resp, nil
}

// ExchangeCodeRequest represents the request to exchange a PKCE authorization code
type ExchangeCodeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	KeyName      string `json:"key_name"`
	PublicKey    string `json:"public_key"`
}

// ExchangeCode exchanges a PKCE authorization code for an API key.
// Generates an Ed25519 key pair locally, registers the public key with the server,
// and returns a composite "key_id.private_key_seed" suitable for storage.
// This endpoint is unauthenticated — no API key or bearer token is needed.
func (c *ControlApi) ExchangeCode(code, codeVerifier, keyName string) (*CreateApiKeyResponse, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	reqBody := ExchangeCodeRequest{
		Code:         code,
		CodeVerifier: codeVerifier,
		KeyName:      keyName,
		PublicKey:    base64.RawURLEncoding.EncodeToString(pubKey),
	}

	var resp CreateApiKeyResponse
	if _, err := c.do("POST", "/auth/exchange", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("exchanging code for API key: %w", err)
	}

	resp.Key = resp.Key + "." + base64.RawURLEncoding.EncodeToString(privKey.Seed())
	return &resp, nil
}

// DeleteApiKey revokes an API key by its ID on the backend.
func (c *ControlApi) DeleteApiKey(keyID string) error {
	if c.privateKey == nil && c.bearerToken == nil {
		return fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}
	if _, err := c.do("DELETE", "/api-keys/"+keyID, nil, nil); err != nil {
		return fmt.Errorf("deleting API key: %w", err)
	}
	return nil
}

type accountLabelsResponse struct {
	Labels map[string]string `json:"labels"`
}

// GetAccountLabels retrieves the user's account labels keyed by account ID.
func (c *ControlApi) GetAccountLabels() (map[string]string, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}
	var resp accountLabelsResponse
	if _, err := c.do("GET", "/accounts/labels", nil, &resp); err != nil {
		return nil, fmt.Errorf("fetching account labels: %w", err)
	}
	return resp.Labels, nil
}

// SetAccountLabel sets the label for an account. An empty label clears it.
func (c *ControlApi) SetAccountLabel(account, label string) error {
	if c.privateKey == nil && c.bearerToken == nil {
		return fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}
	reqBody := map[string]string{
		"account": account,
		"label":   label,
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if _, err := c.do("POST", "/accounts/set-label", reqBody, &resp); err != nil {
		return fmt.Errorf("setting account label: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("setting account label: backend rejected the request")
	}
	return nil
}

// NewControlApiUnauthenticated creates a Control API client with no authentication.
// Used for unauthenticated endpoints like /auth/exchange.
func NewControlApiUnauthenticated(cfg subsettings.ControlApi) *ControlApi {
	return &ControlApi{
		client: &http.Client{
			Timeout: time.Duration(cfg.HttpClient.Timeout) * time.Second,
		},
		url:    cfg.URL,
		logger: logging.NewLogger("control-api"),
	}
}

// VerifyApiKeyInfo represents API key metadata from the auth verify endpoint
type VerifyApiKeyInfo struct {
	KeyID     string `json:"keyId"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// VerifyAuthResponse represents the response from the auth verify endpoint
type VerifyAuthResponse struct {
	Valid  bool              `json:"valid"`
	UserID string            `json:"userId"`
	Email  *string           `json:"email"`
	ApiKey *VerifyApiKeyInfo `json:"apiKey,omitempty"`
}

// VerifyAuth verifies the current token or API key against the auth endpoint
func (c *ControlApi) VerifyAuth() (*VerifyAuthResponse, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	var resp VerifyAuthResponse
	if _, err := c.do("GET", "/auth/verify", nil, &resp); err != nil {
		return nil, fmt.Errorf("verifying auth: %w", err)
	}

	return &resp, nil
}

// PushLoopMetrics pushes loop metrics to the control API
func (c *ControlApi) PushLoopMetrics(loopMetrics []metric_types.LoopMetrics) (int, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return 0, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	reqBody := map[string]any{
		"loopMetrics": loopMetrics,
	}

	var resp struct {
		Received int `json:"received"`
	}
	if _, err := c.do("POST", "/control/push-loop-metrics", reqBody, &resp); err != nil {
		return 0, fmt.Errorf("pushing loop metrics: %w", err)
	}

	return resp.Received, nil
}

// ComputeNormalizationUser is the exchange account side of a normalization request.
type ComputeNormalizationUser struct {
	Account      string  `json:"account"`
	CurrentValue float64 `json:"currentValue"`
}

// ComputeNormalizationTarget holds raw Hyperliquid JSON responses as strings.
type ComputeNormalizationTarget struct {
	UserAbstraction        string `json:"userAbstraction"`
	Portfolio              string `json:"portfolio"`
	SpotClearinghouseState string `json:"spotClearinghouseState,omitempty"`
	SpotMetaAndAssetCtxs   string `json:"spotMetaAndAssetCtxs,omitempty"`
}

// ComputeNormalizationRequest is the request body for the compute-normalization endpoint.
type ComputeNormalizationRequest struct {
	User   ComputeNormalizationUser   `json:"user"`
	Target ComputeNormalizationTarget `json:"target"`
}

// ComputeNormalizationResponse is the response from the compute-normalization endpoint.
type ComputeNormalizationResponse struct {
	Normalization float64 `json:"normalization"`
	TargetValue   float64 `json:"targetValue"`
}

// NormalizationApiErrorDetail is the structured error body for compute-normalization failures.
type NormalizationApiErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Window  string `json:"window,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

// NormalizationApiError is returned when the compute-normalization endpoint rejects the request.
type NormalizationApiError struct {
	StatusCode int
	Detail     NormalizationApiErrorDetail
}

func (e *NormalizationApiError) Error() string {
	if e.Detail.Message != "" {
		return e.Detail.Message
	}
	return fmt.Sprintf("compute-normalization failed with status %d", e.StatusCode)
}

// ComputeNormalization computes account normalization and target value via the control API.
func (c *ControlApi) ComputeNormalization(req ComputeNormalizationRequest) (ComputeNormalizationResponse, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return ComputeNormalizationResponse{}, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	var resp ComputeNormalizationResponse
	status, err := c.do("POST", "/control/compute-normalization", req, &resp)
	if err == nil {
		return resp, nil
	}

	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) && status == 422 {
		var wrapped struct {
			Detail json.RawMessage `json:"detail"`
		}
		if json.Unmarshal(httpErr.Body, &wrapped) == nil && len(wrapped.Detail) > 0 {
			var detail NormalizationApiErrorDetail
			if json.Unmarshal(wrapped.Detail, &detail) == nil && detail.Code != "" {
				return ComputeNormalizationResponse{}, &NormalizationApiError{
					StatusCode: status,
					Detail:     detail,
				}
			}
			var detailString string
			if json.Unmarshal(wrapped.Detail, &detailString) == nil && detailString != "" {
				return ComputeNormalizationResponse{}, fmt.Errorf("%s", detailString)
			}
		}
	}

	return ComputeNormalizationResponse{}, fmt.Errorf("computing normalization: %w", err)
}

// CopyServiceReportRequest holds the data for a copy service report submission
type CopyServiceReportRequest struct {
	UserAgent   string
	Message     string
	WithLogs    bool
	ServiceLogs []byte // nil if not provided
	GuiLogs     []byte // nil if not provided
}

// SendReport submits a copy service report to the control API
func (c *ControlApi) SendReport(req CopyServiceReportRequest) (*CopyServiceReportResponse, error) {
	if c.privateKey == nil && c.bearerToken == nil {
		return nil, fmt.Errorf("no authentication method configured (API key or bearer token required)")
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("user_agent", req.UserAgent); err != nil {
		return nil, fmt.Errorf("writing user_agent field: %w", err)
	}
	if err := mw.WriteField("message", req.Message); err != nil {
		return nil, fmt.Errorf("writing message field: %w", err)
	}

	if err := mw.WriteField("with_logs", strconv.FormatBool(req.WithLogs)); err != nil {
		return nil, fmt.Errorf("writing with_logs field: %w", err)
	}

	if req.ServiceLogs != nil {
		fw, err := mw.CreateFormFile("service_logs", "service.log")
		if err != nil {
			return nil, fmt.Errorf("creating service_logs field: %w", err)
		}
		if _, err := fw.Write(req.ServiceLogs); err != nil {
			return nil, fmt.Errorf("writing service_logs: %w", err)
		}
	}
	if req.GuiLogs != nil {
		fw, err := mw.CreateFormFile("gui_logs", "gui.log")
		if err != nil {
			return nil, fmt.Errorf("creating gui_logs field: %w", err)
		}
		if _, err := fw.Write(req.GuiLogs); err != nil {
			return nil, fmt.Errorf("writing gui_logs: %w", err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url+"/control/copy-service-report", &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set(ContentTypeHeader, mw.FormDataContentType())

	httpReq.Header.Set(ClientVersionHeader, version.Version)
	httpReq.Header.Set(ClientPlatformHeader, version.Platform)
	httpReq.Header.Set(ClientOsHeader, runtime.GOOS)

	if c.privateKey != nil {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		message := fmt.Sprintf("%s\n%s\n%s", "POST", "/control/copy-service-report", timestamp)
		sig := ed25519.Sign(c.privateKey, []byte(message))
		httpReq.Header.Set(ApiKeyIdHeader, c.keyId)
		httpReq.Header.Set(TimestampHeader, timestamp)
		httpReq.Header.Set(SignatureHeader, base64.RawURLEncoding.EncodeToString(sig))
	} else if c.bearerToken != nil {
		httpReq.Header.Set(AuthorizationHeader, "Bearer "+*c.bearerToken)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result CopyServiceReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

func (c *ControlApi) fetchCopyTargets() (CopyTargets, MappingStatus) {
	var targets CopyTargets
	if _, err := c.do("POST", "/control/targets", nil, &targets); err != nil {
		c.logger.Error("Error retrieving copy targets", "error", err)

		fallback := DefaultCopyTargets()
		c.archivedMu.Lock()
		fallback.ArchivedAccounts = c.archivedAccounts
		c.archivedMu.Unlock()

		if versionErr, ok := err.(*VersionControlError); ok {
			if versionErr.ImmediateClose {
				c.logger.Error("Emergency close required by server")
				return fallback, MappingStatusEmergencyClose
			}
			c.logger.Error("Client version blocked by server")
			return fallback, MappingStatusVersionBlocked
		}

		return fallback, MappingStatusError
	}

	c.archivedMu.Lock()
	c.archivedAccounts = targets.ArchivedAccounts
	c.archivedMu.Unlock()

	return targets, MappingStatusOK
}

// buildMappings creates copy mappings for all Exchange accounts
func (c *ControlApi) buildMappings(targets CopyTargets, status MappingStatus) []CopyMapping {
	archived := make(map[string]struct{}, len(targets.ArchivedAccounts))
	for _, accountName := range targets.ArchivedAccounts {
		archived[accountName] = struct{}{}
	}

	result := make([]CopyMapping, 0, len(c.accounts))

	for accountName, account := range c.accounts {
		if _, ok := archived[accountName]; ok {
			// Hands-off: do not sync at all (manual positions must stay open).
			c.logger.Info("Exchange account is archived, skipping sync",
				"account", accountName)
			continue
		}

		result = append(result, CopyMapping{
			Name:    accountName,
			Account: account,
			Target:  c.resolveTarget(targets, accountName),
			Status:  status,
		})
	}

	return result
}

// resolveTarget determines the copy target for a specific account
func (c *ControlApi) resolveTarget(targets CopyTargets, accountName string) CopyTarget {
	accountTarget, exists := targets.ByAccount[accountName]

	if !exists {
		c.logger.Info("Exchange account not assigned in control API mappings, will disable",
			"account", accountName)
		return disabledTarget()
	}

	if !targets.CopyEnabled || !accountTarget.Enabled {
		c.logger.Info("Exchange account has copy disabled", "account", accountName)
		return disabledTarget()
	}

	c.logger.Info("Exchange account will follow target",
		"account", accountName,
		"target", accountTarget.AddressShort())
	return accountTarget
}

func disabledTarget() CopyTarget {
	target := DefaultCopyTarget()
	target.Enabled = false
	return target
}

// do executes an HTTP request and decodes the JSON response into v.
// It always returns the HTTP status code alongside any error so that callers
// can inspect specific non-2xx codes (e.g. 426) when needed.
func (c *ControlApi) do(method, path string, body, v any) (int, error) {
	var bodyBytes []byte
	var reqBody io.Reader
	if body != nil {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return 0, fmt.Errorf("encoding request: %w", err)
		}
		bodyBytes = buf.Bytes()
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, c.url+path, reqBody)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set(ContentTypeHeader, "application/json")
	}

	req.Header.Set(ClientVersionHeader, version.Version)
	req.Header.Set(ClientPlatformHeader, version.Platform)
	req.Header.Set(ClientOsHeader, runtime.GOOS)

	if c.privateKey != nil {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		bodyHash := sha256.Sum256(bodyBytes)
		signedPath := path
		if i := strings.IndexByte(signedPath, '?'); i >= 0 {
			signedPath = signedPath[:i]
		}
		message := fmt.Sprintf("%s\n%s\n%s\n%s",
			method, signedPath, timestamp, hex.EncodeToString(bodyHash[:]))
		sig := ed25519.Sign(c.privateKey, []byte(message))
		req.Header.Set(ApiKeyIdHeader, c.keyId)
		req.Header.Set(TimestampHeader, timestamp)
		req.Header.Set(SignatureHeader, base64.RawURLEncoding.EncodeToString(sig))
	} else if c.bearerToken != nil {
		req.Header.Set(AuthorizationHeader, "Bearer "+*c.bearerToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.Header.Get(EmergencyCloseHeader) != "" {
			return resp.StatusCode, &VersionControlError{ImmediateClose: true}
		}
		if resp.Header.Get(UnsupportedVersionHeader) != "" {
			return resp.StatusCode, &VersionControlError{ImmediateClose: false}
		}
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, &HTTPStatusError{StatusCode: resp.StatusCode, Body: body}
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return resp.StatusCode, fmt.Errorf("decoding response: %w", err)
		}
	}

	return resp.StatusCode, nil
}
