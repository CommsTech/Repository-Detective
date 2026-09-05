package issues

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestBacklogControlBlocksLowSeverity(t *testing.T) {
	bc := BacklogControlConfig{
		Enabled:               true,
		AllowNewIssueSeverity: []string{"high", "critical"},
		AllowMinConfidence:    0.85,
		UpdateExistingOnly:    true,
	}
	issue := &ai.CodeIssue{Severity: "medium", Confidence: 0.9}
	blocked, reason := bc.ShouldBlockNewIssue(issue, 50)
	if !blocked {
		t.Fatal("expected medium severity blocked")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestBacklogControlAllowsHighSeverity(t *testing.T) {
	bc := BacklogControlConfig{
		Enabled:               true,
		AllowNewIssueSeverity: []string{"high", "critical"},
		AllowMinConfidence:    0.85,
	}
	issue := &ai.CodeIssue{Severity: "high", Confidence: 0.9}
	blocked, _ := bc.ShouldBlockNewIssue(issue, 200)
	if blocked {
		t.Fatal("expected high severity allowed")
	}
}

func TestBacklogControlBlocksLowConfidenceHigh(t *testing.T) {
	bc := BacklogControlConfig{
		Enabled:               true,
		AllowNewIssueSeverity: []string{"high", "critical"},
		AllowMinConfidence:    0.85,
	}
	issue := &ai.CodeIssue{Severity: "high", Confidence: 0.7}
	blocked, _ := bc.ShouldBlockNewIssue(issue, 50)
	if !blocked {
		t.Fatal("expected low confidence blocked")
	}
}

func TestBacklogControlOpenCap(t *testing.T) {
	bc := BacklogControlConfig{
		Enabled:               true,
		MaxOpenIssues:         100,
		AllowNewIssueSeverity: []string{"high", "critical"},
		AllowMinConfidence:    0.85,
	}
	issue := &ai.CodeIssue{Severity: "medium", Confidence: 0.95}
	blocked, _ := bc.ShouldBlockNewIssue(issue, 150)
	if !blocked {
		t.Fatal("expected cap to block medium at 150 open issues")
	}
}

func TestBacklogControlAllowedSource(t *testing.T) {
	bc := BacklogControlConfig{
		Enabled: true,
		AllowedSources: map[string]bool{
			"gitleaks": true,
		},
		AllowNewIssueSeverity: []string{"high", "critical"},
	}
	issue := &ai.CodeIssue{Severity: "medium", Source: "gitleaks", Confidence: 0.9}
	blocked, _ := bc.ShouldBlockNewIssue(issue, 50)
	if blocked {
		t.Fatal("expected allowed source to bypass severity gate")
	}
}

func TestShouldBlockSummaryIssue(t *testing.T) {
	bc := BacklogControlConfig{Enabled: true}
	if !bc.ShouldBlockSummaryIssue() {
		t.Fatal("expected summary blocked when enabled")
	}
}
