package gui

import (
	"context"
	"log/slog"
	"time"

	"github.com/getmiroir/trading-service/internal/svc"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// VersionStatus is the outcome of comparing the GUI binary version to the
// version recorded at the last service install.
type VersionStatus string

const (
	VersionStatusOK        VersionStatus = "ok"
	VersionStatusUpgrade   VersionStatus = "upgrade"
	VersionStatusDowngrade VersionStatus = "downgrade"
	VersionStatusUnknown   VersionStatus = "unknown"
)

func (s VersionStatus) String() string {
	return string(s)
}

type VersionInfo struct {
	GUIVersion     string        `json:"guiVersion"`
	ServiceVersion string        `json:"serviceVersion"`
	Status         VersionStatus `json:"status"`
}

func (a *App) GetVersionStatus() VersionInfo {
	return a.versionStatus()
}

func (a *App) versionStatus() VersionInfo {
	svcVer, found := a.monitor.ServiceVersion()
	return compareInstalledServiceVersion(a.logger, svcVer, found)
}

func compareInstalledServiceVersion(log *slog.Logger, svcVer string, found bool) VersionInfo {
	guiVer := version.Version

	if !found {
		// No install record or heartbeat → older install without health DB.
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: "", Status: VersionStatusUnknown}
	}

	// A develop GUI always triggers a reinstall so that dev iteration stays fresh.
	if version.IsDevelop(guiVer) && version.IsDevelop(svcVer) {
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusUpgrade}
	}

	cmp, err := version.Compare(guiVer, svcVer)
	if err != nil {
		if guiVer == svcVer {
			return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusOK}
		}
		log.Warn("Version compare failed", "gui", guiVer, "svc", svcVer, "error", err)
		// if version are different, we notify the user but won't force install
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusDowngrade}
	}

	switch {
	case cmp > 0:
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusUpgrade}
	case cmp < 0:
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusDowngrade}
	default:
		return VersionInfo{GUIVersion: guiVer, ServiceVersion: svcVer, Status: VersionStatusOK}
	}
}

const (
	autoUpgradeStepTimeoutS = 5 * time.Second
	autoUpgradeGracePeriodS = 20 * time.Second
)

func (a *App) clearServiceGrace() {
	a.graceMu.Lock()
	a.graceUntil = time.Time{}
	a.expectedStatus = 0
	a.graceMu.Unlock()
}

func (a *App) runVersionCheck(ctx context.Context) {
	if a.monitor.QueryStatus() == svc.StatusNotInstalled {
		return
	}
	info := a.versionStatus()
	a.logger.Info("Version check", "gui", info.GUIVersion, "service", info.ServiceVersion, "status", info.Status)
	switch info.Status {
	case VersionStatusUpgrade, VersionStatusUnknown:
		a.autoUpgradeService(ctx)
	case VersionStatusDowngrade:
		a.handleDowngrade(info)
	}
}

// handleDowngrade emits the versionDowngrade event when a version downgrade is
// detected (installed service is newer than this GUI binary) so the UI can
// display the remediation card. The service is left running until the user
// pauses it manually.
func (a *App) handleDowngrade(info VersionInfo) {
	a.logger.Warn("Version downgrade detected", "gui", info.GUIVersion, "service", info.ServiceVersion)
	runtime.EventsEmit(a.ctx, "versionDowngrade", info)
}

// autoUpgradeService stops, uninstalls, reinstalls, and starts the OS service
// using the current binary. Called automatically on startup when a version
// mismatch is detected.
func (a *App) autoUpgradeService(ctx context.Context) {
	a.logger.Info("Auto-upgrade: starting", "version", version.Version)

	a.graceMu.Lock()
	a.graceUntil = time.Now().Add(autoUpgradeGracePeriodS)
	a.expectedStatus = svc.StatusRunning
	a.graceMu.Unlock()

	runtime.EventsEmit(a.ctx, "serviceTransition", StatusInfo{Status: "Upgrading service..."})

	intervalMs := serviceStepIntervalMs
	current := a.monitor.QueryStatus()
	if current == svc.StatusRunning || current == svc.StatusFailed {
		if err := a.monitor.Control("stop"); err != nil {
			a.logger.Error("Auto-upgrade: failed to stop service", "error", err)
			a.clearServiceGrace()
			return
		}
		if err := a.monitor.WaitForStatus(ctx, svc.StatusStopped, autoUpgradeStepTimeoutS, intervalMs); err != nil {
			a.logger.Error("Auto-upgrade: timed out waiting for service to stop", "error", err)
			a.clearServiceGrace()
			return
		}
	}

	if err := a.uninstallServiceInternal(ctx); err != nil {
		a.logger.Error("Auto-upgrade: failed to uninstall service", "error", err)
		a.clearServiceGrace()
		return
	}

	if err := a.installServiceInternal(ctx); err != nil {
		a.logger.Error("Auto-upgrade: failed to install service", "error", err)
		a.clearServiceGrace()
		return
	}

	if err := a.monitor.Control("start"); err != nil {
		a.logger.Error("Auto-upgrade: failed to start service", "error", err)
		a.clearServiceGrace()
		return
	}
	a.logger.Info("Auto-upgrade: complete", "version", version.Version)
}
