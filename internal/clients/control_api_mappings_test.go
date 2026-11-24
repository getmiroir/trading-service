package clients

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
)

func TestBuildMappingsSkipsArchivedAccounts(t *testing.T) {
	exchangeType := "bybit"
	api := &ControlApi{
		accounts: map[string]subsettings.ExchangeAccount{
			"active": {
				ExchangeType: &exchangeType,
			},
			"archived": {
				ExchangeType: &exchangeType,
			},
			"disabled": {
				ExchangeType: &exchangeType,
			},
		},
		logger: slog.Default(),
	}

	addr := "0xabc"
	targets := CopyTargets{
		CopyEnabled: true,
		ByAccount: map[string]CopyTarget{
			"active": {
				Address: &addr,
				Enabled: true,
			},
			"disabled": {
				Address: &addr,
				Enabled: false,
			},
		},
		ArchivedAccounts: []string{"archived"},
	}

	mappings := api.buildMappings(targets, MappingStatusOK)

	got := make(map[string]CopyTarget, len(mappings))
	for _, m := range mappings {
		got[m.Name] = m.Target
	}

	if _, ok := got["archived"]; ok {
		t.Fatalf("archived account should not be synced")
	}
	if _, ok := got["active"]; !ok {
		t.Fatalf("active account should be synced")
	}
	if _, ok := got["disabled"]; !ok {
		t.Fatalf("disabled account should still sync toward flat")
	}
	if got["disabled"].Enabled {
		t.Fatalf("disabled mapping should have Enabled=false")
	}
	if got["active"].Address == nil || *got["active"].Address != addr {
		t.Fatalf("active mapping should keep its target address")
	}
}

func TestGetMappingsKeepsArchivedAccountsSkippedWhenTargetsFetchFails(t *testing.T) {
	exchangeType := "bybit"
	addr := "0xabc"

	var respond func(w http.ResponseWriter)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond(w)
	}))
	defer srv.Close()

	ok := func(archived ...string) func(w http.ResponseWriter) {
		return func(w http.ResponseWriter) {
			_ = json.NewEncoder(w).Encode(CopyTargets{
				CopyEnabled:      true,
				ByAccount:        map[string]CopyTarget{"active": {Address: &addr, Enabled: true}},
				ArchivedAccounts: archived,
			})
		}
	}
	failWith := func(header string) func(w http.ResponseWriter) {
		return func(w http.ResponseWriter) {
			if header != "" {
				w.Header().Set(header, "1")
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}

	token := "token"
	api := &ControlApi{
		client:      srv.Client(),
		url:         srv.URL,
		bearerToken: &token,
		accounts: map[string]subsettings.ExchangeAccount{
			"active":   {ExchangeType: &exchangeType},
			"archived": {ExchangeType: &exchangeType},
		},
		logger: slog.Default(),
	}

	mappingsByName := func() map[string]CopyMapping {
		t.Helper()
		mappings, err := api.GetMappings()
		if err != nil {
			t.Fatalf("GetMappings: %v", err)
		}
		byName := make(map[string]CopyMapping, len(mappings))
		for _, m := range mappings {
			byName[m.Name] = m
		}
		return byName
	}

	respond = ok("archived")
	if _, synced := mappingsByName()["archived"]; synced {
		t.Fatalf("archived account should not be synced")
	}

	for name, failure := range map[string]struct {
		header string
		status MappingStatus
	}{
		"unreachable":     {"", MappingStatusError},
		"version blocked": {UnsupportedVersionHeader, MappingStatusVersionBlocked},
		"emergency close": {EmergencyCloseHeader, MappingStatusEmergencyClose},
	} {
		respond = failWith(failure.header)
		got := mappingsByName()
		if _, synced := got["archived"]; synced {
			t.Fatalf("%s: archived account should stay skipped", name)
		}
		if got["active"].Status != failure.status {
			t.Fatalf("%s: active status = %q, want %q", name, got["active"].Status, failure.status)
		}
	}

	// A successful fetch replaces the remembered list, so an unarchived account
	// syncs again.
	respond = ok()
	if _, synced := mappingsByName()["archived"]; !synced {
		t.Fatalf("unarchived account should be synced again")
	}
	respond = failWith("")
	if _, synced := mappingsByName()["archived"]; !synced {
		t.Fatalf("unarchived account should not be skipped on a later failure")
	}
}
