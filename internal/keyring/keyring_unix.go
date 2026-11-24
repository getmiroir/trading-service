//go:build !windows

package keyring

import zalandokeyring "github.com/zalando/go-keyring"

// Get retrieves a secret from the system keyring.
func Get(key string) (string, error) {
	return zalandokeyring.Get(ServiceName, key)
}

// Set stores a secret in the system keyring.
func Set(key, value string) error {
	return zalandokeyring.Set(ServiceName, key, value)
}

// Delete removes a secret from the system keyring.
func Delete(key string) error {
	return zalandokeyring.Delete(ServiceName, key)
}
