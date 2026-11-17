package analyzers

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestComputeScoreResultEmpty(t *testing.T) {
	got := ComputeScoreResult(nil, ScoreInput{})
	if got.Percent != 100 || !got.Complete {
		t.Fatalf("empty want 100 complete, got %+v", got)
	}
}

func TestComputeScoreResultLowGraphDoesNotZero(t *testing.T) {
	issues := make([]ai.CodeIssue, 27)
	for i := range issues {
		issues[i] = ai.CodeIssue{
			Severity:        "low",
			Category:        "graph",
			Source:          "graph",
			RuleID:          "GRAPH-ORPHAN-FILE",
			ReportingAction: profile.ActionAutoIssue,
		}
	}
	got := ComputeScoreResult(issues, ScoreInput{})
	if got.Percent == 0 {
		t.Fatalf("27 low graph findings should not zero score, got %.2f", got.Percent)
	}
	if got.Percent < 80 {
		t.Fatalf("expected score >= 80 with low-noise cap, got %.2f", got.Percent)
	}
}

func TestComputeScoreResultMixedNonCriticalNotZero(t *testing.T) {
	issues := []ai.CodeIssue{
		{Severity: "medium", Category: "security", ReportingAction: profile.ActionAutoIssue},
		{Severity: "medium", Category: "security", ReportingAction: profile.ActionAutoIssue},
		{Severity: "low", Category: "quality", ReportingAction: profile.ActionAutoIssue},
	}
	for i := 3; i < 27; i++ {
		issues = append(issues, ai.CodeIssue{Severity: "low", Category: "quality", ReportingAction: profile.ActionAutoIssue})
	}
	got := ComputeScoreResult(issues, ScoreInput{})
	if got.Percent == 0 {
		t.Fatalf("27 mixed noncritical should not be 0, got %+v", got)
	}
}

func TestComputeScoreResultIgnoresSuppressedAndReportOnly(t *testing.T) {
	issues := []ai.CodeIssue{
		{Severity: "critical", ReportingAction: profile.ActionSuppressedWithReason},
		{Severity: "high", ReportingAction: profile.ActionReportOnly},
		{Severity: "low", ReportingAction: profile.ActionAutoIssue},
	}
	got := ComputeScoreResult(issues, ScoreInput{})
	if got.IgnoredFindings != 2 || got.ScoredFindings != 1 {
		t.Fatalf("ignored/scored counts: %+v", got)
	}
	if got.Percent != 99 {
		t.Fatalf("one low finding should be 99, got %.2f", got.Percent)
	}
}

func TestComputeScoreResultScannerFailureIncompleteWhenNoFindings(t *testing.T) {
	got := ComputeScoreResult(nil, ScoreInput{
		ScannerResults: []scanners.RunResult{{Scanner: "trivy", Status: scanners.StatusTimedOut}},
	})
	if got.Complete {
		t.Fatal("expected incomplete when scanners failed and no findings")
	}
	if !strings.Contains(got.IncompleteReason, "trivy") {
		t.Fatalf("expected scanner in reason: %q", got.IncompleteReason)
	}
	if ComputeOverallScoreWithInput(nil, ScoreInput{
		ScannerResults: []scanners.RunResult{{Scanner: "trivy", Status: scanners.StatusTimedOut}},
	}) != -1 {
		t.Fatal("normalized score should be -1 when incomplete")
	}
}

func TestFormatOverallScoreIncomplete(t *testing.T) {
	got := FormatOverallScore(false, -1, "scanner evidence incomplete", "")
	if got != "incomplete (scanner evidence incomplete)" {
		t.Fatalf("got %q", got)
	}
}

func TestComputeOverallScorePenalizesSeverity(t *testing.T) {
	clean := ComputeOverallScore([]ai.CodeIssue{{Severity: "low", ReportingAction: profile.ActionAutoIssue}})
	critical := ComputeOverallScore([]ai.CodeIssue{{Severity: "critical", ReportingAction: profile.ActionAutoIssue}})
	if critical >= clean {
		t.Fatalf("critical %v should be below low %v", critical, clean)
	}
}
