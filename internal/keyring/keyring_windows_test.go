//go:build windows

package keyring

import (
	"errors"
	"os"
	"testing"
)

func TestSetGetDeleteRoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	setSecretsBaseDirForTests(baseDir)
	t.Cleanup(func() {
		setSecretsBaseDirForTests("")
	})

	key := "unit-test-key"
	value := "secret-value"

	err := Set(key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != value {
		t.Fatalf("Get returned %q, want %q", got, value)
	}

	err = Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = Get(key)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestDeleteMissingIsIdempotent(t *testing.T) {
	baseDir := t.TempDir()
	setSecretsBaseDirForTests(baseDir)
	t.Cleanup(func() {
		setSecretsBaseDirForTests("")
	})

	err := Delete("missing-key")
	if err != nil {
		t.Fatalf("Delete(missing) failed: %v", err)
	}

	_, err = os.Stat(secretPath("missing-key"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be absent, got err=%v", err)
	}
}

func TestDPAPIRoundTrip(t *testing.T) {
	plain := []byte("dpapi-smoke-test")
	cipher, err := dpapiEncrypt(plain)
	if err != nil {
		t.Fatalf("dpapiEncrypt failed: %v", err)
	}
	if len(cipher) == 0 {
		t.Fatalf("dpapiEncrypt returned empty cipher")
	}
	out, err := dpapiDecrypt(cipher)
	if err != nil {
		t.Fatalf("dpapiDecrypt failed: %v", err)
	}
	if string(out) != string(plain) {
		t.Fatalf("dpapi roundtrip mismatch got=%q want=%q", string(out), string(plain))
	}
}
