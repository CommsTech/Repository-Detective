package issues_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/issues"
)

// TestWriteDogfoodIssueSamples regenerates docs/dogfood-issue-samples/*.md for product review.
func TestWriteDogfoodIssueSamples(t *testing.T) {
	dir := filepath.Join("..", "docs", "dogfood-issue-samples")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 16, 0, 0, 0, time.UTC)

	samples := []struct {
		name  string
		title string
		body  string
	}{
		{
			name:  "01-g201-security.md",
			title: issues.OperatorIssueTitle(&ai.CodeIssue{Title: "SQL string formatting", Source: "gosec", RuleID: "G201", File: "store/learning_sqlite.go", Severity: "medium"}),
			body: issues.RenderIssueBody(issues.IssueRenderInput{
				Issue: &ai.CodeIssue{
					Title: "SQL string formatting", Description: "SQL string formatting",
					Severity: "medium", Category: "security", Source: "gosec", RuleID: "G201",
					File: "store/learning_sqlite.go", LineNumber: 473, Confidence: 0.9,
					CodeSnippet: "query := fmt.Sprintf(`DELETE FROM learning_events WHERE id IN (%s)`, placeholders)",
					Fingerprint: "rd-dogfood-g201", Fixable: "true", FixComplexity: "medium",
				},
				Repository: "commstech/Repository-Detective", Owner: "commstech", RepoName: "Repository-Detective",
				GiteaBaseURL: "https://git.commsnet.org", ScanID: "dogfood-scan-1",
				Commit: "4aa38593deadbeef", FindingID: 1201, Provider: "gitea", Now: now,
			}),
		},
		{
			name:  "02-shellcheck-maintainability.md",
			title: issues.OperatorIssueTitle(&ai.CodeIssue{Title: "unused variable i", Source: "shellcheck", RuleID: "SC2034", File: "scripts/sync-gitea-to-github.sh"}),
			body: issues.RenderIssueBody(issues.IssueRenderInput{
				Issue: &ai.CodeIssue{
					Title: "unused variable i", Severity: "low", Category: "maintainability",
					Source: "shellcheck", RuleID: "SC2034", File: "scripts/sync-gitea-to-github.sh",
					LineNumber: 94, CodeSnippet: "tmp=\"$(mktemp -d)\"\nsha=\"$(git rev-parse --short HEAD)\"\ni=1\necho \"building snapshot\"",
					Fixable: "true", SafeForAutoPR: true, FixComplexity: "small", Confidence: 0.95,
					Fingerprint: "rd-dogfood-sc2034",
				},
				Repository: "commstech/Repository-Detective", ScanID: "dogfood-scan-1",
				Commit: "4aa38593deadbeef", FindingID: 1202, Provider: "gitea", Now: now,
			}),
		},
		{
			name:  "03-dependency-vulnerability.md",
			title: issues.OperatorIssueTitle(&ai.CodeIssue{Title: "CVE-2024-12345 in golang.org/x/net", Source: "grype", RuleID: "CVE-2024-12345", PackageName: "golang.org/x/net", Severity: "high"}),
			body: issues.RenderIssueBody(issues.IssueRenderInput{
				Issue: &ai.CodeIssue{
					Title: "CVE-2024-12345 in golang.org/x/net", Severity: "high", Category: "dependency",
					Source: "grype", RuleID: "CVE-2024-12345", PackageName: "golang.org/x/net",
					File: "go.mod", Confidence: 0.92, Fingerprint: "rd-dogfood-dep",
					Evidence: `{"package":"golang.org/x/net","version":"v0.17.0","fixed_version":"v0.23.0","cve":"CVE-2024-12345","ecosystem":"go"}`,
					Description: "Known vulnerability in golang.org/x/net allows request smuggling under specific proxy configurations.",
					Fixable: "true", FixComplexity: "small",
				},
				Repository: "commstech/Repository-Detective", ScanID: "dogfood-scan-1",
				Commit: "4aa38593deadbeef", FindingID: 1203, Provider: "gitea", Now: now,
			}),
		},
		{
			name: "04-posture-summary.md",
			title: "Repository posture: Action required",
			body: func() string {
				cur := issues.BuildRepositoryPosture(&ai.CodeAnalysisResult{Issues: []ai.CodeIssue{
					{Severity: "critical", Confidence: 0.95},
					{Severity: "high", Confidence: 0.9},
					{Severity: "high", Confidence: 0.88},
					{Severity: "medium", Confidence: 0.8},
					{Severity: "medium", Confidence: 0.5, ReportingAction: "manual_review"},
					{Severity: "medium", Confidence: 0.7, SafeForAutoPR: true, Fixable: "true"},
					{Severity: "medium", Confidence: 0.7, SafeForAutoPR: true, Fixable: "true"},
					{Severity: "low", Confidence: 0.9},
				}})
				prev := issues.RepositoryPosture{Critical: 0, High: 1, Medium: 10, Low: 2, Total: 13}
				issues.EnrichPostureDelta(&cur, &prev)
				cur.ExcludedByCalibration = 64
				ok, reason := issues.ShouldCreatePostureIssue(cur, &prev, issues.DefaultPostureTriggerConfig())
				if ok {
					cur.TriggerReason = reason
				}
				return issues.RenderRepositoryPostureMarkdown(cur) +
					"\n## Context\n\n- **Repository:** commstech/Repository-Detective\n- **Scan ID:** `dogfood-scan-1`\n- **Commit:** `4aa38593deadbeef`\n"
			}(),
		},
	}

	for _, s := range samples {
		content := "# " + s.title + "\n\n" + s.body
		path := filepath.Join(dir, s.name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d bytes)", path, len(content))
	}
}
