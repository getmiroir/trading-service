package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/getmiroir/trading-service/internal/exchange"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/svc"
	"github.com/getmiroir/trading-service/internal/version"
)

func main() {
	version.Runtime = version.RuntimeService

	svcAction := flag.String("service", "", "Service control action: install, uninstall, start, stop, restart, status")
	_ = flag.Bool("run-as-service", false, "Compatibility flag for service units; ignored in cmd/copy-service")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version)
		return
	}

	logging.SetDefaultStdoutLogging("DEBUG") // Will be overridden by settings
	logger := logging.NewLogger("main")
	logger.Info("Version", "version", version.Version)

	handled, err := svc.HandleAction(*svcAction, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if handled {
		return
	}

	cfg, err := settings.LoadSettings()
	if err != nil {
		logger.Error("Failed to load settings", "error", err)
		os.Exit(1)
	}

	lgr, err := logging.Setup(logging.ServiceLog, cfg.CopyTrading.Service.LogLevel, cfg.CopyTrading.Logging)
	if err != nil {
		logger.Warn("file logging unavailable", "error", err)
	} else {
		defer lgr.Close()
		lgr.SetSecrets(settings.ResolveSecrets(cfg))
	}

	if err := exchange.ValidateAccounts(cfg); err != nil {
		logger.Error("Account validation failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting copy trading service")

	if err := svc.RunService(cfg, logger); err != nil {
		logger.Error("Service run error", "error", err)
		os.Exit(1)
	}
}
