package svc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/getmiroir/trading-service/internal/health"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/kardianos/service"
)

type ServiceStatus int

const (
	StatusUnknown ServiceStatus = iota
	StatusRunning
	StatusStopped
	StatusNotInstalled
	StatusFailed
)

func (s ServiceStatus) String() string {
	switch s {
	case StatusRunning:
		return "Running"
	case StatusStopped:
		return "Stopped"
	case StatusNotInstalled:
		return "Not Installed"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// noopProgram satisfies service.Interface so we can create a service handle
// without actually running anything.
type noopProgram struct{}

func (p *noopProgram) Start(service.Service) error { return nil }
func (p *noopProgram) Stop(service.Service) error  { return nil }

// Monitor tracks the OS service. Liveness is determined by the SQLite health
// heartbeat written by the running service (no subprocess), so the GUI never
// polls launchctl/systemctl in a loop. The kardianos Status() query is only
// used on discrete edges (startup while dead, an alive->dead transition, or
// right after a control action) to classify why the service is not alive.
type Monitor struct {
	svc    service.Service
	logger *slog.Logger
	health *health.DB
	dirty  atomic.Bool
}

func NewMonitor() (*Monitor, error) {
	cfg := NewServiceConfig()
	s, err := service.New(&noopProgram{}, cfg)
	if err != nil {
		return nil, err
	}
	m := &Monitor{svc: s, logger: logging.NewLogger("service-monitor")}
	if db, err := health.Open(); err != nil {
		m.logger.Warn("Failed to open health database; status will rely on service queries", "error", err)
	} else {
		m.health = db
	}
	return m, nil
}

// Close releases the health database handle, if open.
func (m *Monitor) Close() {
	if m.health != nil {
		m.health.Close()
	}
}

// unknownStatusRequeryInterval throttles kardianos Status() re-queries while
// the dead status is Unknown (e.g. systemd unit still deactivating).
const unknownStatusRequeryInterval = 2 * time.Second

// PollStatus emits the derived service status on every tick. The steady-state
// signal is the heartbeat (a cheap DB read); kardianos Status() is only invoked
// on the first tick while dead, on an alive->dead transition, after a control
// action (see Control), or periodically (throttled) while the status is Unknown
// so a transient state settles on its own.
func (m *Monitor) PollStatus(ctx context.Context, interval time.Duration) <-chan ServiceStatus {
	ch := make(chan ServiceStatus, 1)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		prevAlive := false
		first := true
		deadStatus := StatusUnknown
		var lastQuery time.Time

		emit := func() {
			alive := m.isAlive(interval)
			var status ServiceStatus
			if alive {
				status = StatusRunning
			} else {
				unknownRequeryDue := deadStatus == StatusUnknown && time.Since(lastQuery) >= unknownStatusRequeryInterval
				if first || prevAlive || m.dirty.Swap(false) || unknownRequeryDue {
					deadStatus = m.queryStatus()
					lastQuery = time.Now()
				}
				status = deadStatus
			}
			prevAlive = alive
			first = false
			ch <- status
		}

		emit()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				emit()
			}
		}
	}()

	return ch
}

// CurrentStatus returns the derived service status once. It reads the heartbeat
// and only falls back to a single kardianos Status() query when not alive.
func (m *Monitor) CurrentStatus(interval time.Duration) ServiceStatus {
	if m.isAlive(interval) {
		return StatusRunning
	}
	return m.queryStatus()
}

// isAlive reports whether the latest heartbeat is newer than twice the poll
// interval. Returns false when no heartbeat DB or row is available.
func (m *Monitor) isAlive(interval time.Duration) bool {
	if m.health == nil {
		return false
	}
	rec, err := m.health.ReadLatest()
	if err != nil {
		return false
	}
	return time.Since(rec.Time) < 2*interval
}

func (m *Monitor) queryStatus() ServiceStatus {
	status, err := m.svc.Status()
	if err != nil {
		if errors.Is(err, service.ErrNotInstalled) {
			// kardianos/systemd reports any unmapped is-active state (deactivating,
			// reloading, ...) as not installed, so trust it only if the unit file is gone.
			if m.svc.Platform() == "linux-systemd" && systemdUnitFileExists() {
				return StatusUnknown
			}
			return StatusNotInstalled
		}
		if isFailedStateError(err) {
			return StatusFailed
		}
		m.logger.Error("Failed to query service status", "error", err)
		return StatusUnknown
	}
	switch status {
	case service.StatusRunning:
		return StatusRunning
	case service.StatusStopped:
		return StatusStopped
	default:
		return StatusUnknown
	}
}

// systemdUnitFileExists reports whether the systemd user unit file is present,
// using the same path kardianos/service installs it to.
func systemdUnitFileExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(homeDir, ".config/systemd/user", serviceName()+".service"))
	return err == nil
}

// isFailedStateError checks whether the error returned by kardianos/service
// indicates a failed service. This only works on Linux/systemd — the library
// does not export a sentinel error, so we match on the raw error string.
// On Darwin and Windows the library never emits this error.
func isFailedStateError(err error) bool {
	return err.Error() == "service in failed state"
}

// Control runs an install/uninstall/start/stop/restart action via kardianos and
// marks the monitor dirty so the next dead poll re-queries the real status.
func (m *Monitor) Control(action string) error {
	m.dirty.Store(true)
	if err := service.Control(m.svc, action); err != nil {
		return err
	}
	if action == "uninstall" {
		if err := m.clearHeartbeat(); err != nil {
			m.logger.Warn("Failed to clear service heartbeat", "error", err)
		}
	}
	if action == "install" {
		if err := m.recordInstalledVersion(version.Version); err != nil {
			m.logger.Warn("Failed to record installed service version", "error", err)
		}
	}
	return nil
}

func (m *Monitor) recordInstalledVersion(v string) error {
	if m.health != nil {
		return m.health.WriteInstalledVersion(v)
	}
	db, err := health.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.WriteInstalledVersion(v)
}

func (m *Monitor) clearHeartbeat() error {
	if m.health != nil {
		return m.health.ClearHeartbeat()
	}
	db, err := health.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.ClearHeartbeat()
}

// QueryStatus returns the service status via a single kardianos query.
func (m *Monitor) QueryStatus() ServiceStatus {
	return m.queryStatus()
}

// ServiceVersion returns the version stored in the health DB (from install or
// the last running heartbeat) and whether a row was found.
func (m *Monitor) ServiceVersion() (string, bool) {
	if m.health == nil {
		return "", false
	}
	rec, err := m.health.ReadLatest()
	if err != nil {
		return "", false
	}
	return rec.Version, true
}

func (m *Monitor) WaitForStatus(ctx context.Context, expected ServiceStatus, timeoutS time.Duration, intervalMs time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeoutS)
	defer cancel()

	ticker := time.NewTicker(intervalMs)
	defer ticker.Stop()

	for {
		if m.queryStatus() == expected {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
