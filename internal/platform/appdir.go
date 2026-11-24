package platform

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/getmiroir/trading-service/internal"
)

// DataDirEnvVar overrides the application data directory. When set, the SQLite
// health DB and logs live under this folder (and the settings file too, unless
// SettingsFileEnvVar is also set).
const DataDirEnvVar = "WAILS_APP_DATA_DIR"

// SettingsFileEnvVar overrides only the settings file location. When set, it
// takes precedence over DataDir()/settings.yaml, letting the settings file live
// in an arbitrary location (e.g. a mounted file in Docker) while the health DB
// and logs stay in the data directory.
const SettingsFileEnvVar = "WAILS_APP_SETTINGS_FILE"

const (
	settingsFileName = "settings.yaml"
	healthDBFileName = "service_health.db"
	logsDirName      = "logs"
)

// DataDir resolves the single application data directory.
// Resolution order: WAILS_APP_DATA_DIR env var > platform default
// (Windows: %PROGRAMDATA%/Miroir, otherwise ~/.miroir, falling back to the
// executable's directory if the home directory cannot be determined).
func DataDir() (string, error) {
	if env := os.Getenv(DataDirEnvVar); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs, nil
		}
		return env, nil
	}

	if runtime.GOOS == "windows" {
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = internal.ProgramDataDir
		}
		return filepath.Join(programData, internal.ProgramDataAppDirName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		exe, execErr := os.Executable()
		if execErr != nil {
			return "", err
		}
		return filepath.Join(filepath.Dir(exe), internal.AppDirName), nil
	}
	return filepath.Join(home, internal.AppDirName), nil
}

// SettingsPath returns the path to the settings file. When SettingsFileEnvVar
// is set it takes precedence; otherwise the file lives inside the data directory.
func SettingsPath() (string, error) {
	if env := os.Getenv(SettingsFileEnvVar); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs, nil
		}
		return env, nil
	}
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFileName), nil
}

// HealthDBPath returns the path to the service health SQLite DB inside the data directory.
func HealthDBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, healthDBFileName), nil
}

// LogsDir returns the logs directory inside the data directory.
func LogsDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, logsDirName), nil
}
