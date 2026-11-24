package subsettings

import "go.yaml.in/yaml/v4"

type HyperliquidSource struct {
	URL         string     `yaml:"url"`
	BearerToken *string    `yaml:"bearer_token"`
	HttpClient  HttpClient `yaml:"http_client"`
}

func DefaultHyperliquidSource() HyperliquidSource {
	return HyperliquidSource{
		URL:         "https://api.hyperliquid.xyz/info",
		BearerToken: nil,
		HttpClient:  DefaultHttpClient(),
	}
}

func (h *HyperliquidSource) UnmarshalYAML(node *yaml.Node) error {
	*h = DefaultHyperliquidSource()
	type selfAlias HyperliquidSource
	return node.Decode((*selfAlias)(h))
}
