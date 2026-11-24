package subsettings

import (
	"go.yaml.in/yaml/v4"
)

// Logging controls the lumberjack rolling-file logger.
// MaxAgeDays uses whole days (lumberjack constraint); sub-day retention is
// approximated by setting a low MaxBackups in the config.
// The log directory is not configurable here; logs always live under the
// application data directory (see platform.DataDir).
type Logging struct {
	MaxSizeMB  int  `yaml:"max_size_mb"`
	MaxBackups int  `yaml:"max_backups"`
	MaxAgeDays int  `yaml:"max_age_days"`
	Compress   bool `yaml:"compress"`
	LocalTime  bool `yaml:"local_time"`
}

func DefaultLogging() Logging {
	return Logging{
		MaxSizeMB:  50,
		MaxBackups: 0, // keep all rotated files; MaxAgeDays cleans up
		MaxAgeDays: 1,
		Compress:   false,
		LocalTime:  false,
	}
}

func (l *Logging) UnmarshalYAML(node *yaml.Node) error {
	*l = DefaultLogging()
	type selfAlias Logging
	return node.Decode((*selfAlias)(l))
}
