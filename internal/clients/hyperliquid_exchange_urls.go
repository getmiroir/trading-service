package clients

// Production endpoint is overridable at build time via -ldflags.
// Testnet endpoint is a fixed constant selected by is_testnet at runtime.

const (
	hyperliquidExchangeTestnetURL = "https://api.hyperliquid-testnet.xyz"
)

var (
	HyperliquidExchangeMainnetURL = "https://api.hyperliquid.xyz"
)

func resolveHyperliquidExchangeURL(isTestnet bool) string {
	if isTestnet {
		return hyperliquidExchangeTestnetURL
	}
	return HyperliquidExchangeMainnetURL
}

func resolveHyperliquidExchangeWebsocketURL(isTestnet bool) string {
	if isTestnet {
		return "wss://api.hyperliquid-testnet.xyz/ws"
	}
	return "wss://api.hyperliquid.xyz/ws"
}
