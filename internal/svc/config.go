package svc

import (
	"os"
	"path/filepath"

	"github.com/getmiroir/trading-service/internal/platform"
	"github.com/kardianos/service"
)

const (
	ServiceNameEnvVar  = "WAILS_APP_SERVICE_NAME"
	defaultServiceName = "miroir-trading-service"
	ServiceDisplayName = "Miroir Trading Service"
	ServiceDescription = "Synchronizes positions from Hyperliquid."
)

// ServiceName returns the service name, overridable via WAILS_APP_SERVICE_NAME.
func serviceName() string {
	if name := os.Getenv(ServiceNameEnvVar); name != "" {
		return name
	}
	return defaultServiceName
}

// NewServiceConfig returns a service.Config for the Miroir Trading Service.
// The service is installed as a user service (systemd user unit on Linux).
func NewServiceConfig() *service.Config {
	cfg := &service.Config{
		Name:        serviceName(),
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
		// Tells main() to run the foreground service; required on macOS because
		// Finder-launched GUI apps also have PPID 1 (launchd) and must not be
		// confused with a launchd-managed service process.
		Arguments: []string{"-run-as-service"},
		Option:    platformOptions(),
	}
	// Propagate the data directory and settings file overrides into the
	// installed service unit so the daemon and GUI resolve the same settings
	// file, health DB, and logs.
	envVars := map[string]string{}
	for _, name := range []string{platform.DataDirEnvVar, platform.SettingsFileEnvVar} {
		if val := os.Getenv(name); val != "" {
			if abs, err := filepath.Abs(val); err == nil {
				val = abs
			}
			envVars[name] = val
		}
	}
	if len(envVars) > 0 {
		cfg.EnvVars = envVars
	}
	return cfg
}
