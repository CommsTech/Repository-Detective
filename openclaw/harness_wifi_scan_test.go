//go:build wifi_harness

package openclaw_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
)

// Deterministic Wifi_Collector scan 483e858c63e7e8a2 harness shape (no live DB, no real AI).
const (
	wifiTargetScanID       = "483e858c63e7e8a2"
	wifiTargetRepoID       = int64(10)
	wifiTargetRepoName     = "commstech/Wifi_Collector"
	wifiTargetCommitPrefix = "bde0ca99a72b"
	wifiHighFindingID      = int64(128584)
	wifiHighFingerprint    = "rd-07d2a81ff8aff859"
)

func wifiScan483Fixture() (findings []store.Finding, instances map[int64]store.FindingInstance) {
	// Representative mix: one high Trivy dependency + noise that previously flooded packets.
	findings = []store.Finding{
		{
			ID: wifiHighFindingID, RepositoryID: wifiTargetRepoID, Fingerprint: wifiHighFingerprint,
			Category: "dependency", Severity: "high", Confidence: 0.98, Source: "trivy",
			RuleID: "TRIVY-CVE-2024-26130", PackageName: "cryptography", FilePath: "requirements.txt",
			Title:  "python-cryptography: NULL pointer dereference with pkcs12.serialize_key_and_certificates",
			Status: store.FindingStatusOpen,
		},
		{
			ID: 2, RepositoryID: wifiTargetRepoID, Fingerprint: "fp-medium-static",
			Category: "maintainability", Severity: "medium", Confidence: 0.9, Source: "static",
			RuleID: "HEALTH-LARGE-FILE", FilePath: "wifi_collector.py", Title: "Large file",
			Status: store.FindingStatusOpen,
		},
	}
	for i := 0; i < 20; i++ {
		id := int64(100 + i)
		findings = append(findings, store.Finding{
			ID: id, RepositoryID: wifiTargetRepoID, Fingerprint: "fp-debt-" + strconv.Itoa(i),
			Category: "tech_debt", Severity: "low", Confidence: 0.95, Source: "tech_debt",
			RuleID: "HEALTH-TECH-MARKER", FilePath: "wifi_collector.py", Line: i + 1,
			Title: "Technical debt marker found in code", Status: store.FindingStatusOpen,
		})
	}
	for i := 0; i < 15; i++ {
		id := int64(200 + i)
		findings = append(findings, store.Finding{
			ID: id, RepositoryID: wifiTargetRepoID, Fingerprint: "fp-ruff-" + strconv.Itoa(i),
			Category: "quality", Severity: "info", Confidence: 0.55, Source: "ruff",
			RuleID: "F401", FilePath: "helpers.py", Title: "Ruff lint", Status: store.FindingStatusOpen,
		})
	}

	cryptoEvidence := "python-cryptography: NULL pointer dereference with pkcs12.serialize_key_and_certificates " +
		"when called with a non-matching certificate and private key and an hmac_hash override\n\n" +
		"**Dedup cluster `cluster-032`** — merged 7 related finding(s):\n" +
		"- GHSA-r6ph-v2qm-q3c2 in cryptography (`requirements.txt:0`, confidence 0.97)\n" +
		"- cryptography: Subgroup Attack Due to Missing Subgroup Validation\n" +
		strings.Repeat("evidence-padding-", 40) // force BoundEvidence truncation path

	instances = map[int64]store.FindingInstance{
		wifiHighFindingID: {
			FindingID: wifiHighFindingID, ScanID: wifiTargetScanID,
			EvidenceRedacted: cryptoEvidence,
			RawMetadataJSON:  []byte(`{"fixable":"true","from_ai":false,"rule_id":"TRIVY-CVE-2024-26130","scan_id":"483e858c63e7e8a2","source":"trivy"}`),
		},
		2: {FindingID: 2, ScanID: wifiTargetScanID, EvidenceRedacted: "wifi_collector.py exceeds size threshold"},
	}
	for _, f := range findings {
		if _, ok := instances[f.ID]; ok {
			continue
		}
		instances[f.ID] = store.FindingInstance{FindingID: f.ID, ScanID: wifiTargetScanID, EvidenceRedacted: "marker"}
	}
	return findings, instances
}

func wifiHarnessConfig() openclaw.Config {
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 4000
	cfg.MaxFindingsPerScan = 12
	cfg.FallbackEndpoint = "http://127.0.0.1:1/v1"
	cfg.CAH = openclaw.DefaultCAHConfig()
	cfg.CAH.MaxCandidates = 12
	cfg.CAH.TokenBudgetPerScan = 3000
	cfg.CAH.MaxProtectedCoach = 2
	return cfg
}

