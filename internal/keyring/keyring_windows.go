//go:build windows

package keyring

import "fmt"

// Get retrieves a secret from the Windows DPAPI-backed store.
func Get(key string) (string, error) {
	cipher, err := readSecretBlob(key)
	if err != nil {
		return "", err
	}
	plain, err := dpapiDecrypt(cipher)
	if err != nil {
		return "", fmt.Errorf("decrypting key %q: %w", key, err)
	}
	return string(plain), nil
}

// Set stores a secret in the Windows DPAPI-backed store.
func Set(key, value string) error {
	cipher, err := dpapiEncrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("encrypting key %q: %w", key, err)
	}
	if err := writeSecretBlob(key, cipher); err != nil {
		return err
	}
	return nil
}

// Delete removes a secret from the Windows DPAPI-backed store.
func Delete(key string) error {
	return deleteSecretBlob(key)
}
