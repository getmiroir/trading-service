package svc

import (
	"fmt"
	"log/slog"

	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/kardianos/service"
)

// HandleAction handles a --service CLI action (install, uninstall, start, stop, restart, status).
// Returns true if an action was handled and the caller should exit.
// Returns false if action is empty, meaning no service action was requested.
func HandleAction(action string, logger *slog.Logger) (bool, error) {
	if action == "" {
		return false, nil
	}

	svcConfig := NewServiceConfig()
	prg := &Program{Logger: logger}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return true, fmt.Errorf("failed to create service: %w", err)
	}

	if action == "status" {
		status, err := s.Status()
		if err != nil {
			return true, fmt.Errorf("service status query failed: %w", err)
		}
		switch status {
		case service.StatusRunning:
			fmt.Println("Service is running")
		case service.StatusStopped:
			fmt.Println("Service is stopped")
		default:
			fmt.Println("Service status unknown")
		}
		return true, nil
	}

	if err := service.Control(s, action); err != nil {
		return true, fmt.Errorf("service action %q failed: %w", action, err)
	}
	fmt.Printf("Service action %q succeeded\n", action)
	return true, nil
}

// RunService creates and runs the service (blocking until stopped).
func RunService(cfg *settings.Settings, logger *slog.Logger) error {
	svcConfig := NewServiceConfig()
	prg := &Program{Cfg: &cfg.CopyTrading, Logger: logger}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	return s.Run()
}
