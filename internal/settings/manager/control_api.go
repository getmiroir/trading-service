package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/getmiroir/trading-service/internal/clients"
	"github.com/getmiroir/trading-service/internal/keyring"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/version"
)

const CONTROL_API_KEYRING string = "control-api-key"

// ApiKeyStatus represents the result of an API key verification check.
type ApiKeyStatus string

const (
	ApiKeyMissing     ApiKeyStatus = "missing"
	ApiKeyInvalid     ApiKeyStatus = "invalid"
	ApiKeyUnreachable ApiKeyStatus = "unreachable"
	ApiKeyValid       ApiKeyStatus = "valid"
)

// VerifyApiKeyResult holds the outcome of VerifyControlApiKey.
type VerifyApiKeyResult struct {
	Status ApiKeyStatus                `json:"status"`
	Auth   *clients.VerifyAuthResponse `json:"auth,omitempty"`
}

// VerifyControlApiKey checks the configured control API key.
// It returns a status code indicating whether the key is missing, invalid,
// the server is unreachable, or the key is valid (with auth details).
func (m *Manager) VerifyControlApiKey() VerifyApiKeyResult {
	apiKey, err := m.settings.CopyTrading.ControlApi.ApiKey.Value()
	if err != nil || apiKey == "" {
		return VerifyApiKeyResult{Status: ApiKeyMissing}
	}

	client, err := clients.NewControlApi(m.settings.CopyTrading.ControlApi, m.settings.CopyTrading.Exchange.Accounts())
	if err != nil {
		return VerifyApiKeyResult{Status: ApiKeyMissing}
	}

	auth, err := client.VerifyAuth()
	if err != nil {
		if isAuthError(err) {
			return VerifyApiKeyResult{Status: ApiKeyInvalid}
		}
		return VerifyApiKeyResult{Status: ApiKeyUnreachable}
	}

	return VerifyApiKeyResult{Status: ApiKeyValid, Auth: auth}
}

// isAuthError checks whether the error indicates an authentication failure (401/403).
func isAuthError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "status code: 401") || strings.Contains(msg, "status code: 403")
}

// DeleteControlApiKey removes the control API key from the keyring and settings.
func (m *Manager) DeleteControlApiKey() error {
	oldSecret := m.settings.CopyTrading.ControlApi.ApiKey

	rawKey, err := oldSecret.Value()
	if err == nil && rawKey != "" {
		parts := strings.SplitN(rawKey, ".", 2)
		keyID := parts[0]
		if keyID != "" {
			client, cerr := clients.NewControlApi(m.settings.CopyTrading.ControlApi, m.settings.CopyTrading.Exchange.Accounts())
			if cerr != nil {
				m.logger.Warn("could not create control API client to revoke key on server", "error", cerr)
			} else if derr := client.DeleteApiKey(keyID); derr != nil {
				m.logger.Warn("could not revoke control API key on server", "error", derr)
			}
		}
	} else {
		m.logger.Warn("could not get control API key from settings", "error", err)
	}

	if oldSecret.FromKeyring != nil {
		_ = keyring.Delete(*oldSecret.FromKeyring)
	}

	m.settings.CopyTrading.ControlApi.ApiKey = secrets.Secret{}

	if err := m.save(); err != nil {
		m.settings.CopyTrading.ControlApi.ApiKey = oldSecret
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

// ExchangeCodeForApiKey exchanges a PKCE authorization code and verifier for an API key,
// stores it in the keyring, and updates settings to reference it.
func (m *Manager) ExchangeCodeForApiKey(code, codeVerifier string) (*clients.CreateApiKeyResponse, error) {
	if code == "" || codeVerifier == "" {
		return nil, fmt.Errorf("code and code_verifier are required")
	}

	controlApiKeyringEntry := CONTROL_API_KEYRING

	// Create an unauthenticated control API client (the exchange endpoint is unauthenticated)
	controlApi := clients.NewControlApiUnauthenticated(m.settings.CopyTrading.ControlApi)

	// Build API key name from hostname and OS
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	keyName := fmt.Sprintf("%s-%s", hostname, runtime.GOOS)

	// Exchange code for API key
	apiKeyResp, err := controlApi.ExchangeCode(code, codeVerifier, keyName)
	if err != nil {
		return nil, fmt.Errorf("exchanging code for API key: %w", err)
	}

	oldSecret := m.settings.CopyTrading.ControlApi.ApiKey

	// Capture existing keyring value for rollback
	oldKeyringValue, hadOldKeyring := "", false
	if oldSecret.FromKeyring != nil {
		if v, err := keyring.Get(*oldSecret.FromKeyring); err == nil {
			oldKeyringValue = v
			hadOldKeyring = true
		}
	}

	if err := keyring.Set(controlApiKeyringEntry, apiKeyResp.Key); err != nil {
		return nil, fmt.Errorf("storing API key in keyring: %w", err)
	}

	m.settings.CopyTrading.ControlApi.ApiKey = secrets.Secret{
		FromKeyring: &controlApiKeyringEntry,
	}

	if err := m.save(); err != nil {
		m.settings.CopyTrading.ControlApi.ApiKey = oldSecret
		if hadOldKeyring {
			_ = keyring.Set(controlApiKeyringEntry, oldKeyringValue)
		} else {
			_ = keyring.Delete(controlApiKeyringEntry)
		}
		return nil, fmt.Errorf("saving settings: %w", err)
	}

	return apiKeyResp, nil
}

type LogsBundle struct {
	ServiceLogsSize int `json:"serviceLogsSize"`
	GuiLogsSize     int `json:"guiLogsSize"`
}

type ReportSubmitResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// prepareLogs reads and concatenates the lumberjack log files for the given log
// type. Backup files are sorted oldest-first (lumberjack uses an ISO timestamp
// suffix that is lexicographically sortable), followed by the current active
// file. Missing files are silently skipped; an empty slice is returned when no
// files exist at all. When maxFiles is non-nil only the most recent maxFiles
// files are included; when nil all files are concatenated.
func (m *Manager) prepareLogs(logType logging.LogType, maxFiles *int) ([]byte, error) {
	logDir, err := logging.LogDir()
	if err != nil {
		return nil, fmt.Errorf("resolving log directory: %w", err)
	}

	baseName := string(logType)

	// Glob for rotated backup files: e.g. service-2024-01-02T15-04-05.000.log
	pattern := filepath.Join(logDir, baseName+"-*.log")
	backups, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("listing backup log files: %w", err)
	}

	// Sort lexicographically so the oldest timestamp comes first.
	sort.Strings(backups)

	currentFile := filepath.Join(logDir, baseName+".log")

	// Build the ordered list: backups (oldest→newest) then the active file.
	var files []string
	files = append(files, backups...)
	if _, statErr := os.Stat(currentFile); statErr == nil {
		files = append(files, currentFile)
	}

	// Keep only the most recent files when a limit is provided.
	if maxFiles != nil && len(files) > *maxFiles {
		files = files[len(files)-*maxFiles:]
	}

	if len(files) == 0 {
		return nil, nil
	}

	var combined []byte
	for _, f := range files {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			// Skip files that cannot be read (e.g. permission issues) rather
			// than aborting the whole bundle.
			continue
		}
		combined = append(combined, data...)
	}

	return combined, nil
}

