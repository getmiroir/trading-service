package subsettings

import "go.yaml.in/yaml/v4"

// Metrics settings
type Metrics struct {
	BufferSize           int     `yaml:"buffer_size"`
	BatchSize            int     `yaml:"batch_size"`
	FlushIntervalSeconds float64 `yaml:"flush_interval_seconds"`
	MaxRetryDelaySeconds float64 `yaml:"max_retry_delay_seconds"`
}

func DefaultMetrics() Metrics {
	return Metrics{
		BufferSize:           100,
		BatchSize:            10,
		FlushIntervalSeconds: 5.0,
		MaxRetryDelaySeconds: 60.0,
	}
}
func (m *Metrics) UnmarshalYAML(node *yaml.Node) error {
	*m = DefaultMetrics()
	type selfAlias Metrics
	return node.Decode((*selfAlias)(m))
}