func TestWifiCollectorScan483HarnessDeterministic(t *testing.T) {
	findings, instances := wifiScan483Fixture()
	cfg := wifiHarnessConfig()

	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID: wifiTargetScanID,
		Repository: store.Repository{
			ID: wifiTargetRepoID, FullName: wifiTargetRepoName,
		},
		ScanType:        openclaw.ScanTypeRepo,
		IssueFiling:     "off",
		RemediationPR:   "off",
		ScannerCoverage: []string{"trivy:found", "ruff:found", "tech_debt:found", "static:found"},
		Findings:        findings,
		Instances:       instances,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if pkt.ScanID != wifiTargetScanID || pkt.RepoID != wifiTargetRepoID || pkt.RepoName != wifiTargetRepoName {
		t.Fatalf("packet identity mismatch: scan=%q repo=%d/%q", pkt.ScanID, pkt.RepoID, pkt.RepoName)
	}
	if pkt.HarnessVersion != openclaw.HarnessVersion {
		t.Fatalf("harness version %q", pkt.HarnessVersion)
	}
	if len(pkt.Findings) == 0 || len(pkt.Findings) > cfg.MaxFindingsPerScan {
		t.Fatalf("unexpected packet size %d (max %d)", len(pkt.Findings), cfg.MaxFindingsPerScan)
	}

	var high *openclaw.FindingInput
	sources := map[string]int{}
	for i := range pkt.Findings {
		f := &pkt.Findings[i]
		sources[f.Source]++
		if f.Fingerprint == wifiHighFingerprint {
			high = f
		}
		if len(f.EvidenceRedacted) > openclaw.MaxEvidenceChars+len("…[truncated]") {
			t.Fatalf("evidence not bounded for %s len=%d", f.Fingerprint, len(f.EvidenceRedacted))
		}
	}
	if high == nil {
		t.Fatalf("expected high cryptography finding in packet, sources=%v", sources)
	}
	if high.Source != "trivy" || high.Path != "requirements.txt" || !strings.Contains(strings.ToLower(high.Title), "cryptography") {
		t.Fatalf("high finding shape wrong: %+v", high)
	}
	if high.EvidenceRedacted == "" {
		t.Fatal("expected preserved redacted evidence for cryptography finding")
	}
	if !high.ProtectedFromDowngrade || high.CoachLane != "protected_coach" {
		t.Fatalf("expected protected coach lane, got prot=%v lane=%q", high.ProtectedFromDowngrade, high.CoachLane)
	}
	if sources["tech_debt"] == len(pkt.Findings) {
		t.Fatal("packet still entirely tech_debt")
	}

	est := openclaw.EstimatePacketTokens(pkt)
	// MaxTokensPerScan gates completion size; CAH TokenBudgetPerScan bounds selection.
	// Packet JSON estimate should stay in a sane envelope for these budgets.
	if est <= 0 {
		t.Fatal("expected positive token estimate")
	}
	softCap := cfg.CAH.TokenBudgetPerScan*3 + 800
	if softCap < 2000 {
		softCap = 2000
	}
	if est > softCap {
		t.Fatalf("token estimate %d exceeds soft packet cap %d", est, softCap)
	}

	redactions, err := openclaw.RedactPacket(&pkt, cfg)
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	_ = redactions

	// Metadata-only fake review — never hits a real provider.
	resp := openclaw.ReviewResponse{
		ReviewID: "air-wifi-483", OverallAssessment: "high dependency vuln present; noise deprioritized",
		Recommendations: []openclaw.Recommendation{{
			Fingerprint: wifiHighFingerprint, Classification: openclaw.ClassLikelyTruePositive,
			SuggestedAction: "fix", SuggestedSeverity: "high", SuggestedConfidence: "high",
			Reason: "Trivy cryptography advisory on requirements.txt — upgrade path, no secret material",
		}},
	}
	raw, _ := json.Marshal(resp)
	client, err := openclaw.NewClient(cfg, stubTransport{content: string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Review(context.Background(), cfg, "air-wifi-483", pkt)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.RecommendationsCount < 1 {
		t.Fatalf("fake review failed: %+v", result)
	}
}

func TestWifiCollectorScan483TokenBudgetPrefersDependency(t *testing.T) {
	findings, instances := wifiScan483Fixture()
	cfg := wifiHarnessConfig()
	cfg.MaxTokensPerScan = 900
	cfg.CAH.TokenBudgetPerScan = 600
	cfg.MaxFindingsPerScan = 6
	cfg.CAH.MaxCandidates = 6

	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID:     wifiTargetScanID,
		Repository: store.Repository{ID: wifiTargetRepoID, FullName: wifiTargetRepoName},
		ScanType:   openclaw.ScanTypeRepo,
		Findings:   findings,
		Instances:  instances,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range pkt.Findings {
		if f.Fingerprint == wifiHighFingerprint {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tight token budget still must keep high dependency finding, got %d findings", len(pkt.Findings))
	}
	if len(pkt.Findings) > cfg.MaxFindingsPerScan {
		t.Fatalf("findings %d exceed max %d", len(pkt.Findings), cfg.MaxFindingsPerScan)
	}
}

func TestBoundEvidencePreservesShortRedactedText(t *testing.T) {
	short := "cryptography 42.0.2 advisory (redacted)"
	if got := openclaw.BoundEvidence(short, openclaw.MaxEvidenceChars); got != short {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("x", openclaw.MaxEvidenceChars+50)
	got := openclaw.BoundEvidence(long, openclaw.MaxEvidenceChars)
	if !strings.HasSuffix(got, "…[truncated]") || len(got) <= openclaw.MaxEvidenceChars {
		t.Fatalf("expected truncation, len=%d", len(got))
	}
}

// Ensure captureTransport from harness_test remains unused here; stubTransport is enough.
var _ ai.ChatTransport = stubTransport{}
