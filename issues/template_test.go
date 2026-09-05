package issues

import (
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestRenderIssueBodyIncludesSections(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:          "Semgrep finding: rule-x",
		Description:    "Use of eval is risky",
		Severity:       "high",
		Category:       "security",
		Source:         "semgrep",
		RuleID:         "rule-x",
		Fingerprint:    "rd-abc123",
		File:           "src/app.py",
		LineNumber:     10,
		Confidence:     0.9,
		CodeSnippet:    "eval(input)",
		RegressionRisk: "high",
		Fixable:        "unknown",
		FixComplexity:  "medium",
		RequiredTests:  "Add regression test",
		LifecycleState: LifecycleOpen,
	}

	body := RenderIssueBody(IssueRenderInput{
		Issue:        issue,
		Repository:   "owner/repo",
		Owner:        "owner",
		RepoName:     "repo",
		GiteaBaseURL: "https://git.example.org",
		ScanID:       "scan-1",
		Commit:       "main",
		Ref:          "main",
		FindingID:    99,
		Provider:     "gitea",
		Now:          time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
	})

	for _, section := range []string{
		"## Summary",
		"## Finding",
		"## Location",
		"## Why this matters",
		"## Evidence",
		"## Recommended fix",
		"## Verification",
		"## Issue filing policy",
		"## False-positive guidance",
		"## Repository Detective metadata",
		"## Regression risk",
		"## Suggested tests",
		"## Reproduction",
		"## Report flow",
		"## Acceptance criteria",
		"## Tracking",
		"Repository Detective fingerprint: rd-abc123",
		"- Scan ID: `scan-1`",
		"- Finding ID: `99`",
		"src/app.py",
	} {
		if !strings.Contains(body, section) {
			t.Fatalf("missing section %q in body", section)
		}
	}
}

func TestRenderIssueBodyRedactsSecrets(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "Secret finding",
		Severity:    "high",
		Category:    "secret",
		Source:      "gitleaks",
		RuleID:      "aws-key",
		Confidence:  0.95,
		CodeSnippet: `token="AKIAIOSFODNN7EXAMPLE"`,
		Fingerprint: "rd-secret",
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Repository: "owner/repo"})
	if strings.Contains(body, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatal("issue body must not contain raw secret")
	}
}

func TestRenderIssueBodyContainerFinding(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "CVE in base image",
		Severity:    "high",
		Category:    "vulnerability",
		Source:      "trivy",
		PackageName: "openssl",
		File:        "alpine:3.20",
		Evidence:    `{"image":"alpine:3.20","image_digest":"sha256:abc","version":"3.1.4","fixed_version":"3.1.5","cve":"CVE-2024-TEST"}`,
		Fingerprint: "rd-container",
		Confidence:  0.92,
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Repository: "owner/repo", ScanID: "scan-c"})
	for _, want := range []string{"## Container context", "alpine:3.20", "sha256:abc", "openssl", "3.1.4", "3.1.5", "CVE-2024-TEST"} {
		if !strings.Contains(body, want) {
			t.Fatalf("container body missing %q", want)
		}
	}
}

func TestRenderIssueBodySBOMFinding(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "SBOM component",
		Severity:    "medium",
		Category:    "dependency",
		Source:      "sbom",
		PackageName: "lodash",
		Evidence:    `{"sbom_component":"pkg:npm/lodash@4.17.20","ecosystem":"npm","license":"MIT"}`,
		Fingerprint: "rd-sbom",
		Confidence:  0.88,
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Repository: "owner/repo"})
	for _, want := range []string{"## SBOM context", "lodash", "npm", "MIT"} {
		if !strings.Contains(body, want) {
			t.Fatalf("sbom body missing %q", want)
		}
	}
}

func TestRenderIssueBodyHistoricalSecret(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "Historical secret",
		Severity:    "high",
		Category:    "secret",
		Source:      "gitleaks",
		CommitSHA:   "deadbeef",
		SourceType:  "history",
		Fingerprint: "rd-hist",
		Confidence:  0.9,
		CodeSnippet: "key=[REDACTED]",
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Repository: "owner/repo"})
	for _, want := range []string{"## Secret / history context", "deadbeef", "Rotation required", "historical"} {
		if !strings.Contains(body, want) {
			t.Fatalf("historical secret body missing %q", want)
		}
	}
}

func TestRenderIssueBodyReportOnlyPolicy(t *testing.T) {
	body := RenderIssueBody(IssueRenderInput{
		Issue:      &ai.CodeIssue{Title: "t", Severity: "low", Confidence: 0.5, Fingerprint: "fp"},
		ScanID:     "scan-ro",
		ReportOnly: true,
	})
	if !strings.Contains(body, "Report-only: yes") {
		t.Fatal("expected report-only yes in filing policy")
	}
}

func TestBuildLabelsIncludesCategoryAndSeverity(t *testing.T) {
	issue := &ai.CodeIssue{
		Severity:   "high",
		Category:   "secret",
		Source:     "gitleaks",
		Confidence: 0.95,
	}
	labels := BuildLabels([]string{"custom"}, issue)
	want := map[string]bool{
		"custom":                      true,
		"repository-detective":        true,
		"automated-review":            true,
		"repository-detective/secret": true,
		"severity/high":               true,
		"repository-detective/open":   true,
	}
	for _, label := range labels {
		if !want[label] {
			t.Fatalf("unexpected label %q in %v", label, labels)
		}
		delete(want, label)
	}
	if len(want) > 0 {
		t.Fatalf("missing expected labels: %v", want)
	}
}

func TestConfidenceNeedsHumanReviewLabel(t *testing.T) {
	issue := &ai.CodeIssue{
		Severity:   "medium",
		Category:   "security",
		Source:     "semgrep",
		Confidence: 0.55,
	}
	EnrichIssue("owner/repo", issue, "scan-1")
	labels := BuildLabels(nil, issue)
	want := "repository-detective/needs-human-review"
	found := false
	for _, label := range labels {
		if label == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s label, got %v", want, labels)
	}
}
