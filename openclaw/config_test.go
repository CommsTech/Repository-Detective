package openclaw_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
)

func TestLegacyConfigKeysMerge(t *testing.T) {
	cfg := openclaw.Config{
		LegacyEnabled:            true,
		LegacyEndpoint:           "http://legacy.example/v1",
		LegacyMaxTokensPerScan:   500,
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

func TestDefaultConfigIsLeanAndSafe(t *testing.T) {
	cfg := openclaw.DefaultConfig().Normalized()
	if cfg.AutoAfterScan {
		t.Fatal("auto_after_scan must default false")
	}
	if cfg.MaxFindingsPerScan > 12 {
		t.Fatalf("max findings too high: %d", cfg.MaxFindingsPerScan)
	}
	if cfg.CAH.MaxCandidates > 6 {
		t.Fatalf("cah max candidates too high: %d", cfg.CAH.MaxCandidates)
	}
	if cfg.CAH.TokenBudgetPerScan > 1200 {
		t.Fatalf("token budget too high: %d", cfg.CAH.TokenBudgetPerScan)
	}
	if cfg.ValueMode != openclaw.ValueModeActionableSecurity {
		t.Fatalf("value mode: %s", cfg.ValueMode)
	}
	if !cfg.AbortOnFirstTimeout {
		t.Fatal("abort_on_first_timeout should default true")
	}
}

func TestCanInvokeRequiresExplicitTokenBudget(t *testing.T) {
	cfg := openclaw.Config{
		Enabled:          true,
		Endpoint:         "http://x",
		MaxTokensPerScan: 900,
		CAH:              openclaw.CAHConfig{TokenBudgetPerScan: 900, MaxCandidates: 4},
	}.Normalized()
	if !cfg.CanInvoke() {
		t.Fatal("expected CanInvoke true with explicit token budget")
	}
}