// PrepareSupportLogs reads the service and GUI log bundles and returns their
// sizes in bytes. This is called by the frontend when the user enables the
// "Attach logs" checkbox so the UI can display the bundle size before submission.
// The actual log bytes stay in Go and are re-read at submit time.
// When maxFiles is non-nil only the most recent maxFiles files per log type are
// included; when nil all files are concatenated.
func (m *Manager) PrepareSupportLogs(maxFiles *int) (LogsBundle, error) {
	serviceLogs, err := m.prepareLogs(logging.ServiceLog, maxFiles)
	if err != nil {
		return LogsBundle{}, fmt.Errorf("preparing service logs: %w", err)
	}
	guiLogs, err := m.prepareLogs(logging.GUILog, maxFiles)
	if err != nil {
		return LogsBundle{}, fmt.Errorf("preparing GUI logs: %w", err)
	}
	return LogsBundle{
		ServiceLogsSize: len(serviceLogs),
		GuiLogsSize:     len(guiLogs),
	}, nil
}

// SubmitReport collects OS/user-agent information and the application runtime,
// optionally attaches the local log bundle, and posts the report to the control
// API. It always returns a ReportSubmitResult — the frontend should inspect the
// Success field rather than catching a rejected promise.
// When maxFiles is non-nil only the most recent maxFiles files per log type are
// included; when nil all files are concatenated.
func (m *Manager) SubmitReport(message string, withLogs bool, maxFiles *int) ReportSubmitResult {
	ua := fmt.Sprintf("copy-service/%s-%s (%s)", version.Version, version.Runtime, runtime.GOOS)

	var serviceLogs, guiLogs []byte
	if withLogs {
		var err error
		serviceLogs, err = m.prepareLogs(logging.ServiceLog, maxFiles)
		if err != nil {
			return ReportSubmitResult{
				Success: false,
				Message: "Your report could not be submitted. Could not read service logs.",
			}
		}
		guiLogs, err = m.prepareLogs(logging.GUILog, maxFiles)
		if err != nil {
			return ReportSubmitResult{
				Success: false,
				Message: "Your report could not be submitted. Could not read GUI logs.",
			}
		}
	}

	cfg := m.settings.CopyTrading
	controlApi, err := clients.NewControlApi(cfg.ControlApi, cfg.Exchange.Accounts())
	if err != nil {
		return ReportSubmitResult{
			Success: false,
			Message: "Your report could not be submitted. Please ensure you are logged in and try again.",
		}
	}

	res, err := controlApi.SendReport(clients.CopyServiceReportRequest{
		UserAgent:   ua,
		Message:     message,
		WithLogs:    withLogs,
		ServiceLogs: serviceLogs,
		GuiLogs:     guiLogs,
	})
	if err != nil {
		return ReportSubmitResult{
			Success: false,
			Message: "Your report could not be submitted. Please check your connection and try again.",
		}
	}

	var responseMessage = "Your report has been submitted successfully. Thank you for helping us improve the service."
	if !res.Success {
		responseMessage = fmt.Sprintf("Your report could not be submitted. %s Please check your connection and try again.", res.Cause)
	}
	return ReportSubmitResult{
		Success: res.Success,
		Message: responseMessage,
	}
}

// GetAccountLabels retrieves the user's account labels keyed by account ID
// from the control API.
func (m *Manager) GetAccountLabels() (map[string]string, error) {
	cfg := m.settings.CopyTrading
	controlApi, err := clients.NewControlApi(cfg.ControlApi, cfg.Exchange.Accounts())
	if err != nil {
		return nil, fmt.Errorf("creating control API client: %w", err)
	}
	return controlApi.GetAccountLabels()
}

// SetAccountLabel sets the label for an account on the control API.
// An empty label clears it.
func (m *Manager) SetAccountLabel(accountID, label string) error {
	cfg := m.settings.CopyTrading
	controlApi, err := clients.NewControlApi(cfg.ControlApi, cfg.Exchange.Accounts())
	if err != nil {
		return fmt.Errorf("creating control API client: %w", err)
	}
	return controlApi.SetAccountLabel(accountID, label)
}
