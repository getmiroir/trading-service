package gui

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings"
	"github.com/getmiroir/trading-service/internal/settings/manager"
	"github.com/getmiroir/trading-service/internal/svc"
	"github.com/getmiroir/trading-service/internal/version"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	DefaultWindowWidth  = 500
	DefaultWindowHeight = 800
	MinWindowWidth      = 300
	MinWindowHeight     = 500
	MaxWindowHeight     = 1200
)

// StatusInfo is the JSON payload sent to the frontend.
type StatusInfo struct {
	Status string `json:"status"`
}

// App is the combined Wails + systray application struct.
type App struct {
	ctx      context.Context
	cancel   context.CancelFunc
	monitor  *svc.Monitor
	logger   *slog.Logger
	Settings *manager.Manager

	versionItem   *systray.MenuItem
	statusItem    *systray.MenuItem
	showItem      *systray.MenuItem
	startItem     *systray.MenuItem
	stopItem      *systray.MenuItem
	installItem   *systray.MenuItem
	uninstallItem *systray.MenuItem
	quitItem      *systray.MenuItem

	trayEnd  func() // cleanup function from RunWithExternalLoop
	quitting atomic.Bool
	ctxReady chan struct{} // closed when ctx is set by Startup

	graceMu        sync.Mutex
	graceUntil     time.Time
	currentStatus  svc.ServiceStatus
	expectedStatus svc.ServiceStatus

	deepLinkMu      sync.Mutex
	pendingDeepLink string

	loginMu     sync.Mutex
	loginCancel context.CancelFunc
	loginGen    uint64
}

func NewApp(settings *settings.Settings) (*App, error) {
	monitor, err := svc.NewMonitor()
	if err != nil {
		return nil, err
	}
	mgr, err := manager.New("", settings)
	if err != nil {
		return nil, err
	}
	return &App{monitor: monitor, logger: logging.NewLogger("wails-app"), Settings: mgr, ctxReady: make(chan struct{})}, nil
}

// InitTray must be called on the main goroutine before wails.Run().
// On macOS, NSStatusItem/NSStatusBarWindow must be created on the AppKit main thread.
// Wails invokes OnStartup on a worker goroutine, so native tray start cannot run there.
func (a *App) InitTray() {
	start, end := systray.RunWithExternalLoop(a.onTrayReady, a.onTrayExit)
	a.trayEnd = end
	start()
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.currentStatus = svc.StatusUnknown
	close(a.ctxReady)

	a.resizeWindowToFitCurrentScreen()
	a.logger.Info("Start GUI", "stage", "wails_startup", "msg", "launch copy trading desktop application")

	a.flushPendingDeepLink()

	pollCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	statusCh := a.monitor.PollStatus(pollCtx, a.healthInterval())

	go func() {
		for status := range statusCh {
			a.graceMu.Lock()
			if time.Now().Before(a.graceUntil) && status != a.expectedStatus {
				a.graceMu.Unlock()
				continue
			}
			a.graceUntil = time.Time{}
			a.expectedStatus = 0
			a.graceMu.Unlock()
			a.updateTray(status)
			runtime.EventsEmit(a.ctx, "serviceStatus", StatusInfo{
				Status: status.String(),
			})
		}
	}()

	go func() {
		a.runVersionCheck(ctx)
	}()
}

func (a *App) resizeWindowToFitCurrentScreen() {
	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil || len(screens) == 0 {
		runtime.WindowSetMaxSize(a.ctx, DefaultWindowWidth, DefaultWindowHeight)
		return
	}

	currentScreen := screens[0]
	for _, screen := range screens {
		if screen.IsCurrent {
			currentScreen = screen
			break
		}
		if screen.IsPrimary {
			currentScreen = screen
		}
	}

	targetWidth := DefaultWindowWidth
	targetHeight := DefaultWindowHeight
	screenWidth := currentScreen.Size.Width
	screenHeight := currentScreen.Size.Height
	if targetWidth > screenWidth || targetHeight > screenHeight {
		targetHeight = (screenHeight * 80) / 100
		targetWidth = (DefaultWindowWidth * targetHeight) / DefaultWindowHeight
		if targetWidth > screenWidth {
			targetWidth = screenWidth
		}
	}

	maxHeight := MaxWindowHeight
	if maxHeight > screenHeight {
		maxHeight = screenHeight
	}
	runtime.WindowSetMaxSize(a.ctx, DefaultWindowWidth, maxHeight)
	runtime.WindowSetSize(a.ctx, targetWidth, targetHeight)
	runtime.WindowCenter(a.ctx)
}

