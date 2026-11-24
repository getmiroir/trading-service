package secrets

import (
	"fmt"
	"os"
	"strings"

	"github.com/getmiroir/trading-service/internal/keyring"
	"go.yaml.in/yaml/v4"
)

type YamlSecret struct {
	Path  *string   `yaml:"path"`
	Entry *[]string `yaml:"entry"`
}

func (s *YamlSecret) Load() (string, error) {
	if s.Path == nil || *s.Path == "" {
		return "", fmt.Errorf("yaml secret: path is required")
	}
	if s.Entry == nil || len(*s.Entry) == 0 {
		return "", fmt.Errorf("yaml secret: entry path is required")
	}

	data, err := os.ReadFile(*s.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read yaml secret file %s: %w", *s.Path, err)
	}

	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parsing %s: %w", *s.Path, err)
	}

	value, err := traverseYaml(doc, *s.Entry)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(value, "\n\r"), nil
}

func traverseYaml(doc any, keys []string) (string, error) {
	for i, key := range keys {
		node, ok := doc.(map[string]any)
		if !ok {
			return "", fmt.Errorf("yaml node at path %q is %T, expected map", strings.Join(keys[:i], "."), doc)
		}
		child, ok := node[key]
		if !ok {
			return "", fmt.Errorf("key %q not found in path %q", key, strings.Join(keys[:i+1], "."))
		}
		doc = child
	}

	stringValue, ok := doc.(string)
	if !ok {
		return "", fmt.Errorf("value at path %q is %T, expected string", strings.Join(keys, "."), doc)
	}
	return stringValue, nil
}

type Secret struct {
	FromKeyring *string     `yaml:"from_keyring"`
	FromEnvVar  *string     `yaml:"from_env_var"`
	FromYaml    *YamlSecret `yaml:"from_yaml"`
	FromFile    *string     `yaml:"from_file"`
	Raw         *string     `yaml:"raw"`
}

func (s *Secret) Value() (string, error) {
	if s.FromKeyring != nil {
		return keyring.Get(*s.FromKeyring)
	}
	if s.FromEnvVar != nil {
		valueInEnv, ok := os.LookupEnv(*s.FromEnvVar)
		if ok {
			return valueInEnv, nil
		} else {
			return "", fmt.Errorf("environment variable %s not found", *s.FromEnvVar)
		}
	}
	if s.FromYaml != nil {
		return s.FromYaml.Load()
	}
	if s.FromFile != nil {
		data, err := os.ReadFile(*s.FromFile)
		if err != nil {
			return "", fmt.Errorf("failed to read secret from file %s: %w", *s.FromFile, err)
		}
		return strings.TrimRight(string(data), "\n\r"), nil
	}
	if s.Raw != nil {
		return *s.Raw, nil
	}
	return "", fmt.Errorf("a source for the secret must be provided: raw, keyring, env var, yaml, or file")
}
