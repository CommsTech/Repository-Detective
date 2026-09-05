package openclaw_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
)

func TestLegacyConfigKeysMerge(t *testing.T) {
	cfg := openclaw.Config{
		LegacyEnabled:           true,
		LegacyEndpoint:          "http://legacy.example/v1",
		LegacyMaxTokensPerScan:  500,
		LegacyMaxFindingsPerScan: 10,
	}.Normalized()
	if !cfg.Enabled {
		t.Fatal("expected legacy enabled to merge")
	}
	if cfg.EffectiveEndpoint() != "http://legacy.example/v1" {
		t.Fatalf("endpoint: %s", cfg.EffectiveEndpoint())
	}
	if cfg.MaxTokensPerScan != 500 {
		t.Fatalf("tokens: %d", cfg.MaxTokensPerScan)
	}
}

func TestPreferredKeysOverrideLegacy(t *testing.T) {
	cfg := openclaw.Config{
		Enabled:          true,
		Endpoint:         "http://preferred/v1",
		MaxTokensPerScan: 100,
		LegacyEnabled:    false,
		LegacyEndpoint:   "http://legacy/v1",
	}.Normalized()
	if cfg.EffectiveEndpoint() != "http://preferred/v1" {
		t.Fatalf("endpoint: %s", cfg.EffectiveEndpoint())
	}
}

func TestZeroTokensMeansNoInvoke(t *testing.T) {
	cfg := openclaw.Config{Enabled: true, Endpoint: "http://x", MaxTokensPerScan: 0}.Normalized()
	if cfg.CanInvoke() {
		t.Fatal("expected CanInvoke false with zero token budget")
	}
}
