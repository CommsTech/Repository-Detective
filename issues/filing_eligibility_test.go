package issues

import (
	"git.commsnet.org/commstech/repository-detective/ai"
	"testing"
)

func TestAutomaticFilingGateFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		issue ai.CodeIssue
		ok    bool
	}{
		{"missing confidence", ai.CodeIssue{Title: "Unknown", Category: "security", Confidence: 0}, false},
		{"low confidence", ai.CodeIssue{Title: "Unknown", Category: "security", Confidence: 0.49}, false},
		{"deterministic with evidence", ai.CodeIssue{Title: "CVE", Category: "dependency", Confidence: 0.97}, true},
		{"AI lacking context", ai.CodeIssue{Title: "Claim", Category: "security", Confidence: 0.99, FromAI: true}, false},
		{"AI weak confidence", ai.CodeIssue{Title: "Claim", Category: "security", Confidence: 0.89, FromAI: true, File: "app.go", LineNumber: 4, Description: "risk", Evidence: "line"}, false},
		{"AI with location and evidence", ai.CodeIssue{Title: "Claim", Category: "security", Confidence: 0.95, FromAI: true, File: "app.go", LineNumber: 4, Description: "risk", Evidence: "line"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := EligibleForAutomaticFiling(&tt.issue, 0.5)
			if got != tt.ok {
				t.Fatalf("eligible=%v want %v", got, tt.ok)
			}
		})
	}
}

func TestGithubClientMissingDoesNotRouteToGitea(t *testing.T) {
	m := &Manager{giteaForge: &GiteaForge{}}
	if m.ForgeFor("github") != nil {
		t.Fatal("missing GitHub client must not default to Gitea")
	}
}
