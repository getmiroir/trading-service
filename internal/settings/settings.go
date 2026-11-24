package settings

import (
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"go.yaml.in/yaml/v4"
)

// Copy trading settings
type CopyTradingSettings struct {
	Exchange   subsettings.Exchange   `yaml:"exchange"`
	Source     subsettings.Source     `yaml:"source"`
	Auth       subsettings.Auth       `yaml:"auth"`
	ControlApi subsettings.ControlApi `yaml:"control_api"`
	Service    subsettings.Service    `yaml:"service"`
	Metrics    subsettings.Metrics    `yaml:"metrics"`
	Logging    subsettings.Logging    `yaml:"logging"`
	Features   subsettings.Feature    `yaml:"features"`
	Update     subsettings.Update     `yaml:"update"`
}

func DefaultCopyTradingSettings() CopyTradingSettings {
	return CopyTradingSettings{
		Exchange:   subsettings.DefaultExchange(),
		Source:     subsettings.DefaultSource(),
		Auth:       subsettings.DefaultAuth(),
		ControlApi: subsettings.DefaultControlApi(),
		Service:    subsettings.DefaultService(),
		Metrics:    subsettings.DefaultMetrics(),
		Logging:    subsettings.DefaultLogging(),
		Features:   subsettings.DefaultFeature(),
		Update:     subsettings.DefaultUpdate(),
	}
}

func (c *CopyTradingSettings) UnmarshalYAML(node *yaml.Node) error {
	*c = DefaultCopyTradingSettings()
	type selfAlias CopyTradingSettings
	return node.Decode((*selfAlias)(c))
}

type Settings struct {
	Version     int                 `yaml:"version"`
	CopyTrading CopyTradingSettings `yaml:"copy_trading"`
}

func DefaultSettings() Settings {
	return Settings{
		Version:     currentSettingsVersion,
		CopyTrading: DefaultCopyTradingSettings(),
	}
}

func (s *Settings) UnmarshalYAML(node *yaml.Node) error {
	*s = DefaultSettings()
	type selfAlias Settings
	return node.Decode((*selfAlias)(s))
}
