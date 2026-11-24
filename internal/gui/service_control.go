package gui

import (
	"context"
	"time"

	"github.com/getmiroir/trading-service/internal/svc"
)

const serviceStepIntervalMs = 300 * time.Millisecond

func (a *App) uninstallServiceInternal(ctx context.Context) error {
	if err := a.monitor.Control("uninstall"); err != nil {
		return err
	}
	return a.monitor.WaitForStatus(ctx, svc.StatusNotInstalled, autoUpgradeStepTimeoutS, serviceStepIntervalMs)
}

func (a *App) installServiceInternal(ctx context.Context) error {
	if err := a.monitor.Control("install"); err != nil {
		return err
	}
	return a.monitor.WaitForStatus(ctx, svc.StatusStopped, autoUpgradeStepTimeoutS, serviceStepIntervalMs)
}
