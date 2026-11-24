package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/getmiroir/trading-service/internal/gui"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/svc"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/kardianos/service"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

const defaultInstanceID = "3f16aa0b-f63a-4c4a-9a73-ae13c2b6641f"

func instanceID() string {
	if id := os.Getenv("WAILS_APP_INSTANCE_ID"); id != "" {
		return id
	}
	return defaultInstanceID
}

// shouldRunDaemonService decides between the headless copy-trading service and the Wails GUI.
// On macOS, kardianos/service reports Interactive()==false when PPID is 1 (launchd), which is
// also true for Finder/Dock-launched .app bundles, so we treat those as GUI unless
// -run-as-service is set (launchd/systemd after install).
func shouldRunDaemonService(explicitRunAsService bool) bool {
	if explicitRunAsService {
		return true
	}
	if service.Interactive() {
		return false
	}
	if runtime.GOOS == "darwin" && isMacOSAppBundle() {
		return false
	}
	return true
}

func isMacOSAppBundle() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(exe, ".app/Contents/MacOS/")
}

//go:embed frontend/dist
var assets embed.FS

func main() {
	version.Runtime = version.RuntimeGUI

	svcAction := flag.String("service", "", "Service control action: install, uninstall, start, stop, restart, status")
	runAsService := flag.Bool("run-as-service", false, "Run as foreground service (set by launchd/systemd plist; not for manual use)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version)
		return
	}

	logging.SetDefaultStdoutLogging("INFO")
	logger := logging.NewLogger("copy-gui")
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

	logging.SetDefaultStdoutLogging(cfg.CopyTrading.Service.LogLevel)
	isDaemonService := shouldRunDaemonService(*runAsService)

	logType := logging.GUILog
	if isDaemonService {
		logType = logging.ServiceLog
	}
	lgr, err := logging.Setup(logType, cfg.CopyTrading.Service.LogLevel, cfg.CopyTrading.Logging)
	if err != nil {
		logger.Warn("file logging unavailable", "error", err)
	} else {
		defer lgr.Close()
		lgr.SetSecrets(settings.ResolveSecrets(cfg))
	}

	if isDaemonService {
		logger.Info("Starting copy trading service")
		if err := svc.RunService(cfg, logger); err != nil {
			logger.Error("Service run error", "error", err)
			os.Exit(1)
		}
		return
	}

	app, err := gui.NewApp(cfg)
	if err != nil {
		logger.Error("Failed to create GUI app", "error", err)
		os.Exit(1)
	}

	// When macOS (or another OS that supports custom URL schemes) launches us
	// via `miroir://...`, the URL arrives as an argv entry. Buffer it now; it
	// will be emitted to the frontend once the Wails context is ready.
	if deepLink := gui.FindDeepLinkInArgs(os.Args[1:]); deepLink != "" {
		app.EnqueueDeepLink(deepLink)
	}

	// Systray native start must run on the main thread (here), not in Wails OnStartup.
	app.InitTray()

	err = wails.Run(&options.App{
		Title:       "Miroir Trading App",
		Width:       gui.DefaultWindowWidth,
		Height:      gui.DefaultWindowHeight,
		MinWidth:    gui.MinWindowWidth,
		MinHeight:   gui.MinWindowHeight,
		StartHidden: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               instanceID(),
			OnSecondInstanceLaunch: app.OnSecondInstanceLaunch,
		},
		Linux: &linux.Options{
			Icon: gui.Icon(),
		},
		OnStartup:     app.Startup,
		OnBeforeClose: app.BeforeClose,
		OnShutdown:    app.Shutdown,
		Bind: []interface{}{
			app,
			app.Settings,
		},
	})
	if err != nil {
		logger.Error("Wails error", "error", err)
		os.Exit(1)
	}
}
