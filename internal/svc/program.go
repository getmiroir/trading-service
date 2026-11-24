package svc

import (
	"context"
	"log/slog"
	"time"

	"github.com/getmiroir/trading-service/internal/health"
	"github.com/getmiroir/trading-service/internal/runners"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/kardianos/service"
)

// Program implements service.Interface for kardianos/service.
type Program struct {
	Cfg    *settings.CopyTradingSettings
	Logger *slog.Logger
	cancel context.CancelFunc
	done   chan struct{}
}

func (p *Program) Start(s service.Service) error {
	p.done = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	// Heartbeat writer runs independently of the copy loop so the GUI can
	// detect liveness even if the copy loop fails.
	go health.RunWriter(ctx, p.Cfg.Service.HealthInterval(), version.Version, p.Logger)

	copyLoop := runners.NewCopyLoop()

	go func() {
		defer close(p.done)
		if err := copyLoop.Run(ctx, p.Cfg); err != nil && err != context.Canceled {
			p.Logger.Error("Copy loop error", "error", err)
		}
	}()

	return nil
}

func (p *Program) Stop(s service.Service) error {
	p.Logger.Info("Stopping service")
	p.cancel()
	timeout := p.Cfg.Service.StopTimeout()
	select {
	case <-p.done:
		p.Logger.Info("Service stopped")
	case <-time.After(timeout):
		p.Logger.Warn("Copy loop did not shut down in time, exiting anyway", "timeout", timeout)
	}
	return nil
}
