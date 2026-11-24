package gui

import (
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DeepLinkScheme is the custom URL scheme the desktop app is registered for.
// It must stay in sync with the `protocols` block in gui/wails.json.
const DeepLinkScheme = "miroir"

// DeepLinkEventName is the Wails event emitted to the frontend when a deep
// link is received (either via initial launch argv or via a second-instance
// re-launch from the OS).
const DeepLinkEventName = "deepLink"

// FindDeepLinkInArgs returns the first argv entry that looks like a deep link
// using DeepLinkScheme. Empty string when none is found.
func FindDeepLinkInArgs(args []string) string {
	prefix := DeepLinkScheme + "://"
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return arg
		}
	}
	return ""
}

// EnqueueDeepLink records a deep link URL to be delivered to the frontend.
// If the Wails context is already initialised, the event is emitted right
// away; otherwise it is buffered and flushed during Startup.
func (a *App) EnqueueDeepLink(url string) {
	if url == "" {
		return
	}
	a.deepLinkMu.Lock()
	if a.ctx == nil {
		a.pendingDeepLink = url
		a.deepLinkMu.Unlock()
		return
	}
	ctx := a.ctx
	a.deepLinkMu.Unlock()
	runtime.EventsEmit(ctx, DeepLinkEventName, url)
}

// flushPendingDeepLink emits any deep link that was buffered before the
// Wails context became available. Called from Startup.
func (a *App) flushPendingDeepLink() {
	a.deepLinkMu.Lock()
	url := a.pendingDeepLink
	a.pendingDeepLink = ""
	a.deepLinkMu.Unlock()
	if url == "" {
		return
	}
	runtime.EventsEmit(a.ctx, DeepLinkEventName, url)
}
