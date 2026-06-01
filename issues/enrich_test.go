package issues_test

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/issues"
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