// BeforeClose hides the window instead of quitting, so the tray keeps running.
// When an explicit quit is requested, it allows the close to proceed.
func (a *App) BeforeClose(_ context.Context) bool {
	if a.quitting.Load() {
		return false // allow actual close
	}
	runtime.WindowHide(a.ctx)
	return true // prevent close, just hide
}

func (a *App) Shutdown(_ context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.trayEnd != nil {
		a.trayEnd()
	}
	a.monitor.Close()
}

// healthInterval is the heartbeat poll interval, taken from settings.
func (a *App) healthInterval() time.Duration {
	return a.Settings.Settings().CopyTrading.Service.HealthInterval()
}

func (a *App) GetVersion() string {
	return version.Version
}

// GetEnabledClients returns the exchange clients enabled in the GUI. The GUI
// uses it to decide which exchanges to expose in the account picker.
func (a *App) GetEnabledClients() []string {
	enabled := a.Settings.Settings().CopyTrading.Features.EnabledClients
	out := make([]string, len(enabled))
	for i, c := range enabled {
		out[i] = string(c)
	}
	return out
}

func (a *App) GetExchangeReferralURL(exchangeType string) string {
	switch exchangeType {
	case "binance":
		return internal.BinanceReferralURL
	case "bybit":
		return internal.BybitReferralURL
	default:
		return ""
	}
}

func (a *App) GetStatus() StatusInfo {
	return StatusInfo{Status: a.monitor.CurrentStatus(a.healthInterval()).String()}
}

func (a *App) StartService() error {
	return a.controlWithTransition("start")
}

func (a *App) StopService() error {
	return a.controlWithTransition("stop")
}

func (a *App) RestartService() error {
	return a.controlWithTransition("restart")
}

func (a *App) InstallService() error {
	return a.controlWithTransition("install")
}

func (a *App) UninstallService() error {
	return a.controlWithTransition("uninstall")
}

const gracePeriodS = 5 * time.Second

func (a *App) controlWithTransition(action string) error {
	a.logger.Info("Controlling service", "action", action)
	var label string
	var expected svc.ServiceStatus
	switch action {
	case "start":
		label = "Starting..."
		expected = svc.StatusRunning
	case "stop":
		label = "Stopping..."
		expected = svc.StatusStopped
	case "restart":
		label = "Restarting..."
		expected = svc.StatusRunning
	case "install":
		label = "Installing..."
		expected = svc.StatusStopped
	case "uninstall":
		label = "Uninstalling..."
		expected = svc.StatusNotInstalled
	default:
		label = "Unknown"
	}

	a.graceMu.Lock()
	defer a.graceMu.Unlock()
	a.graceUntil = time.Now().Add(gracePeriodS)
	a.expectedStatus = expected

	a.updateTray(svc.StatusUnknown)

	runtime.EventsEmit(a.ctx, "serviceTransition", StatusInfo{Status: label})

	err := a.monitor.Control(action)

	// graceMu is held across Control, so statuses queued during a slow action
	// would leak once the original grace expired; restart it from completion.
	if err != nil {
		a.graceUntil = time.Time{}
		a.expectedStatus = 0
		a.logger.Error("Failed to execute service action", "action", action, "Error", err)
	} else {
		a.graceUntil = time.Now().Add(gracePeriodS)
	}

	return err
}

// OnSecondInstanceLaunch is called when a second instance of the app is launched.
// It brings the existing window to the foreground instead, and forwards any
// deep link URL that was passed on the command line to the frontend.
func (a *App) OnSecondInstanceLaunch(data options.SecondInstanceData) {
	if url := FindDeepLinkInArgs(data.Args); url != "" {
		a.EnqueueDeepLink(url)
	}
	a.showWindow()
}

