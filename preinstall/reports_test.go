package preinstall_test

import (
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

func testReportConfig() preinstall.Config {
	cfg := preinstall.DefaultConfig()
	cfg.RepositoryDetectiveProjectURL = "https://git.example.com/commstech/Repository-Detective"
	return cfg
}

func TestGenerateReportsInstallSummaryAlways(t *testing.T) {
	cfg := testReportConfig()
	audit := store.AuditRequest{
		AuditID:           "a1",
		NormalizedRepoURL: "https://github.com/o/r",
		CommitSHA:         "abc",
		StartedAt:         time.Now().UTC(),
		RiskScore:         0,
		Recommendation:    store.AuditRecommendationSafe,
	}
	reports := preinstall.GenerateReports(cfg, audit, nil, nil)
	if len(reports) == 0 {
		t.Fatal("expected install summary report")
	}
	if reports[0].ReportType != store.ReportTypeInstallRiskSummary {
		t.Fatalf("expected install_risk_summary, got %s", reports[0].ReportType)
	}
	if !strings.Contains(reports[0].BodyMarkdown, preinstall.ReportDisclaimerText()) {
		t.Fatal("missing human review disclaimer")
	}
	if !strings.Contains(reports[0].BodyMarkdown, preinstall.ReportFooterLead()) {
		t.Fatal("missing marketing footer")
	}
	if !strings.Contains(reports[0].BodyMarkdown, cfg.RepositoryDetectiveProjectURL) {
		t.Fatal("missing project URL in footer")
	}
}

func TestGenerateReportsSecurityDisclosureForHighFinding(t *testing.T) {
	cfg := testReportConfig()
	audit := store.AuditRequest{
		AuditID:           "a1",
		NormalizedRepoURL: "https://github.com/o/r",
		CommitSHA:         "abc",
		StartedAt:         time.Now().UTC(),
		RiskScore:         40,
		Recommendation:    store.AuditRecommendationCaution,
	}
	findings := []store.AuditFinding{{
		ID: 1, Severity: "high", Category: "security", Confidence: 0.95,
		Title: "SQL injection risk", FilePath: "app.go", EvidenceRedacted: "query built from input",
	}}
	reports := preinstall.GenerateReports(cfg, audit, findings, nil)
	var foundSecurity bool
	for _, r := range reports {
		if r.ReportType == store.ReportTypeSecurityDisclosure {
			foundSecurity = true
			if !strings.Contains(r.BodyMarkdown, "Private security disclosure draft") {
				t.Fatal("security draft should be marked private")
			}
		}
	}
	if !foundSecurity {
		t.Fatal("expected security disclosure draft")
	}
}

func TestGenerateReportsNoPublicDraftForSecrets(t *testing.T) {
	cfg := testReportConfig()
	audit := store.AuditRequest{AuditID: "a1", NormalizedRepoURL: "https://github.com/o/r", StartedAt: time.Now().UTC()}
	findings := []store.AuditFinding{{
		ID: 2, Severity: "high", Category: "secret", Source: "gitleaks", Confidence: 0.99,
		Title: "AWS key", EvidenceRedacted: "AKIA1234567890ABCD",
	}}
	reports := preinstall.GenerateReports(cfg, audit, findings, nil)
	for _, r := range reports {
		if r.ReportType == store.ReportTypeGeneralBug || r.ReportType == store.ReportTypeSecurityDisclosure {
			t.Fatalf("should not generate external draft for secret finding, got %s", r.ReportType)
		}
	}
}

func TestReportsDoNotContainRawSecrets(t *testing.T) {
	cfg := testReportConfig()
	audit := store.AuditRequest{AuditID: "a1", NormalizedRepoURL: "https://github.com/o/r", StartedAt: time.Now().UTC()}
	raw := `password = "supersecret12345"`
	findings := []store.AuditFinding{{
		ID: 3, Severity: "high", Category: "quality", Confidence: 0.95,
		Title: "Hardcoded value", EvidenceRedacted: raw,
	}}
	reports := preinstall.GenerateReports(cfg, audit, findings, nil)
	for _, r := range reports {
		if strings.Contains(r.BodyMarkdown, "supersecret12345") {
			t.Fatalf("report %s contains raw secret", r.ReportType)
		}
	}
}

func TestReportFooterOmitsProjectLinkWhenDisabled(t *testing.T) {
	cfg := testReportConfig()
	cfg.ReportIncludeProjectLink = false
	audit := store.AuditRequest{AuditID: "a1", NormalizedRepoURL: "https://github.com/o/r", StartedAt: time.Now().UTC()}
	reports := preinstall.GenerateReports(cfg, audit, nil, nil)
	if strings.Contains(reports[0].BodyMarkdown, cfg.RepositoryDetectiveProjectURL) {
		t.Fatal("project URL should be omitted when disabled")
	}
}
