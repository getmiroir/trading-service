package clients

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

type BybitWebsocket struct {
	*Websocket
	key    secrets.Secret
	secret secrets.Secret
	creds  *secrets.SecretCache
}

func NewBybitWebsocket(
	account subsettings.BybitAccount,
	pingInterval time.Duration,
	reconnectDelay time.Duration,
	timeout time.Duration,
	creds *secrets.SecretCache,
) *BybitWebsocket {
	bw := &BybitWebsocket{
		Websocket: NewWebsocket(
			resolveBybitWebsocketURL(account.IsDemo, account.IsTestnet, account.Region),
			pingInterval,
			reconnectDelay,
			timeout,
		),
		key:    account.Key,
		secret: account.Secret,
		creds:  creds,
	}
	bw.SetConnector(bw)
	return bw
}

// generateSignature generates a signature for the authentication of the websocket connection
func (bw *BybitWebsocket) generateSignature(expires int64) (interface{}, error) {
	apiSecret, secretErr := bw.creds.Get(bw.secret)
	if secretErr != nil {
		return nil, newExchangeError(ErrApiKeyNotSet, &internal.BybitError{Message: "Bybit API secret is not set: " + secretErr.Error()})
	}
	message := fmt.Sprintf("GET/realtime%d", expires)
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Authenticate authenticates the websocket connection
// Authenticate implements WebsocketConnector
func (bw *BybitWebsocket) Authenticate() error {
	apiKey, err := bw.creds.Get(bw.key)
	if err != nil {
		return newExchangeError(ErrApiKeyNotSet, &internal.BybitError{Message: "Bybit API key is not set: " + err.Error()})
	}

	expires := (time.Now().Unix() + 1) * 1000
	signature, err := bw.generateSignature(expires)
	if err != nil {
		return err
	}

	authRequest := map[string]interface{}{
		"op":   "auth",
		"args": []interface{}{apiKey, expires, signature},
	}

	var authResp BybitWebsocketResponse
	if err := bw.SendAndReceive(authRequest, &authResp); err != nil {
		return fmt.Errorf("failed to send authentication request: %w", err)
	}

	if authResp.RetMsg != "OK" && authResp.RetCode != 0 {
		return fmt.Errorf("authentication failed with error code %d: %s", authResp.RetCode, authResp.RetMsg)
	}

	return nil
}

// ResetOnDisconnect implements WebsocketConnector — nothing to clean up,
// resetHeartbeat is handled by the base Websocket
func (bw *BybitWebsocket) ResetOnDisconnect() error {
	return nil
}

// Ping implements WebsocketConnector — sends Bybit-specific ping and validates pong
func (bw *BybitWebsocket) Ping() error {
	pingRequest := map[string]interface{}{"op": "ping"}
	var pongResp BybitWebsocketResponse
	if err := bw.SendAndReceive(pingRequest, &pongResp); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	if pongResp.Op != "pong" {
		return fmt.Errorf("unexpected heartbeat response: %+v", pongResp)
	}
	return nil
}

// header generates authentication headers for Bybit websocket API
func (bw *BybitWebsocket) header() map[string]string {
	return map[string]string{
		"X-BAPI-TIMESTAMP":   strconv.FormatInt(time.Now().UnixMilli(), 10),
		"X-BAPI-RECV-WINDOW": "5000",
	}
}

// PlaceMarketOrder places a market order on the websocket client
func (bw *BybitWebsocket) PlaceMarketOrder(
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	reduceOnly bool,
	closeOnTrigger bool,
) error {
	params := map[string]interface{}{
		"category":       "linear",
		"symbol":         symbol,
		"orderType":      "Market",
		"side":           side,
		"qty":            size,
		"reduceOnly":     reduceOnly,
		"closeOnTrigger": closeOnTrigger,
	}

	if takeProfit != nil || stopLoss != nil {
		params["tpslMode"] = "Full"
	}
	if takeProfit != nil {
		params["takeProfit"] = strconv.FormatFloat(*takeProfit, 'f', -1, 64)
		params["tpTriggerBy"] = "LastPrice"
	}
	if stopLoss != nil {
		params["stopLoss"] = strconv.FormatFloat(*stopLoss, 'f', -1, 64)
		params["slTriggerBy"] = "LastPrice"
	}

	request := map[string]interface{}{
		"reqId":  fmt.Sprintf("place_order_%d", time.Now().UnixMilli()),
		"op":     "order.create",
		"args":   []interface{}{params},
		"header": bw.header(),
	}

	var response BybitWebsocketResponse
	err := bw.SendAndReceive(request, &response)
	if err != nil {
		return fmt.Errorf("failed to send place order request: %w", err)
	}

	// Check if operation was successful
	// RetMsg "OK" indicates success even if Success field is false
	if response.RetMsg != "OK" && response.RetCode != 0 {
		return fmt.Errorf("place order failed with error code %d: %s", response.RetCode, response.RetMsg)
	}

	return nil
}

// SetPositionTpsl sets take profit and stop loss by amending two separate orders.
// Each leg is amended only when its order identifier is provided; at least one is required.
func (bw *BybitWebsocket) SetPositionTpsl(
	symbol string,
	takeProfitOrderId *string,
	stopLossOrderId *string,
	takeProfit float64,
	stopLoss float64,
) error {
	// Validate that at least one order identifier is provided
	if takeProfitOrderId == nil && stopLossOrderId == nil {
		return fmt.Errorf("takeProfitOrderId or stopLossOrderId must be provided")
	}

	if takeProfitOrderId != nil {
		// Amend the take profit order
		tpParams := map[string]interface{}{
			"category":     "linear",
			"symbol":       symbol,
			"orderId":      *takeProfitOrderId,
			"triggerBy":    "LastPrice",
			"triggerPrice": strconv.FormatFloat(takeProfit, 'f', -1, 64),
		}

		tpRequest := map[string]interface{}{
			"reqId":  fmt.Sprintf("amend_tp_order_%d", time.Now().UnixMilli()),
			"op":     "order.amend",
			"args":   []interface{}{tpParams},
			"header": bw.header(),
		}

		var tpResponse BybitWebsocketResponse
		err := bw.SendAndReceive(tpRequest, &tpResponse)
		if err != nil {
			return fmt.Errorf("failed to send amend TP order request: %w", err)
		}

		// Check if TP amend was successful
		if tpResponse.RetMsg != "OK" && tpResponse.RetCode != 0 {
			return fmt.Errorf("amend TP order failed with error code %d: %s", tpResponse.RetCode, tpResponse.RetMsg)
		}
	}

	if stopLossOrderId == nil {
		return nil
	}

	// Amend the stop loss order
	slParams := map[string]interface{}{
		"category":     "linear",
		"symbol":       symbol,
		"orderId":      *stopLossOrderId,
		"triggerBy":    "LastPrice",
		"triggerPrice": strconv.FormatFloat(stopLoss, 'f', -1, 64),
	}

	slRequest := map[string]interface{}{
		"reqId":  fmt.Sprintf("amend_sl_order_%d", time.Now().UnixMilli()),
		"op":     "order.amend",
		"args":   []interface{}{slParams},
		"header": bw.header(),
	}

	var slResponse BybitWebsocketResponse
	err := bw.SendAndReceive(slRequest, &slResponse)
	if err != nil {
		return fmt.Errorf("failed to send amend SL order request: %w", err)
	}

	// Check if SL amend was successful
	if slResponse.RetMsg != "OK" && slResponse.RetCode != 0 {
		return fmt.Errorf("amend SL order failed with error code %d: %s", slResponse.RetCode, slResponse.RetMsg)
	}

	return nil
}
