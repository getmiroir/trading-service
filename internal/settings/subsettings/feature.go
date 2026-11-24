package subsettings

import "go.yaml.in/yaml/v4"

// ClientType identifies a supported exchange client. The same values are used
// for the EnabledClients setting and for the GUI's per-client availability
// checks, so the settings and the GUI never drift apart.
type ClientType string

const (
	ClientBybit       ClientType = "bybit"
	ClientBinance     ClientType = "binance"
	ClientHyperliquid ClientType = "hyperliquid"
)

type Feature struct {
	// EnabledClients lists the exchange clients exposed in the GUI. Clients that
	// are not listed are hidden from the account picker and cannot be added from
	// the GUI. It does not affect syncing of already-configured accounts.
	EnabledClients []ClientType `yaml:"enabled_clients"`
}

func DefaultFeature() Feature {
	return Feature{
		EnabledClients: []ClientType{ClientBybit, ClientBinance, ClientHyperliquid},
	}
}

// IsClientEnabled reports whether the given client is enabled in the GUI.
func (f Feature) IsClientEnabled(c ClientType) bool {
	for _, ec := range f.EnabledClients {
		if ec == c {
			return true
		}
	}
	return false
}

func (f *Feature) UnmarshalYAML(node *yaml.Node) error {
	*f = DefaultFeature()
	type selfAlias Feature
	return node.Decode((*selfAlias)(f))
}
