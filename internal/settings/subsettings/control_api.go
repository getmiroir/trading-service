package subsettings

import (
	"github.com/getmiroir/trading-service/internal/settings/secrets"
	"go.yaml.in/yaml/v4"
)

// Control API settings
type ControlApi struct {
	URL        string         `yaml:"url"`
	ApiKey     secrets.Secret `yaml:"api_key"`
	HttpClient HttpClient     `yaml:"http_client"`
}

func DefaultControlApi() ControlApi {
	return ControlApi{
		URL: "https://getmiroir.xyz/api",
		HttpClient: HttpClient{
			PoolSize:      10,
			Timeout:       60.0,
			MaxConcurrent: 10,
			RateLimiter:   DefaultRateLimiterSettings(),
		},
	}
}

func (c *ControlApi) UnmarshalYAML(node *yaml.Node) error {
	*c = DefaultControlApi()
	type selfAlias ControlApi
	return node.Decode((*selfAlias)(c))
}
