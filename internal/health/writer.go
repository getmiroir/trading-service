package health

import (
	"context"
	"log/slog"
	"time"
)

// RunWriter opens the health database and writes a heartbeat every interval
// until the context is cancelled. It is intended to run as an independent
// goroutine launched at service start, so the GUI can detect liveness even if
// the main copy loop fails. Errors are logged and do not stop the loop.
func RunWriter(ctx context.Context, interval time.Duration, version string, logger *slog.Logger) {
	db, err := Open()
	if err != nil {
		logger.Error("Failed to open health database", "error", err)
		return
	}
	defer db.Close()

	if err := db.WriteHeartbeat(version); err != nil {
		logger.Error("Failed to write initial heartbeat", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := db.WriteHeartbeat(version); err != nil {
				logger.Error("Failed to write heartbeat", "error", err)
			}
		}
	}
}
