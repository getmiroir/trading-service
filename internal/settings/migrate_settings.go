package settings

import (
	"fmt"
	"strconv"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"go.yaml.in/yaml/v4"
)

// currentSettingsVersion is the schema version written by this build.
// settingsMigrations[i] upgrades from version i to i+1; keep len equal to this.
const currentSettingsVersion = 1

// settingsMigrations[i] upgrades a settings document from version i to i+1.
var settingsMigrations = []func(*yaml.Node) error{
	migrateSettingsToV1,
}

// migrateSettingsData runs schema migrations on raw settings YAML.
// Missing version is treated as 0. Returns migrated=true when any step ran.
func migrateSettingsData(data []byte) (migratedData []byte, migrated bool, err error) {
	root, doc, err := parseSettingsYAMLDoc(data)
	if err != nil {
		return nil, false, err
	}

	version, err := readSettingsVersion(doc)
	if err != nil {
		return nil, false, err
	}
	if version > currentSettingsVersion {
		return nil, false, fmt.Errorf("settings version %d is newer than supported version %d", version, currentSettingsVersion)
	}
	if version == currentSettingsVersion {
		return data, false, nil
	}
	if currentSettingsVersion != len(settingsMigrations) {
		return nil, false, fmt.Errorf("settingsMigrations length %d != currentSettingsVersion %d", len(settingsMigrations), currentSettingsVersion)
	}

	for v := version; v < currentSettingsVersion; v++ {
		if err := settingsMigrations[v](doc); err != nil {
			return nil, false, fmt.Errorf("migrating settings from version %d to %d: %w", v, v+1, err)
		}
	}
	setSettingsVersion(doc, currentSettingsVersion)

	out, err := yaml.Marshal(root)
	if err != nil {
		return nil, false, fmt.Errorf("marshaling migrated settings: %w", err)
	}
	return out, true, nil
}

func parseSettingsYAMLDoc(data []byte) (root, doc *yaml.Node, err error) {
	root = &yaml.Node{}
	if err := yaml.Unmarshal(data, root); err != nil {
		return nil, nil, err
	}
	switch {
	case root.Kind == yaml.DocumentNode && len(root.Content) > 0:
		return root, root.Content[0], nil
	case root.Kind == yaml.MappingNode:
		return root, root, nil
	default:
		return nil, nil, fmt.Errorf("settings root must be a mapping")
	}
}

func readSettingsVersion(doc *yaml.Node) (int, error) {
	n := subsettings.YAMLMapChild(doc, "version")
	if n == nil {
		return 0, nil
	}
	switch n.Kind {
	case yaml.ScalarNode:
		v, err := strconv.Atoi(n.Value)
		if err != nil {
			return 0, fmt.Errorf("parsing settings version: %w", err)
		}
		return v, nil
	default:
		return 0, fmt.Errorf("settings version must be a scalar")
	}
}

func setSettingsVersion(doc *yaml.Node, version int) {
	yamlSetMapChild(doc, "version", &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: strconv.Itoa(version),
	})
}

// migrateSettingsToV1 renames pre-v0.28 Hyperliquid keys:
// - copy_trading.hyperliquid → copy_trading.source.hyperliquid
// - exchange.hyperliquid_exchange → exchange.hyperliquid
func migrateSettingsToV1(doc *yaml.Node) error {
	copyTrading := subsettings.YAMLMapChild(doc, "copy_trading")
	if copyTrading == nil {
		return nil
	}

	if legacy := subsettings.YAMLMapChild(copyTrading, "hyperliquid"); legacy != nil {
		source := subsettings.YAMLMapChild(copyTrading, "source")
		if source == nil {
			source = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			yamlSetMapChild(copyTrading, "source", source)
		}
		if subsettings.YAMLMapChild(source, "hyperliquid") == nil {
			yamlSetMapChild(source, "hyperliquid", legacy)
		}
		yamlDeleteMapChild(copyTrading, "hyperliquid")
	}

	exchange := subsettings.YAMLMapChild(copyTrading, "exchange")
	if exchange == nil {
		return nil
	}
	if legacy := subsettings.YAMLMapChild(exchange, "hyperliquid_exchange"); legacy != nil {
		if subsettings.YAMLMapChild(exchange, "hyperliquid") == nil {
			yamlSetMapChild(exchange, "hyperliquid", legacy)
		}
		yamlDeleteMapChild(exchange, "hyperliquid_exchange")
	}
	return nil
}

func yamlSetMapChild(node *yaml.Node, key string, value *yaml.Node) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = value
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func yamlDeleteMapChild(node *yaml.Node, key string) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}
