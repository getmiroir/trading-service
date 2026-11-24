package clients

import (
	"errors"
	"testing"

	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"go.yaml.in/yaml/v4"
)

func TestSymbolMapperExplicitUnmappedOverride(t *testing.T) {
	api := &ControlApi{}
	service := subsettings.DefaultService()
	service.HyperToBybitOverrides = map[string]map[string]string{
		"USDT": {
			"PURR": "",
		},
	}
	mapper := NewSymbolMapper(api, service)

	_, err := mapper.HyperToBybitSymbol("USDT", "PURR")
	if !errors.Is(err, ErrSymbolNotExist) {
		t.Fatalf("expected ErrSymbolNotExist, got %v", err)
	}
}

func TestSymbolMapperOverrideWins(t *testing.T) {
	api := &ControlApi{}
	service := subsettings.DefaultService()
	service.HyperToBybitOverrides = map[string]map[string]string{
		"USDT": {
			"xyz:PURRDAT": "PURRUSDT",
		},
	}
	mapper := NewSymbolMapper(api, service)

	sym, err := mapper.HyperToBybitSymbol("USDT", "xyz:PURRDAT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sym != "PURRUSDT" {
		t.Fatalf("expected PURRUSDT, got %q", sym)
	}
}

func TestServiceYAMLNullOverrideBecomesEmptyString(t *testing.T) {
	const yamlBody = `
hyper_to_bybit_overrides:
  USDT:
    PURR: null
    xyz:PURRDAT: PURRUSDT
`
	var service subsettings.Service
	if err := yaml.Unmarshal([]byte(yamlBody), &service); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := service.HyperToBybitOverrides["USDT"]["PURR"]; got != "" {
		t.Fatalf("expected empty string for null override, got %q", got)
	}
	if got := service.HyperToBybitOverrides["USDT"]["xyz:PURRDAT"]; got != "PURRUSDT" {
		t.Fatalf("expected PURRUSDT, got %q", got)
	}
	// Confirm the empty override surfaces as ErrSymbolNotExist via SymbolMapper.
	mapper := NewSymbolMapper(&ControlApi{}, service)
	_, err := mapper.HyperToBybitSymbol("USDT", "PURR")
	if !errors.Is(err, ErrSymbolNotExist) {
		t.Fatalf("expected ErrSymbolNotExist after YAML null, got %v", err)
	}
}
