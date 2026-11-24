package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getmiroir/trading-service/internal/platform"
)

func TestLoadSettingsMigratesV0Aliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	legacy := []byte(`
copy_trading:
  exchange:
    hyperliquid_exchange:
      accounts:
        hyperliquid-0xabc:
          master_wallet_address: "0xabc"
          private_key:
            from_keyring: "hyperliquid-0xabc-key"
  hyperliquid:
    url: "https://custom.example/info"
`)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatalf("writing legacy settings: %v", err)
	}

	t.Setenv(platform.SettingsFileEnvVar, path)
	t.Setenv(platform.DataDirEnvVar, dir)

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if loaded.Version != currentSettingsVersion {
		t.Fatalf("Version = %d, want %d", loaded.Version, currentSettingsVersion)
	}
	if got := loaded.CopyTrading.Source.Hyperliquid.URL; got != "https://custom.example/info" {
		t.Fatalf("source URL = %q, want custom URL", got)
	}
	acc, ok := loaded.CopyTrading.Exchange.Hyperliquid.Accounts["hyperliquid-0xabc"]
	if !ok {
		t.Fatal("expected account under exchange.hyperliquid")
	}
	if acc.MasterWalletAddress != "0xabc" {
		t.Fatalf("master wallet = %q, want 0xabc", acc.MasterWalletAddress)
	}

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading rewritten settings: %v", err)
	}
	body := string(rewritten)
	if strings.Contains(body, "hyperliquid_exchange") {
		t.Fatalf("rewritten settings still contain hyperliquid_exchange:\n%s", body)
	}
	if !strings.Contains(body, "version:") {
		t.Fatalf("rewritten settings missing version:\n%s", body)
	}
	if !strings.Contains(body, "source:") {
		t.Fatalf("rewritten settings missing source:\n%s", body)
	}
}

func TestLoadSettingsLeavesCurrentVersionAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	current := []byte(`
version: 1
copy_trading:
  exchange:
    hyperliquid:
      accounts:
        hyperliquid-0xabc:
          master_wallet_address: "0xabc"
          private_key:
            from_keyring: "hyperliquid-0xabc-key"
  source:
    hyperliquid:
      url: "https://custom.example/info"
`)
	if err := os.WriteFile(path, current, 0600); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	t.Setenv(platform.SettingsFileEnvVar, path)
	t.Setenv(platform.DataDirEnvVar, dir)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	if _, err := LoadSettings(); err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings after load: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("settings file changed for current version:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestLoadSettingsRejectsFutureVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	future := []byte("version: 99\ncopy_trading: {}\n")
	if err := os.WriteFile(path, future, 0600); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	t.Setenv(platform.SettingsFileEnvVar, path)
	t.Setenv(platform.DataDirEnvVar, dir)

	if _, err := LoadSettings(); err == nil {
		t.Fatal("expected error for future settings version")
	}
}

func TestMigrateSettingsToV1(t *testing.T) {
	in := []byte(`
copy_trading:
  exchange:
    hyperliquid_exchange:
      accounts: {}
  hyperliquid:
    url: https://custom.example/info
`)
	out, migrated, err := migrateSettingsData(in)
	if err != nil {
		t.Fatalf("migrateSettingsData: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration to run")
	}
	body := string(out)
	if strings.Contains(body, "hyperliquid_exchange") {
		t.Fatalf("migrated YAML still has hyperliquid_exchange:\n%s", body)
	}
	if !strings.Contains(body, "source:") {
		t.Fatalf("migrated YAML missing source:\n%s", body)
	}
	if !strings.Contains(body, "version: 1") {
		t.Fatalf("migrated YAML missing version 1:\n%s", body)
	}

	again, migrated, err := migrateSettingsData(out)
	if err != nil {
		t.Fatalf("second migrateSettingsData: %v", err)
	}
	if migrated {
		t.Fatal("expected no second migration")
	}
	if string(again) != string(out) {
		t.Fatalf("current-version YAML changed on migrate")
	}
}
