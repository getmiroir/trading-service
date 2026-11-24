package subsettings

import (
	"time"

	"go.yaml.in/yaml/v4"
)

type Websockets struct {
	PingInterval   time.Duration `yaml:"ping_interval"`
	ReconnectDelay time.Duration `yaml:"reconnect_delay"`
	Timeout        time.Duration `yaml:"timeout"`
}

func DefaultWebsockets() Websockets {
	return Websockets{
		PingInterval:   20 * time.Second,
		ReconnectDelay: 1 * time.Second,
		Timeout:        20 * time.Second,
	}
}

func (w *Websockets) UnmarshalYAML(node *yaml.Node) error {
	*w = DefaultWebsockets()
	type selfAlias Websockets
	if err := node.Decode((*selfAlias)(w)); err != nil {
		return err
	}
	if w.Timeout <= 0 {
		w.Timeout = DefaultWebsockets().Timeout
	}
	return nil
}
