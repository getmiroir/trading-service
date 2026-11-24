package clients

type WebsocketConnector interface {
	Authenticate() error      // Authenticate the websocket connection
	ResetOnDisconnect() error // Reset exchange-specific state after the connection drops
	Ping() error              // Ping the websocket connection
}

type defaultConnector struct{}

func (d *defaultConnector) Authenticate() error      { return nil }
func (d *defaultConnector) ResetOnDisconnect() error { return nil }
func (d *defaultConnector) Ping() error              { return nil }
