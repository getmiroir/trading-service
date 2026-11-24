package settings

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getmiroir/trading-service/internal/platform"
	"go.yaml.in/yaml/v4"
)

// resolveSettingsPath resolves the settings file path inside the application
// data directory (see platform.DataDir).
func resolveSettingsPath() (string, error) {
	return platform.SettingsPath()
}

// ensureDefaultSettings ensures the data directory and default settings file exist.
func ensureDefaultSettings(settingsPath string) error {
	appDir := filepath.Dir(settingsPath)

	// Create the .{appDirName} directory if it doesn't exist
	if err := os.MkdirAll(appDir, 0750); err != nil {
		return fmt.Errorf("creating %s: %w", appDir, err)
	}

	// Create the settings file with defaults if it doesn't exist
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		defaults := DefaultSettings()
		data, err := marshalNonDefault(&defaults)
		if err != nil {
			return fmt.Errorf("marshaling default settings: %w", err)
		}
		if err := os.WriteFile(settingsPath, data, 0600); err != nil {
			return fmt.Errorf("writing default %s: %w", settingsPath, err)
		}
	}
	return nil
}

// LoadSettings loads settings from the settings file inside the application
// data directory (see platform.DataDir). Ensures the data directory exists and
// creates settings.yaml with defaults if missing.
// Returns an error if the file cannot be read or does not match the required schema (DisallowUnknownFields + Validate).
//
// When the file's version is older than currentSettingsVersion, LoadSettings
// migrates it and tries to save the result. If saving fails, the migrated
// settings are still returned.
func LoadSettings() (*Settings, error) {
	settingsPath, err := resolveSettingsPath()
	if err != nil {
		return nil, err
	}

	if err := ensureDefaultSettings(settingsPath); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	data, migrated, err := migrateSettingsData(data)
	if err != nil {
		return nil, fmt.Errorf("migrating %s: %w", settingsPath, err)
	}

	loaded, err := decodeSettings(data, settingsPath)
	if err != nil {
		return nil, err
	}

	if migrated {
		_ = writeSettingsFile(settingsPath, loaded)
	}

	return loaded, nil
}

// SaveSettings saves settings to a settings file.
// Path resolution order: configPath argument (if non-empty) > WAILS_APP_SETTINGS_FILE env var > ~/.{appDirName}/settings.yaml default.
// Returns an error if the file cannot be written.
func SaveSettings(settings *Settings) error {
	settingsPath, err := resolveSettingsPath()
	if err != nil {
		return err
	}
	return writeSettingsFile(settingsPath, settings)
}

func writeSettingsFile(settingsPath string, settings *Settings) error {
	data, err := marshalNonDefault(settings)
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	return nil
}

// ResolveSecrets returns the literal values of all API keys and secrets
// across all configured exchange accounts, for use in log redaction.
// Errors from keyring/env/file sources are silently skipped — a missing
// secret is simply not redacted rather than causing a failure.
func ResolveSecrets(cfg *Settings) []string {
	var out []string
	for _, acc := range cfg.CopyTrading.Exchange.Bybit.Accounts {
		if v, err := acc.Key.Value(); err == nil {
			out = append(out, v)
		}
		if v, err := acc.Secret.Value(); err == nil {
			out = append(out, v)
		}
	}
	for _, acc := range cfg.CopyTrading.Exchange.Binance.Accounts {
		if v, err := acc.Key.Value(); err == nil {
			out = append(out, v)
		}
		if v, err := acc.Secret.Value(); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func decodeSettings(data []byte, settingsPath string) (*Settings, error) {
	var loadedSettings Settings
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&loadedSettings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", settingsPath, err)
	}

	return &loadedSettings, nil
}
