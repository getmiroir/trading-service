package clients

import (
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

// Demo endpoints are fixed constants selected by is_demo at runtime.
const (
	bybitDemoURL             = "https://api-demo.bybit.com/v5"
	bybitTestnetURL          = "https://api-testnet.bybit.com/v5"
	bybitTestnetWebsocketURL = "wss://stream-testnet.bybit.com/v5/trade"
)

// Production endpoints are overridable at build time via -ldflags.
var (
	BybitBaseURL                = "https://api.bybit.com/v5"
	BybitWebsocketURL           = "wss://stream.bybit.com/v5/trade"
	BybitTurkeyWebsocketURL     = "wss://stream.bybit.tr/v5/trade"
	BybitKazakhstanWebsocketURL = "wss://stream.bybit.kz/v5/trade"
	BybitProxyKey               = ""
	BybitUseWebsocket           = "false"
)

func resolveBybitBaseURL(isDemo bool, isTestnet bool) string {
	if isTestnet {
		return bybitTestnetURL
	}
	if isDemo {
		return bybitDemoURL
	}
	return BybitBaseURL
}

func resolveBybitWebsocketURL(isDemo bool, isTestnet bool, region subsettings.BybitRegion) string {
	if isTestnet {
		return bybitTestnetWebsocketURL
	}
	switch region {
	case subsettings.BybitRegionTurkey:
		return BybitTurkeyWebsocketURL
	case subsettings.BybitRegionKazakhstan:
		return BybitKazakhstanWebsocketURL
	default:
		return BybitWebsocketURL
	}
}

// Websockets are unavailable on the demo endpoint; testnet and prod support them.
func BybitWebsocketsEnabled(isDemo bool, isTestnet bool) bool {
	if isTestnet {
		return true
	}
	if isDemo {
		return false
	}
	return true
}

func resolveBybitProxyKey() *string {
	if BybitProxyKey == "" {
		return nil
	}
	return &BybitProxyKey
}
