package gui

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// callbackResult holds the code and state received from the browser callback.
type callbackResult struct {
	code  string
	state string
}

// Login starts a local callback server, opens the login URL in the browser,
// and waits for the website to redirect back with a PKCE authorization code.
// It then exchanges the code for a control API key.
func (a *App) Login() error {
	// Per-login cancellable context so the user can abort a pending login
	// (e.g. closed the browser tab) without quitting the whole app.
	loginCtx, cancel := context.WithCancel(a.ctx)
	a.loginMu.Lock()
	if a.loginCancel != nil {
		a.loginCancel() // supersede any stale in-flight login
	}
	a.loginGen++
	gen := a.loginGen
	a.loginCancel = cancel
	a.loginMu.Unlock()
	defer func() {
		cancel()
		a.loginMu.Lock()
		// Only clear if a newer login hasn't replaced ours.
		if a.loginGen == gen {
			a.loginCancel = nil
		}
		a.loginMu.Unlock()
	}()

	// Pick a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting callback listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Generate PKCE parameters
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		listener.Close()
		return fmt.Errorf("generating code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	challengeHash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		listener.Close()
		return fmt.Errorf("generating state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Build login URL with callback and PKCE parameters
	loginURL, err := url.Parse(a.Settings.GetLoginURL())
	if err != nil {
		listener.Close()
		return fmt.Errorf("parsing login URL: %w", err)
	}
	q := loginURL.Query()
	q.Set("callback", callbackURL)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	loginURL.RawQuery = q.Encode()

	resultCh := make(chan callbackResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		returnedState := r.URL.Query().Get("state")
		if code == "" || returnedState == "" {
			http.Error(w, "missing code or state parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><p>Login successful. You can close this tab.</p><script>window.close()</script></body></html>`)

		resultCh <- callbackResult{code: code, state: returnedState}
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Emit the fully-built URL so the frontend can offer a manual "open"/"copy"
	// fallback in case the default browser fails to launch, then open the browser.
	runtime.EventsEmit(a.ctx, "login:url", loginURL.String())
	runtime.BrowserOpenURL(a.ctx, loginURL.String())

	// Wait for callback or timeout
	select {
	case result := <-resultCh:
		if result.state != state {
			return fmt.Errorf("state mismatch: possible CSRF attack")
		}
		_, err := a.Settings.ExchangeCodeForApiKey(result.code, codeVerifier)
		if err != nil {
			return fmt.Errorf("exchanging code for API key: %w", err)
		}
		return nil
	case err := <-errCh:
		return fmt.Errorf("callback server error: %w", err)
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("login timed out")
	case <-loginCtx.Done():
		// Cancelled by the user (CancelLogin) or app shutdown (a.ctx).
		return fmt.Errorf("login cancelled")
	}
}

// CancelLogin aborts an in-flight Login started from the GUI, unblocking it
// immediately. Safe to call when no login is pending.
func (a *App) CancelLogin() {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	if a.loginCancel != nil {
		a.loginCancel()
	}
}
