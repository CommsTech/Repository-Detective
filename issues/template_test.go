package issues

import (
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
)

func TestRenderIssueBodyIncludesSections(t *testing.T) {
	issue := &ai.CodeIssue{
		Title:          "Semgrep finding: rule-x",
		Description:    "Use of eval is risky",
		Severity:       "high",
		Category:       "security",
		Source:         "semgrep",
		RuleID:         "rule-x",
		Fingerprint:    "bugbot-abc123",
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
		Issue:      issue,
		Repository: "owner/repo",
		ScanID:     "scan-1",
		Commit:     "abc1234",
		Now:        time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
	})

	for _, section := range []string{
		"## Summary",
		"## Finding Type",
		"## Location",
		"## Why this matters",
		"## Evidence",
		"## Recommended fix",
		"## Regression risk",
		"## Suggested tests",
		"## Tracking",
		"Repository Detective fingerprint: bugbot-abc123",
		"Scan ID: scan-1",
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
		Fingerprint: "bugbot-secret",
	}
	body := RenderIssueBody(IssueRenderInput{Issue: issue, Repository: "owner/repo"})
	if strings.Contains(body, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatal("issue body must not contain raw secret")
	}
}

func TestBuildLabelsIncludesCategoryAndSeverity(t *testing.T) {
	SetLabelCompatMode(LabelCompatDual)
	issue := &ai.CodeIssue{
		Severity:   "high",
		Category:   "secret",
		Source:     "gitleaks",
		Confidence: 0.95,
	}
	labels := BuildLabels([]string{"custom"}, issue)
	want := map[string]bool{
		"custom":                      true,
		"bugbot":                      true,
		"repository-detective":        true,
		"automated-review":            true,
		"bugbot/secret":               true,
		"repository-detective/secret": true,
		"severity/high":               true,
		"bugbot/open":                 true,
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
	found := false
	for _, label := range labels {
		if label == LifecycleNeedsHumanReview {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected needs-human-review label, got %v", labels)
	}
}