// showWindow makes the Wails window visible and raises it to the front.
func (a *App) showWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

func (a *App) onTrayReady() {
	systray.SetIcon(TrayBytes(iconYellow))
	systray.SetTitle("Miroir")
	systray.SetTooltip("Miroir Trading App")

	// Left click on tray icon opens the GUI window directly.
	systray.SetOnTapped(func() {
		a.showWindow()
	})

	a.versionItem = systray.AddMenuItem(version.Version, "Application version")
	a.versionItem.Disable()

	a.statusItem = systray.AddMenuItem("Status: Unknown", "Current service status")
	a.statusItem.Disable()

	systray.AddSeparator()

	a.showItem = systray.AddMenuItem("Show", "Show the GUI window")

	systray.AddSeparator()

	a.startItem = systray.AddMenuItem("Start Service", "Start the copy trading service")
	a.stopItem = systray.AddMenuItem("Stop Service", "Stop the copy trading service")

	systray.AddSeparator()

	a.installItem = systray.AddMenuItem("Install Service", "Install the copy trading service")
	a.uninstallItem = systray.AddMenuItem("Uninstall Service", "Uninstall the copy trading service")

	systray.AddSeparator()

	a.quitItem = systray.AddMenuItem("Quit", "Quit the application")

	go a.handleTrayClicks()
}

func (a *App) onTrayExit() {
	a.logger.Info("Tray exiting")
}

func (a *App) handleTrayClicks() {
	<-a.ctxReady // wait for Startup to set a.ctx
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.showItem.ClickedCh:
			a.showWindow()
		case <-a.startItem.ClickedCh:
			if err := a.controlWithTransition("start"); err != nil {
				a.logger.Error("Failed to start service", "error", err)
			}
		case <-a.stopItem.ClickedCh:
			a.showWindow()
			runtime.EventsEmit(a.ctx, "serviceStopConfirmRequested")
		case <-a.installItem.ClickedCh:
			if err := a.controlWithTransition("install"); err != nil {
				a.logger.Error("Failed to install service", "error", err)
			}
		case <-a.uninstallItem.ClickedCh:
			if err := a.controlWithTransition("uninstall"); err != nil {
				a.logger.Error("Failed to uninstall service", "error", err)
			}
		case <-a.quitItem.ClickedCh:
			a.quitting.Store(true)
			runtime.Quit(a.ctx)
			return
		}
	}
}

func (a *App) updateTray(status svc.ServiceStatus) {
	if a.statusItem == nil {
		return // onTrayReady hasn't fired yet; poll goroutine will retry
	}
	if a.currentStatus == status {
		return
	}
	a.currentStatus = status
	switch status {
	case svc.StatusRunning:
		systray.SetIcon(TrayBytes(iconGreen))
		a.statusItem.SetTitle("Status: Running")
		a.startItem.Disable()
		a.stopItem.Enable()
		a.installItem.Disable()
		a.uninstallItem.Disable()
	case svc.StatusStopped:
		systray.SetIcon(TrayBytes(iconRed))
		a.statusItem.SetTitle("Status: Stopped")
		a.startItem.Enable()
		a.stopItem.Disable()
		a.installItem.Disable()
		a.uninstallItem.Enable()
	case svc.StatusFailed:
		systray.SetIcon(TrayBytes(iconRed))
		a.statusItem.SetTitle("Status: Failed")
		a.startItem.Disable()
		a.stopItem.Enable()
		a.installItem.Enable()
		a.uninstallItem.Enable()
	case svc.StatusNotInstalled:
		systray.SetIcon(TrayBytes(iconRed))
		a.statusItem.SetTitle("Status: Not Installed")
		a.startItem.Disable()
		a.stopItem.Disable()
		a.installItem.Enable()
		a.uninstallItem.Disable()
	default:
		systray.SetIcon(TrayBytes(iconYellow))
		a.statusItem.SetTitle("Status: Unknown")
		a.startItem.Disable()
		a.stopItem.Disable()
		a.installItem.Disable()
		a.uninstallItem.Disable()
	}
}
