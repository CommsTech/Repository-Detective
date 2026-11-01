package ai

import (
	"strings"
	"testing"
)

// Regression: OpenRouter HTTP-Referer header default must not advertise a
// deployment-specific forge URL. Neutral placeholder keeps provider tracking
// functional without leaking where the deployment lives.
func TestOpenRouterRefererIsNeutral(t *testing.T) {
	cfg, err := ResolveConfig(Config{Provider: ProviderOpenRouter, Model: "x"}, LegacyConfig{})
	if err != nil {
		t.Fatalf("normalize openrouter config: %v", err)
	}
	referer := cfg.ExtraHeaders["HTTP-Referer"]
	if referer == "" {
		t.Fatal("expected HTTP-Referer header for openrouter provider")
	}
	lowered := strings.ToLower(referer)
	for _, bad := range []string{"commsnet", "commstech"} {
		if strings.Contains(lowered, bad) {
			t.Fatalf("openrouter referer leaks deployment marker %q: %s", bad, referer)
		}
	}
}
