package subsettings

import "go.yaml.in/yaml/v4"

// Retry settings
type RetrySettings struct {
	InitialRetryDelay float64 `yaml:"initial_retry_delay"`
	RetryDelayFactor  float64 `yaml:"retry_delay_factor"`
	RetryDelayJitter  float64 `yaml:"retry_delay_jitter"`
	MaxRetryDelay     float64 `yaml:"max_retry_delay"`
}

func DefaultRetrySettings() RetrySettings {
	return RetrySettings{
		InitialRetryDelay: 1.0,
		RetryDelayFactor:  1.5,
		RetryDelayJitter:  0.5,
		MaxRetryDelay:     2.0,
	}
}

func (r *RetrySettings) UnmarshalYAML(node *yaml.Node) error {
	*r = DefaultRetrySettings()
	type selfAlias RetrySettings
	return node.Decode((*selfAlias)(r))
}

// Single rate limit settings
type RateLimit struct {
	Requests int `yaml:"requests"`
	Seconds  int `yaml:"seconds"`
}

func DefaultRateLimit() RateLimit {
	return RateLimit{
		Requests: 1,
		Seconds:  1,
	}
}
func (r *RateLimit) UnmarshalYAML(node *yaml.Node) error {
	*r = DefaultRateLimit()
	type selfAlias RateLimit
	return node.Decode((*selfAlias)(r))
}

// Rate limiter settings
type RateLimiterSettings struct {
	Limits          []RateLimit           `yaml:"limits"`
	MaxRetries      int                   `yaml:"max_retries"`
	DefaultRetry    RetrySettings         `yaml:"default_retry"`
	RetryRetcode    map[int]RetrySettings `yaml:"retry_retcode"`
	RetryStatuscode map[int]RetrySettings `yaml:"retry_statuscode,omitempty"`
}

func DefaultRateLimiterSettings() RateLimiterSettings {
	return RateLimiterSettings{
		Limits: []RateLimit{
			{Requests: 100, Seconds: 10},
		},
		MaxRetries:      3,
		DefaultRetry:    DefaultRetrySettings(),
		RetryRetcode:    map[int]RetrySettings{10006: DefaultRetrySettings()},
		RetryStatuscode: map[int]RetrySettings{418: DefaultRetrySettings(), 429: DefaultRetrySettings()},
	}
}
func (r *RateLimiterSettings) UnmarshalYAML(node *yaml.Node) error {
	*r = DefaultRateLimiterSettings()
	type selfAlias RateLimiterSettings
	return node.Decode((*selfAlias)(r))
}

// Http client settings
type HttpClient struct {
	PoolSize      int                 `yaml:"pool_size"`
	Timeout       float64             `yaml:"timeout"`
	MaxConcurrent int                 `yaml:"max_concurrent"`
	RateLimiter   RateLimiterSettings `yaml:"rate_limiter"`
	ExtraHeaders  map[string]string   `yaml:"extra_headers"`
}

func DefaultHttpClient() HttpClient {
	return HttpClient{
		PoolSize:      10,
		Timeout:       10.0,
		MaxConcurrent: 10,
		RateLimiter:   DefaultRateLimiterSettings(),
	}
}
func (h *HttpClient) UnmarshalYAML(node *yaml.Node) error {
	*h = DefaultHttpClient()
	type selfAlias HttpClient
	return node.Decode((*selfAlias)(h))
}
