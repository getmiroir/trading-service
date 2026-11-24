package clients

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/logging"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// HyperliquidExchangeWebsocket is a websocket client for Hyperliquid exchange actions.
// It handles per-request ECDSA signing and sends actions via the Hyperliquid WS post API.
type HyperliquidExchangeWebsocket struct {
	*Websocket
	account   subsettings.HyperliquidExchangeAccount
	creds     *secrets.SecretCache
	nextNonce func() (int64, error)
	logger    *slog.Logger
}

type hlWsPostRequest struct {
	Method  string        `json:"method"`
	ID      int64         `json:"id"`
	Request hlWsPostInner `json:"request"`
}

type hlWsPostInner struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type hlWsActionPayload struct {
	Action       hlOrderedMap `json:"action"`
	Nonce        int64        `json:"nonce"`
	Signature    HLSignature  `json:"signature"`
	VaultAddress *string      `json:"vaultAddress"`
}

type hlWsPostResponse struct {
	Channel string           `json:"channel"`
	Data    hlWsPostRespData `json:"data"`
}

type hlWsPostRespData struct {
	ID       int64             `json:"id"`
	Response hlWsPostRespInner `json:"response"`
}

type hlWsPostRespInner struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type hlWsPingRequest struct {
	Method string `json:"method"`
}

type hlWsPingResponse struct {
	Channel string `json:"channel"`
}

func NewHyperliquidExchangeWebsocket(
	account subsettings.HyperliquidExchangeAccount,
	pingInterval time.Duration,
	reconnectDelay time.Duration,
	timeout time.Duration,
	creds *secrets.SecretCache,
	nextNonce func() (int64, error),
) *HyperliquidExchangeWebsocket {
	url := resolveHyperliquidExchangeWebsocketURL(account.IsTestnet)
	ws := NewWebsocket(
		url,
		pingInterval,
		reconnectDelay,
		timeout,
	)
	hw := &HyperliquidExchangeWebsocket{
		Websocket: ws,
		account:   account,
		creds:     creds,
		nextNonce: nextNonce,
		logger:    logging.NewLogger("hyperliquid-exchange-websocket"),
	}
	hw.SetConnector(hw)
	return hw
}

func (hw *HyperliquidExchangeWebsocket) Authenticate() error {
	return nil
}

func (hw *HyperliquidExchangeWebsocket) ResetOnDisconnect() error {
	return nil
}

func (hw *HyperliquidExchangeWebsocket) Ping() error {
	req := hlWsPingRequest{Method: "ping"}
	var resp hlWsPingResponse
	if err := hw.SendAndReceive(req, &resp); err != nil {
		return err
	}
	if resp.Channel != "pong" {
		return fmt.Errorf("hyperliquid websocket ping: unexpected channel %q", resp.Channel)
	}
	return nil
}

// postAction signs an action and sends it via the websocket post API.
func (hw *HyperliquidExchangeWebsocket) postAction(action hlOrderedMap) (*HLExchangeResponse, error) {
	privKey, err := hw.creds.Get(hw.account.PrivateKey)
	if err != nil {
		return nil, newExchangeError(ErrApiKeyNotSet, fmt.Errorf("hyperliquid websocket: private key not set: %w", err))
	}

	if hw.nextNonce == nil {
		return nil, fmt.Errorf("hyperliquid websocket: nonce source not configured")
	}
	nonce, err := hw.nextNonce()
	if err != nil {
		return nil, err
	}
	isMainnet := !hw.account.IsTestnet

	sig, err := hlSignL1Action(privKey, action, hw.account.SubaccountAddress, nonce, isMainnet)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid websocket: sign action: %w", err)
	}

	req := hlWsPostRequest{
		Method: "post",
		ID:     nonce,
		Request: hlWsPostInner{
			Type: "action",
			Payload: hlWsActionPayload{
				Action:       action,
				Nonce:        nonce,
				Signature:    *sig,
				VaultAddress: hw.account.SubaccountAddress,
			},
		},
	}

	var resp hlWsPostResponse
	if err := hw.SendAndReceive(req, &resp); err != nil {
		return nil, err
	}

	var exchangeResp HLExchangeResponse
	if err := json.Unmarshal(resp.Data.Response.Payload, &exchangeResp); err != nil {
		return nil, fmt.Errorf("hyperliquid websocket: unmarshal response: %w", err)
	}

	if msg, isErr := exchangeResp.ErrorMessage(); isErr {
		return nil, &internal.HyperliquidExchangeError{Message: msg}
	}

	return &exchangeResp, nil
}
