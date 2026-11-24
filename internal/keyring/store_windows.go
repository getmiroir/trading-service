//go:build windows

package keyring

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

const (
	programDataDir    = `C:\ProgramData`
	programDataAppDir = "Miroir"
	secretsSubdir     = "secrets"
)

var secretsBaseDirOverride string

func setSecretsBaseDirForTests(path string) {
	secretsBaseDirOverride = path
}

func secretsBaseDir() string {
	if secretsBaseDirOverride != "" {
		return secretsBaseDirOverride
	}
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = programDataDir
	}
	return filepath.Join(programData, programDataAppDir, secretsSubdir)
}

func secretPath(key string) string {
	safeKey := url.QueryEscape(key)
	return filepath.Join(secretsBaseDir(), safeKey+".dat")
}

func ensureSecretsDir() error {
	baseDir := secretsBaseDir()
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}
	if err := applySecretsDACL(baseDir); err != nil {
		return fmt.Errorf("applying ACL to secrets directory: %w", err)
	}
	return nil
}

func readSecretBlob(key string) ([]byte, error) {
	path := secretPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading key %q: %w", key, err)
	}
	return data, nil
}

func writeSecretBlob(key string, data []byte) error {
	if err := ensureSecretsDir(); err != nil {
		return err
	}

	path := secretPath(key)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("writing temp file for key %q: %w", key, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file for key %q: %w", key, err)
	}
	if err := applySecretsDACL(path); err != nil {
		return fmt.Errorf("applying ACL to key %q: %w", key, err)
	}
	return nil
}

func deleteSecretBlob(key string) error {
	path := secretPath(key)
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("deleting key %q: %w", key, err)
	}
	return nil
}

func applySecretsDACL(path string) error {
	token := windows.GetCurrentProcessToken()
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("getting process token user: %w", err)
	}

	userSID := tokenUser.User.Sid.String()
	sddl := "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;" + userSID + ")"

	securityDescriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return fmt.Errorf("building security descriptor: %w", err)
	}
	dacl, _, err := securityDescriptor.DACL()
	if err != nil {
		return fmt.Errorf("reading DACL from security descriptor: %w", err)
	}

	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return fmt.Errorf("invalid empty path for ACL")
	}

	err = windows.SetNamedSecurityInfo(
		normalizedPath,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
	if err != nil {
		return fmt.Errorf("setting security info: %w", err)
	}
	return nil
}
