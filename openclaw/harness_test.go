package openclaw_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestExtractJSONObjectFromProse(t *testing.T) {
	raw := "Looking at these findings carefully,\n```json\n{\"review_id\":\"air-1\",\"overall_assessment\":\"mixed\",\"recommendations\":[]}\n```\nHope that helps."
	got := openclaw.ExtractJSONObject(raw)
	if !strings.HasPrefix(got, "{") || !strings.Contains(got, `"review_id":"air-1"`) {
		t.Fatalf("extract failed: %q", got)
	}
}

func TestRepairRelaxedJSONEscapesNewlinesInStrings(t *testing.T) {
	raw := "{\n  \"review_id\": \"air-1\",\n  \"overall_assessment\": \"line1\nline2\",\n  \"recommendations\": []\n}"
	fixed := openclaw.RepairRelaxedJSON(raw)
	var parsed openclaw.ReviewResponse
	if err := json.Unmarshal([]byte(fixed), &parsed); err != nil {
		t.Fatalf("repair failed: %v fixed=%q", err, fixed)
	}
	if parsed.OverallAssessment != "line1\nline2" {
		t.Fatalf("assessment=%q", parsed.OverallAssessment)
	}
}

func TestReviewAcceptsProseWrappedJSON(t *testing.T) {
	resp := openclaw.ReviewResponse{
		ReviewID: "air-test", OverallAssessment: "ok",
		Recommendations: []openclaw.Recommendation{{
			Fingerprint: "fp1", Classification: openclaw.ClassNeedsHumanReview,
			SuggestedAction: "fix", SuggestedSeverity: "high", Reason: "real secret",
		}},
	}
	raw, _ := json.Marshal(resp)
	wrapped := "Looking at the packet:\n" + string(raw) + "\nDone."
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 500
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	client, err := openclaw.NewClient(cfg, stubTransport{content: wrapped})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Review(context.Background(), cfg, "air-test", openclaw.ReviewPacket{
		Findings: []openclaw.FindingInput{{
			Fingerprint: "fp1", Title: "secret", Severity: "high", ProtectedFromDowngrade: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.RecommendationsCount != 1 {
		t.Fatalf("got %+v", result)
	}
}

func TestHarnessPacketQualityWifiLikeMix(t *testing.T) {
	findings := []store.Finding{
		{ID: 1, Fingerprint: "fp-secret", Severity: "high", Category: "secret", Confidence: 0.99, Title: "Possible API key", Source: "gitleaks", RuleID: "generic-api-key", FilePath: "config/example.env"},
		{ID: 2, Fingerprint: "fp-semgrep", Severity: "medium", Category: "security", Confidence: 0.7, Title: "JWT hardcoded", Source: "semgrep", RuleID: "jwt-hardcode", FilePath: "app/auth.py", Line: 42},
		{ID: 3, Fingerprint: "fp-debt1", Severity: "low", Category: "tech_debt", Confidence: 0.95, Title: "Technical debt marker found in code", Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", FilePath: "wifi_collector.py", Line: 100},
		{ID: 4, Fingerprint: "fp-debt2", Severity: "low", Category: "tech_debt", Confidence: 0.95, Title: "Technical debt marker found in code", Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", FilePath: "wifi_collector.py", Line: 200},
		{ID: 5, Fingerprint: "fp-debt3", Severity: "low", Category: "tech_debt", Confidence: 0.95, Title: "Technical debt marker found in code", Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", FilePath: "wifi_collector.py", Line: 300},
		{ID: 6, Fingerprint: "fp-debt4", Severity: "medium", Category: "tech_debt", Confidence: 0.95, Title: "Technical debt marker found in code", Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", FilePath: "wifi_collector.py", Line: 400},
		{ID: 7, Fingerprint: "fp-ruff1", Severity: "info", Category: "maintainability", Confidence: 0.9, Title: "Ruff lint", Source: "ruff", RuleID: "F401", FilePath: "a.py"},
		{ID: 8, Fingerprint: "fp-grype", Severity: "high", Category: "dependency", Confidence: 0.9, Title: "CVE in lib", Source: "grype", RuleID: "GHSA-x", PackageName: "lib"},
	}
	instances := map[int64]store.FindingInstance{
		1: {EvidenceRedacted: "KEY=***REDACTED*** in example.env"},
		2: {EvidenceRedacted: "jwt.encode(..., key='hardcoded')", RawMetadataJSON: []byte(`{"snippet":"jwt.encode(payload, key='hardcoded')"}`)},
		3: {}, 4: {}, 5: {}, 6: {}, 7: {},
		8: {EvidenceRedacted: "GHSA-x affects lib < 1.2.3"},
	}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 4000
	cfg.MaxFindingsPerScan = 8
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	cfg.CAH = openclaw.DefaultCAHConfig()
	cfg.CAH.MaxCandidates = 8
	cfg.CAH.TokenBudgetPerScan = 3000
	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID: "scan-wifi-like", Repository: store.Repository{ID: 10, FullName: "commstech/Wifi_Collector"},
		ScanType: openclaw.ScanTypeRepo, Findings: findings, Instances: instances,
		ScannerCoverage: []string{"gitleaks:found", "semgrep:found", "tech_debt:found", "ruff:found", "grype:found"},
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.HarnessVersion != openclaw.HarnessVersion {
		t.Fatalf("harness version %q", pkt.HarnessVersion)
	}
	if len(pkt.Findings) == 0 {
		t.Fatal("expected findings")
	}
	sources := map[string]int{}
	protected := 0
	for _, f := range pkt.Findings {
		sources[f.Source]++
		if f.ProtectedFromDowngrade {
			protected++
		}
		if f.CoachFocus == "" {
			t.Fatalf("missing coach_focus for %s", f.Fingerprint)
		}
	}
	if protected == 0 {
		t.Fatal("expected protected coach lane findings (secret/grype/semgrep)")
	}
	if sources["tech_debt"] > 3 {
		t.Fatalf("tech_debt flooded packet: %d", sources["tech_debt"])
	}
	if sources["gitleaks"]+sources["grype"]+sources["semgrep"] == 0 {
		t.Fatalf("expected security/dependency findings in packet, got %#v", sources)
	}
	// Capture request shape for agent harness inspection.
	var captured ai.ChatRequest
	transport := captureTransport{onComplete: func(req ai.ChatRequest) {
		captured = req
	}, content: `{"review_id":"air-x","overall_assessment":"ok","recommendations":[]}`}
	client, err := openclaw.NewClient(cfg, transport)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Review(context.Background(), cfg, "air-x", pkt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(captured.Messages[0].Content, "JSON object only") {
		t.Fatal("system prompt missing JSON-only contract")
	}
	if !strings.Contains(captured.Messages[1].Content, "Packet JSON:") {
		t.Fatal("user prompt missing packet envelope")
	}
	if !strings.Contains(captured.Messages[1].Content, `"harness_version"`) {
		t.Fatal("packet missing harness_version")
	}
}

type captureTransport struct {
	content    string
	onComplete func(ai.ChatRequest)
}

func (c captureTransport) Name() string { return "capture" }
func (c captureTransport) Complete(_ context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if c.onComplete != nil {
		c.onComplete(req)
	}
	return &ai.ChatResponse{Content: c.content, Model: "test"}, nil
}
