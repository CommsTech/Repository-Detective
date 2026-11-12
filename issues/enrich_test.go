package issues_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/issues"
)

func TestEnrichIssuesSetsFingerprint(t *testing.T) {
	codeIssues := []ai.CodeIssue{
		{Title: "test", File: "main.go", LineNumber: 1, Severity: "high"},
	}
	issues.EnrichIssues("commstech/demo", "scan-1", codeIssues)
	if codeIssues[0].Fingerprint == "" {
		t.Fatal("expected fingerprint after enrich")
	}
}

func TestHadolintNotMarkedFromAI(t *testing.T) {
	issue := ai.CodeIssue{Title: "pin apk", Source: "hadolint", Severity: "medium", File: "Dockerfile", LineNumber: 10}
	issues.EnrichIssue("commstech/demo", &issue, "scan-1")
	if issue.FromAI {
		t.Fatal("hadolint findings must not be treated as AI-generated")
	}
}
