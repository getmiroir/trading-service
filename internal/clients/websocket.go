package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"log/slog"

	"github.com/getmiroir/trading-service/internal/logging"
	"golang.org/x/net/websocket"
)

// Base websocket client.
// Connect opens the socket and authenticates.
// ConnectWithSupervisor additionally starts a background goroutine that pings on every pingInterval.
// If the ping fails (or the connection is already down) it reconnects with exponential backoff.
// SendAndReceive notifies the supervisor immediately on failure so reconnect is not delayed until the next ping.
type Websocket struct {
	conn *websocket.Conn
	url  string

	// mutexes for concurrent access
	connMu    sync.RWMutex // Protects the connection assignment and nil checks
	wsWriteMu sync.Mutex   // Protects concurrent writes to the websocket

	pingInterval   time.Duration
	reconnectDelay time.Duration // initial backoff; doubles up to 1 minute
	timeout        time.Duration // dial timeout and per-request deadline

	stopSupervisor  chan struct{}
	reconnectNotify chan struct{} // buffered; takes the first reconnect request only
	supervisorWg    sync.WaitGroup

	connector WebsocketConnector

	logger *slog.Logger
}

func NewWebsocket(
	url string,
	pingInterval time.Duration,
	reconnectDelay time.Duration,
	timeout time.Duration,
) *Websocket {
	return &Websocket{
		url:             url,
		pingInterval:    pingInterval,
		reconnectDelay:  reconnectDelay,
		timeout:         timeout,
		stopSupervisor:  make(chan struct{}),
		reconnectNotify: make(chan struct{}, 1),
		logger:          logging.NewLogger("websocket"),
	}
}

func (w *Websocket) SetConnector(c WebsocketConnector) {
	w.connector = c
}

func (w *Websocket) IsConnected() bool {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	return w.conn != nil
}

func (w *Websocket) markDisconnected() {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
}

func (w *Websocket) requestReconnect() {
	select {
	case w.reconnectNotify <- struct{}{}:
		// reconnect request sent
	default:
		// reconnect request not sent (channel is full)
	}
}

// connect opens the TCP+WS connection and authenticates. It does not start
// any background goroutine.
func (w *Websocket) Connect() error {
	config, err := websocket.NewConfig(w.url, "http://localhost")
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()
	ws, err := config.DialContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	w.connMu.Lock()
	w.conn = ws
	w.connMu.Unlock()

	w.logger.Info("Connected to websocket", "url", w.url)

	if err := w.connector.Authenticate(); err != nil {
		w.markDisconnected()
		return fmt.Errorf("websocket failed to authenticate: %w", err)
	}
	return nil
}

// ConnectWithSupervisor attempts an initial connection and starts the
// supervisor goroutine. A failed initial connection is logged but not fatal —
// the supervisor will keep retrying.
func (w *Websocket) ConnectWithSupervisor(ctx context.Context) {
	if err := w.Connect(); err != nil {
		w.logger.Warn("Initial websocket connect failed, supervisor will retry", "url", w.url, "error", err)
	}
	w.startSupervisor(ctx)
}

func (w *Websocket) startSupervisor(ctx context.Context) {
	w.supervisorWg.Add(1)
	go func() {
		defer w.supervisorWg.Done()
		ticker := time.NewTicker(w.pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopSupervisor:
				return
			case <-w.reconnectNotify:
				w.logger.Info("Websocket disconnected, reconnecting", "url", w.url)
				w.reconnectWithBackoff(ctx)
			case <-ticker.C:
				if w.IsConnected() {
					if err := w.connector.Ping(); err != nil {
						w.logger.Error("Ping failed, reconnecting", "url", w.url, "error", err)
						w.markDisconnected()
						w.reconnectWithBackoff(ctx)
					}
				} else {
					w.logger.Info("Websocket disconnected, reconnecting", "url", w.url)
					w.reconnectWithBackoff(ctx)
				}
			}
		}
	}()
}

func (w *Websocket) reconnectWithBackoff(ctx context.Context) {
	if err := w.connector.ResetOnDisconnect(); err != nil {
		w.logger.Warn("ResetOnDisconnect failed", "error", err)
	}

	backoff := w.reconnectDelay
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopSupervisor:
			return
		default:
		}

		if err := w.Connect(); err != nil {
			w.logger.Warn("Reconnect attempt failed", "url", w.url, "error", err)
		} else {
			w.logger.Info("Reconnected websocket", "url", w.url)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-w.stopSupervisor:
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// Close shuts down the supervisor and closes the connection. The connection is
// closed before waiting for the supervisor so that a request blocked on it
// (e.g. a ping) fails right away instead of holding the shutdown.
func (w *Websocket) Close() error {
	select {
	case <-w.stopSupervisor:
		// channel already closed
	default:
		close(w.stopSupervisor)
	}
	err := w.closeConn()
	w.supervisorWg.Wait()

	// A reconnect racing with the shutdown may have opened a new connection.
	if lateErr := w.closeConn(); err == nil {
		err = lateErr
	}
	return err
}

func (w *Websocket) closeConn() error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	if w.conn == nil {
		return nil
	}
	err := w.conn.Close()
	w.conn = nil
	if err != nil {
		return fmt.Errorf("failed to close websocket: %w", err)
	}
	return nil
}

func (w *Websocket) SendAndReceive(message interface{}, response interface{}) error {
	if !w.IsConnected() {
		w.requestReconnect()
		return fmt.Errorf("websocket not connected, supervisor will reconnect")
	}

	w.wsWriteMu.Lock()
	defer w.wsWriteMu.Unlock()

	w.connMu.RLock()
	conn := w.conn
	w.connMu.RUnlock()
	if conn == nil {
		w.requestReconnect()
		return fmt.Errorf("websocket connection is not open")
	}

	if err := conn.SetDeadline(time.Now().Add(w.timeout)); err != nil {
		w.markDisconnected()
		w.requestReconnect()
		return fmt.Errorf("failed to set websocket deadline: %w", err)
	}

	if err := websocket.JSON.Send(conn, message); err != nil {
		w.markDisconnected()
		w.requestReconnect()
		return fmt.Errorf("failed to send message: %w", err)
	}

	if err := websocket.JSON.Receive(conn, response); err != nil {
		w.markDisconnected()
		w.requestReconnect()
		return fmt.Errorf("failed to receive response: %w", err)
	}

	return nil
}
