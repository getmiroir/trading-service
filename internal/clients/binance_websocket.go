package clients

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getmiroir/trading-service/internal"
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// BinanceWebsocket represents the Binance websocket client
type BinanceWebsocket struct {
	*Websocket
	key    secrets.Secret
	secret secrets.Secret
	creds  *secrets.SecretCache
}

func NewBinanceWebsocket(
	account subsettings.BinanceAccount,
	pingInterval time.Duration,
	reconnectDelay time.Duration,
	timeout time.Duration,
	creds *secrets.SecretCache,
) *BinanceWebsocket {
	bw := &BinanceWebsocket{
		Websocket: NewWebsocket(
			resolveBinanceFuturesWebsocketURL(account.IsDemo),
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
func (bw *BinanceWebsocket) generateSignature(params map[string]string) (interface{}, error) {
	apiSecret, secretErr := bw.creds.Get(bw.secret)
	if secretErr != nil {
		return nil, newExchangeError(ErrApiKeyNotSet, &internal.BinanceError{Message: "Binance API secret is not set: " + secretErr.Error()})
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, params[k]))
	}
	queryString := strings.Join(parts, "&")

	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Authenticate authenticates the websocket connection
// Binance websocket is not authenticated, requests are signed one by one
// Binance websocket does not need an hearbeat because Binance takes care of sending
// ping messages to our service every 2 minutes.
// The golan websocket library will automatically respond to the ping messages with
// a pong message.
func (bw *BinanceWebsocket) Authenticate() error {
	return nil
}

// ResetOnDisconnect implements WebsocketConnector — nothing to clean up for Binance
func (bw *BinanceWebsocket) ResetOnDisconnect() error {
	return nil
}

// Ping implements WebsocketConnector — Binance manages its own pings,
// so we return nil and let Binance send ping frames to us
func (bw *BinanceWebsocket) Ping() error {
	return nil
}

// PlaceMarketOrder places a market order on the websocket client
func (bw *BinanceWebsocket) PlaceMarketOrder(
	symbol string,
	side string,
	size string,
	takeProfit *float64,
	stopLoss *float64,
	closePosition bool,
) error {
	binanceSide := "BUY"
	if side == "Sell" {
		binanceSide = "SELL"
	}
	closeSide := "SELL"
	if side == "Sell" {
		closeSide = "BUY"
	}

	// Get API key
	apiKey, keyErr := bw.creds.Get(bw.key)
	if keyErr != nil {
		return newExchangeError(ErrApiKeyNotSet, &internal.BinanceError{Message: "Binance API key is not set: " + keyErr.Error()})
	}

	// Build params for market order
	params := map[string]string{
		"apiKey":    apiKey,
		"symbol":    symbol,
		"side":      binanceSide,
		"type":      "MARKET",
		"quantity":  size,
		"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	// If closing position, set the close side
	if closePosition {
		params["side"] = closeSide
	}

	// Generate signature
	signature, sigErr := bw.generateSignature(params)
	if sigErr != nil {
		return sigErr
	}
	params["signature"] = signature.(string)

	// Build request
	request := map[string]interface{}{
		"id":     fmt.Sprintf("place_order_%d", time.Now().UnixMilli()),
		"method": "order.place",
		"params": params,
	}

	// Send request
	var response BinanceWebsocketResponse
	err := bw.SendAndReceive(request, &response)
	if err != nil {
		return fmt.Errorf("failed to send place order request: %w", err)
	}

	// Check if operation was successful
	if response.Status != 200 {
		if response.Error != nil {
			return &internal.BinanceError{
				Message: fmt.Sprintf("place order failed with error code %d: %s", response.Error.Code, response.Error.Msg),
			}
		}
		return &internal.BinanceError{
			Message: fmt.Sprintf("place order failed with status %d", response.Status),
		}
	}

	// do not place any TP/SL if closing position
	if closePosition {
		return nil
	}

	// Set take profit if provided
	if takeProfit != nil {
		tpParams := map[string]string{
			"apiKey":        apiKey,
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "TAKE_PROFIT_MARKET",
			"triggerPrice":  strconv.FormatFloat(*takeProfit, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
			"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		tpSignature, tpSigErr := bw.generateSignature(tpParams)
		if tpSigErr != nil {
			return tpSigErr
		}
		tpParams["signature"] = tpSignature.(string)

		tpRequest := map[string]interface{}{
			"id":     fmt.Sprintf("place_tp_%d", time.Now().UnixMilli()),
			"method": "algoOrder.place",
			"params": tpParams,
		}

		var tpResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(tpRequest, &tpResponse)
		if err != nil {
			return fmt.Errorf("failed to send take profit order request: %w", err)
		}

		if tpResponse.Status != 200 {
			if tpResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("take profit order failed with error code %d: %s", tpResponse.Error.Code, tpResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("take profit order failed with status %d", tpResponse.Status),
			}
		}
	}

	// Set stop loss if provided
	if stopLoss != nil {
		slParams := map[string]string{
			"apiKey":        apiKey,
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "STOP_MARKET",
			"triggerPrice":  strconv.FormatFloat(*stopLoss, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
			"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		slSignature, slSigErr := bw.generateSignature(slParams)
		if slSigErr != nil {
			return slSigErr
		}
		slParams["signature"] = slSignature.(string)

		slRequest := map[string]interface{}{
			"id":     fmt.Sprintf("place_sl_%d", time.Now().UnixMilli()),
			"method": "algoOrder.place",
			"params": slParams,
		}

		var slResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(slRequest, &slResponse)
		if err != nil {
			return fmt.Errorf("failed to send stop loss order request: %w", err)
		}

		if slResponse.Status != 200 {
			if slResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("stop loss order failed with error code %d: %s", slResponse.Error.Code, slResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("stop loss order failed with status %d", slResponse.Status),
			}
		}
	}

	return nil
}

// SetPositionTpsl sets take profit and stop loss for a position
func (bw *BinanceWebsocket) SetPositionTpsl(
	symbol string,
	positionSide string, // "Buy" or "Sell"
	takeProfit float64,
	stopLoss float64,
	takeProfitOrderId int64,
	stopLossOrderId int64,
) error {
	closeSide := "SELL"
	if positionSide == "Sell" {
		closeSide = "BUY"
	}

	// Get API key
	apiKey, keyErr := bw.creds.Get(bw.key)
	if keyErr != nil {
		return newExchangeError(ErrApiKeyNotSet, &internal.BinanceError{Message: "Binance API key is not set: " + keyErr.Error()})
	}

	// Cancel existing take profit order if provided
	if takeProfitOrderId > 0 {
		cancelTpParams := map[string]string{
			"apiKey":    apiKey,
			"algoId":    strconv.FormatInt(takeProfitOrderId, 10),
			"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		cancelTpSignature, cancelTpSigErr := bw.generateSignature(cancelTpParams)
		if cancelTpSigErr != nil {
			return cancelTpSigErr
		}
		cancelTpParams["signature"] = cancelTpSignature.(string)

		cancelTpRequest := map[string]interface{}{
			"id":     fmt.Sprintf("cancel_tp_%d", time.Now().UnixMilli()),
			"method": "algoOrder.cancel",
			"params": cancelTpParams,
		}

		var cancelTpResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(cancelTpRequest, &cancelTpResponse)
		if err != nil {
			return fmt.Errorf("failed to send cancel take profit order request: %w", err)
		}

		if cancelTpResponse.Status != 200 {
			if cancelTpResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("cancel take profit order failed with error code %d: %s", cancelTpResponse.Error.Code, cancelTpResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("cancel take profit order failed with status %d", cancelTpResponse.Status),
			}
		}
	}

	// Cancel existing stop loss order if provided
	if stopLossOrderId > 0 {
		cancelSlParams := map[string]string{
			"apiKey":    apiKey,
			"algoId":    strconv.FormatInt(stopLossOrderId, 10),
			"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		cancelSlSignature, cancelSlSigErr := bw.generateSignature(cancelSlParams)
		if cancelSlSigErr != nil {
			return cancelSlSigErr
		}
		cancelSlParams["signature"] = cancelSlSignature.(string)

		cancelSlRequest := map[string]interface{}{
			"id":     fmt.Sprintf("cancel_sl_%d", time.Now().UnixMilli()),
			"method": "algoOrder.cancel",
			"params": cancelSlParams,
		}

		var cancelSlResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(cancelSlRequest, &cancelSlResponse)
		if err != nil {
			return fmt.Errorf("failed to send cancel stop loss order request: %w", err)
		}

		if cancelSlResponse.Status != 200 {
			if cancelSlResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("cancel stop loss order failed with error code %d: %s", cancelSlResponse.Error.Code, cancelSlResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("cancel stop loss order failed with status %d", cancelSlResponse.Status),
			}
		}
	}

	// Set take profit if requested (0 means no take profit)
	if takeProfit > 0 {
		tpParams := map[string]string{
			"apiKey":        apiKey,
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "TAKE_PROFIT_MARKET",
			"triggerPrice":  strconv.FormatFloat(takeProfit, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
			"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		tpSignature, tpSigErr := bw.generateSignature(tpParams)
		if tpSigErr != nil {
			return tpSigErr
		}
		tpParams["signature"] = tpSignature.(string)

		tpRequest := map[string]interface{}{
			"id":     fmt.Sprintf("set_tp_%d", time.Now().UnixMilli()),
			"method": "algoOrder.place",
			"params": tpParams,
		}

		var tpResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(tpRequest, &tpResponse)
		if err != nil {
			return fmt.Errorf("failed to send take profit order request: %w", err)
		}

		if tpResponse.Status != 200 {
			if tpResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("take profit order failed with error code %d: %s", tpResponse.Error.Code, tpResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("take profit order failed with status %d", tpResponse.Status),
			}
		}
	}

	// Set stop loss if requested (0 means no stop loss)
	if stopLoss > 0 {
		slParams := map[string]string{
			"apiKey":        apiKey,
			"algoType":      "CONDITIONAL",
			"symbol":        symbol,
			"side":          closeSide,
			"type":          "STOP_MARKET",
			"triggerPrice":  strconv.FormatFloat(stopLoss, 'f', -1, 64),
			"workingType":   "MARK_PRICE",
			"closePosition": "true",
			"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		}

		slSignature, slSigErr := bw.generateSignature(slParams)
		if slSigErr != nil {
			return slSigErr
		}
		slParams["signature"] = slSignature.(string)

		slRequest := map[string]interface{}{
			"id":     fmt.Sprintf("set_sl_%d", time.Now().UnixMilli()),
			"method": "algoOrder.place",
			"params": slParams,
		}

		var slResponse BinanceWebsocketResponse
		err := bw.SendAndReceive(slRequest, &slResponse)
		if err != nil {
			return fmt.Errorf("failed to send stop loss order request: %w", err)
		}

		if slResponse.Status != 200 {
			if slResponse.Error != nil {
				return &internal.BinanceError{
					Message: fmt.Sprintf("stop loss order failed with error code %d: %s", slResponse.Error.Code, slResponse.Error.Msg),
				}
			}
			return &internal.BinanceError{
				Message: fmt.Sprintf("stop loss order failed with status %d", slResponse.Status),
			}
		}
	}

	return nil
}
