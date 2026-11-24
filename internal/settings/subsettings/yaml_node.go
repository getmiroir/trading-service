package subsettings

import "go.yaml.in/yaml/v4"

// YAMLMapChild returns the value node for key in a YAML mapping node, or nil.
func YAMLMapChild(node *yaml.Node, key string) *yaml.Node {
	return yamlMapChild(node, key)
}

func yamlMapChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
