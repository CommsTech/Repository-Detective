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
		Description:    "Use of eval is risky because attacker-controlled input can execute arbitrary code.",
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
		Commit:       "abcdef1234567890",
		Ref:          "main",
		FindingID:    99,
		Provider:     "gitea",
		Now:          time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
	})

	for _, section := range []string{
		"## Summary",
		"## Why this matters",
		"## Location",
		"## Evidence",
		"## Recommended action",
		"## Verification",
		"<details>",
		"Fingerprint: `rd-abc123`",
		"eval(input)",
		"abcdef1234567890",
		"src/app.py",
	} {
		if !strings.Contains(body, section) {
			t.Fatalf("missing section %q in body:\n%s", section, body)
		}
	}
	for _, banned := range []string{
		"## Finding",
		"## Acceptance criteria",
		"## Report flow",
		"## Proof of concept",
		"## Reproduction",
		"## Issue filing policy",
		"Commit: `main`",
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("bloated/banned section still present: %q", banned)
		}
	}
}

func TestRenderIssueBodyG201Context(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:       "SQL string formatting",
		Description: "SQL string formatting",
		Severity:    "medium",
		Category:    "security",
		Source:      "gosec",
		RuleID:      "G201",
		File:        "store/learning_sqlite.go",
		LineNumber:  473,
		Confidence:  0.9,
		CodeSnippet: "query := fmt.Sprintf(`DELETE ... IN (%s)`, in)",
		Fingerprint: "rd-g201",
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Commit: "deadbeefcafe"})
	if !strings.Contains(body, "SQL injection") {
		t.Fatal("expected actionable G201 impact text")
	}
	if !strings.Contains(body, "placeholder") {
		t.Fatal("expected calibration guidance for trusted placeholders")
	}
	title := OperatorIssueTitle(issue)
	if !strings.Contains(title, "G201") || !strings.Contains(title, "learning_sqlite.go") {
		t.Fatalf("unexpected title %q", title)
	}
}

func TestRenderIssueBodyAutoPRBanner(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:         "unused var",
		Severity:      "low",
		Category:      "maintainability",
		Source:        "shellcheck",
		RuleID:        "SC2034",
		File:          "scripts/run.sh",
		LineNumber:    89,
		CodeSnippet:   "i=1\necho hello",
		Fixable:       "true",
		SafeForAutoPR: true,
		FixComplexity: "small",
		Confidence:    0.95,
		Fingerprint:   "rd-sc",
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue})
	if !strings.Contains(body, "Automated fix available") {
		t.Fatal("expected auto-PR banner")
	}
	if !strings.Contains(body, "i=1") {
		t.Fatal("evidence should show shell lines, not only scanner prose")
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
	for _, want := range []string{"**Container**", "alpine:3.20", "sha256:abc", "openssl", "3.1.4", "3.1.5", "CVE-2024-TEST"} {
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
	for _, want := range []string{"**SBOM**", "lodash", "npm", "MIT"} {
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
	for _, want := range []string{"**Secret / history**", "deadbeef", "Rotation required"} {
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
		t.Fatal("expected report-only yes in details")
	}
}

func TestBuildLabelsIncludesCategoryAndSeverity(t *testing.T) {
	issue := &ai.CodeIssue{
		Severity:   "high",
		Category:   "secret",
		Source:     "gitleaks",
		Confidence: 0.95,
		Fixable:    "true",
	}
	labels := BuildLabels([]string{"custom"}, issue)
	want := map[string]bool{
		"custom":                     true,
		"source/repository-detective": true,
		"category/secret":            true,
		"severity/high":              true,
		"scanner/gitleaks":           true,
		"triage/needs-review":        true,
		"remediation/manual":         true,
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
	found := false
	for _, label := range labels {
		if label == TriageNeedsReview {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s label, got %v", TriageNeedsReview, labels)
	}
}

func TestDecideExistingIssueUpdateSilentByDefault(t *testing.T) {
	issue := &ai.CodeIssue{Severity: "medium", Confidence: 0.9, File: "a.go", LineNumber: 1}
	match := &ExistingIssueMatch{Body: "**Severity:** Medium\n\n`a.go:1`\n\n**Last seen:** Sep 8, 2026\n"}
	d := DecideExistingIssueUpdate(issue, match, "scan-9", time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC))
	if d.Comment {
		t.Fatalf("expected silent update, got comment reason=%s body=%s", d.Reason, d.Body)
	}
}

func TestDecideExistingIssueUpdateSeverityChange(t *testing.T) {
	issue := &ai.CodeIssue{Severity: "high", Confidence: 0.9, File: "a.go", LineNumber: 1}
	match := &ExistingIssueMatch{Body: "**Severity:** Medium\n\n`a.go:1`\n"}
	d := DecideExistingIssueUpdate(issue, match, "scan-9", time.Now().UTC())
	if !d.Comment || d.Reason != "severity_changed" {
		t.Fatalf("expected severity_changed, got %+v", d)
	}
}

func TestRepositoryPostureNoPercentScore(t *testing.T) {
	result := &ai.CodeAnalysisResult{
		Issues: []ai.CodeIssue{
			{Severity: "high", Confidence: 0.9},
			{Severity: "medium", Confidence: 0.8},
			{Severity: "medium", Confidence: 0.5, ReportingAction: "manual_review"},
		},
		OverallScore: 0,
	}
	md := RenderRepositoryPostureMarkdown(BuildRepositoryPosture(result))
	if strings.Contains(md, "%") || strings.Contains(md, "0.00") {
		t.Fatalf("posture must not use percentage score: %s", md)
	}
	if !strings.Contains(md, "Action required") {
		t.Fatal("expected Action required headline")
	}
}

func TestShouldCreateSummaryIssueRare(t *testing.T) {
	small := make([]ai.CodeIssue, 10)
	for i := range small {
		small[i] = ai.CodeIssue{Severity: "medium"}
	}
	if shouldCreateSummaryIssue(small) {
		t.Fatal("medium-only small sets must not create summary issues")
	}
	big := make([]ai.CodeIssue, 20)
	for i := range big {
		big[i] = ai.CodeIssue{Severity: "medium"}
	}
	big[0].Severity = "high"
	if !shouldCreateSummaryIssue(big) {
		t.Fatal("large sets with high findings may create rare posture issues")
	}
}
