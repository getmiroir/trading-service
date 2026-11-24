package clients

// Production endpoints are overridable at build time via -ldflags.
// Demo endpoints are fixed constants selected by is_demo at runtime.

const (
	binanceFuturesDemoURL          = "https://demo-fapi.binance.com/fapi"
	binanceFuturesWebsocketDemoURL = "wss://testnet.binancefuture.com/ws-fapi/v1"
	binanceSpotDemoURL             = "https://testnet.binance.vision/api"
	binanceSpotAccountURL          = "https://api.binance.com/sapi"
)

var (
	BinanceFuturesBaseURL      = "https://fapi.binance.com/fapi"
	BinanceFuturesWebsocketURL = "wss://ws-fapi.binance.com/ws-fapi/v1"
	BinanceSpotBaseURL         = "https://api.binance.com/api"
	BinanceSpotAccountBaseURL  = "https://api.binance.com/sapi"
	BinanceProxyKey            = ""
	BinanceUseWebsocket        = "false"
)

func resolveBinanceFuturesBaseURL(isDemo bool) string {
	if isDemo {
		return binanceFuturesDemoURL
	}
	return BinanceFuturesBaseURL
}

func resolveBinanceFuturesWebsocketURL(isDemo bool) string {
	if isDemo {
		return binanceFuturesWebsocketDemoURL
	}
	return BinanceFuturesWebsocketURL
}

func resolveBinanceSpotBaseURL(isDemo bool) string {
	if isDemo {
		return binanceSpotDemoURL
	}
	return BinanceSpotBaseURL
}

func resolveBinanceSpotAccountBaseURL(isDemo bool) string {
	// Spot account API has no separate demo endpoint; prod URL is used for both.
	if isDemo {
		return binanceSpotAccountURL
	}
	return BinanceSpotAccountBaseURL
}

func resolveBinanceProxyKey() *string {
	if BinanceProxyKey == "" {
		return nil
	}
	return &BinanceProxyKey
}
