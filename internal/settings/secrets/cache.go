package secrets

import (
	"strings"
	"sync"
)

// The cache lifetime matches the client that owns it: a fresh SecretCache is
// created each time a client is constructed, so restarting the service always
// picks up updated keychain entries.
type SecretCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewSecretCache() *SecretCache {
	return &SecretCache{store: make(map[string]string)}
}

func (c *SecretCache) Get(s Secret) (string, error) {
	if s.Raw != nil {
		return *s.Raw, nil
	}

	key := cacheKey(s)
	if key == "" {
		// No source configured — delegate to Value() for its standard error.
		return s.Value()
	}

	c.mu.RLock()
	if v, ok := c.store[key]; ok {
		c.mu.RUnlock()
		return v, nil
	}
	c.mu.RUnlock()

	// Cache miss: resolve and store under a write lock.
	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock.
	if v, ok := c.store[key]; ok {
		return v, nil
	}
	v, err := s.Value()
	if err != nil {
		return "", err
	}
	c.store[key] = v
	return v, nil
}

// cacheKey derives a stable, non-secret string that uniquely identifies the
// source of a Secret. The secret value itself is never used as a key.
func cacheKey(s Secret) string {
	if s.FromKeyring != nil {
		return "kr:" + *s.FromKeyring
	}
	if s.FromEnvVar != nil {
		return "env:" + *s.FromEnvVar
	}
	if s.FromFile != nil {
		return "file:" + *s.FromFile
	}
	if s.FromYaml != nil && s.FromYaml.Path != nil && s.FromYaml.Entry != nil {
		return "yaml:" + *s.FromYaml.Path + ":" + strings.Join(*s.FromYaml.Entry, ".")
	}
	return ""
}
