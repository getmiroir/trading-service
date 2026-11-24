package subsettings

import "go.yaml.in/yaml/v4"

type Source struct {
	Hyperliquid HyperliquidSource `yaml:"hyperliquid"`
}

func DefaultSource() Source {
	return Source{
		Hyperliquid: DefaultHyperliquidSource(),
	}
}

func (s *Source) UnmarshalYAML(node *yaml.Node) error {
	*s = DefaultSource()
	type selfAlias Source
	return node.Decode((*selfAlias)(s))
}
