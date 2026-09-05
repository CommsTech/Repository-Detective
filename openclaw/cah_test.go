package openclaw_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestCAHSkipsHighConfidenceCritical(t *testing.T) {
	findings := []store.Finding{{
		ID: 1, Fingerprint: "fp-critical", Severity: "critical", Confidence: 0.95, Title: "critical issue",
	}}
	instances := map[int64]store.FindingInstance{1: {EvidenceRedacted: "long evidence " + repeat("x", 50)}}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 2000
	cah := openclaw.DefaultCAHConfig()
	selected, scores := openclaw.SelectCAHCandidates(findings, instances, nil, cfg, cah)
	if len(selected) != 0 {
		t.Fatalf("expected 0 selected, got %d", len(selected))
	}
	if len(scores) == 0 || scores[0].SkipReason == "" {
		t.Fatal("expected skip reason for protected finding")
	}
}

func TestCAHSelectsLowConfidence(t *testing.T) {
	findings := []store.Finding{{
		ID: 2, Fingerprint: "fp-low", Severity: "low", Confidence: 0.35, Title: "maybe noise in test fixture",
	}}
	instances := map[int64]store.FindingInstance{2: {EvidenceRedacted: "weak pattern match with limited context " + repeat("x", 20)}}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 2000
	cah := openclaw.DefaultCAHConfig()
	cah.MinUncertaintyScore = 0.35
	selected, _ := openclaw.SelectCAHCandidates(findings, instances, nil, cfg, cah)
	if len(selected) != 1 {
		t.Fatalf("expected 1 selected, got %d", len(selected))
	}
}

func TestCAHFallbackWhenAllFiltered(t *testing.T) {
	findings := make([]store.Finding, 3)
	instances := map[int64]store.FindingInstance{}
	for i := range findings {
		id := int64(i + 1)
		findings[i] = store.Finding{
			ID: id, Fingerprint: "fp-medium", Severity: "medium",
			Confidence: 0.95, Title: "high confidence medium issue",
		}
		instances[id] = store.FindingInstance{EvidenceRedacted: "evidence " + repeat("x", 40)}
	}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 2000
	cah := openclaw.DefaultCAHConfig()
	selected, _ := openclaw.SelectCAHCandidates(findings, instances, nil, cfg, cah)
	if len(selected) == 0 {
		t.Fatal("expected CAH fallback to select findings when uncertainty filter removes all")
	}
	if len(selected) > cah.MaxCandidates {
		t.Fatalf("expected at most %d selected, got %d", cah.MaxCandidates, len(selected))
	}
}

func TestTokenBudgetEnforced(t *testing.T) {
	findings := make([]store.Finding, 5)
	instances := map[int64]store.FindingInstance{}
	for i := range findings {
		id := int64(i + 1)
		findings[i] = store.Finding{ID: id, Fingerprint: "fp", Severity: "medium", Confidence: 0.5, Title: "x"}
		instances[id] = store.FindingInstance{EvidenceRedacted: repeat("e", 500)}
	}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 200
	cah := openclaw.DefaultCAHConfig()
	cah.TokenBudgetPerScan = 200
	cah.MinUncertaintyScore = 0.2
	selected, _ := openclaw.SelectCAHCandidates(findings, instances, nil, cfg, cah)
	if len(selected) > 2 {
		t.Fatalf("expected token budget to limit selection, got %d", len(selected))
	}
}

func repeat(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[0]
	}
	return string(out)
}
