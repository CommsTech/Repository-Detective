package analyzers

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

const lowNoisePenaltyCap = 10.0

// ScoreInput carries scan context for repository health scoring.
type ScoreInput struct {
	ScannerResults []scanners.RunResult
}

// ScoreResult is a transparent 0–100 repository health score.
type ScoreResult struct {
	Percent           float64
	Complete          bool
	IncompleteReason  string
	Explanation       string
	ScoredFindings    int
	IgnoredFindings   int
	ScannerFailures   []string
}

// ComputeScoreResult calculates a repository health score from issue-worthy findings.
// Starts at 100 and subtracts weighted penalties. Suppressed and report-only findings are ignored.
func ComputeScoreResult(issues []ai.CodeIssue, input ScoreInput) ScoreResult {
	failures := scannerFailures(input.ScannerResults)
	if len(failures) > 0 && len(issues) == 0 {
		return ScoreResult{
			Complete:         false,
			IncompleteReason: fmt.Sprintf("scanner evidence incomplete: %s", strings.Join(failures, ", ")),
			Explanation:      "Score unavailable until required scanners produce findings evidence.",
			ScannerFailures:  failures,
		}
	}

	if len(issues) == 0 && len(failures) == 0 {
		return ScoreResult{
			Percent:     100,
			Complete:    true,
			Explanation: "No issue-worthy findings; score starts at 100.",
		}
	}

	var penalty, lowNoisePenalty float64
	scored, ignored := 0, 0
	breakdown := map[string]int{}

	for _, issue := range issues {
		if !shouldAffectScore(issue) {
			ignored++
			continue
		}
		scored++
		p := severityPenaltyPoints(issue.Severity)
		if isLowNoiseFinding(issue) {
			remaining := lowNoisePenaltyCap - lowNoisePenalty
			if remaining <= 0 {
				continue
			}
			if p > remaining {
				p = remaining
			}
			lowNoisePenalty += p
		} else {
			penalty += p
		}
		breakdown[strings.ToLower(strings.TrimSpace(issue.Severity))]++
	}

	penalty += lowNoisePenalty
	score := 100.0 - penalty
	if score < 0 {
		score = 0
	}

	explanation := fmt.Sprintf(
		"Start 100; subtract severity penalties (critical -30, high -15, medium -5, low -1); "+
			"cap graph/low-health noise at %.0f; ignored %d suppressed/report-only findings; scored %d findings.",
		lowNoisePenaltyCap, ignored, scored,
	)
	if len(breakdown) > 0 {
		explanation += fmt.Sprintf(" Breakdown: %v.", breakdown)
	}
	if len(failures) > 0 {
		explanation += fmt.Sprintf(" Note: some scanners did not complete (%s); score is approximate.", strings.Join(failures, ", "))
	}

	return ScoreResult{
		Percent:          score,
		Complete:         true,
		Explanation:      explanation,
		ScoredFindings:   scored,
		IgnoredFindings:  ignored,
		ScannerFailures:  failures,
	}
}

// ComputeOverallScore returns a 0–1 normalized score for legacy callers.
// When incomplete, returns -1.
func ComputeOverallScore(issues []ai.CodeIssue) float64 {
	return ComputeOverallScoreWithInput(issues, ScoreInput{})
}

// ComputeOverallScoreWithInput returns a 0–1 normalized score or -1 when incomplete.
func ComputeOverallScoreWithInput(issues []ai.CodeIssue, input ScoreInput) float64 {
	result := ComputeScoreResult(issues, input)
	if !result.Complete {
		return -1
	}
	return result.Percent / 100.0
}

// FormatOverallScore renders a human-readable score line for reports and issues.
func FormatOverallScore(complete bool, normalized float64, incompleteReason, explanation string) string {
	if !complete || normalized < 0 {
		if strings.TrimSpace(incompleteReason) != "" {
			return "incomplete (" + incompleteReason + ")"
		}
		return "incomplete"
	}
	line := fmt.Sprintf("%.2f%%", normalized*100)
	if strings.TrimSpace(explanation) != "" {
		line += " — " + explanation
	}
	return line
}

func shouldAffectScore(issue ai.CodeIssue) bool {
	action := strings.ToLower(strings.TrimSpace(issue.ReportingAction))
	switch action {
	case profile.ActionSuppressedWithReason, profile.ActionDisabledByPolicy, profile.ActionReportOnly:
		return false
	}
	state := strings.ToLower(strings.TrimSpace(issue.LifecycleState))
	return state != profile.LifecycleSuppressed
}

func severityPenaltyPoints(severity string) float64 {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return 30
	case "high", "error":
		return 15
	case "medium", "warning", "warn":
		return 5
	case "low", "info", "note":
		return 1
	default:
		return 3
	}
}

func isLowNoiseFinding(issue ai.CodeIssue) bool {
	cat := strings.ToLower(strings.TrimSpace(issue.Category))
	src := strings.ToLower(strings.TrimSpace(issue.Source))
	rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
	if cat == "graph" || strings.HasPrefix(rule, "GRAPH-") {
		return true
	}
	switch cat {
	case "tech_debt", "maintainability", "test_gap", "performance", "code_quality", "reliability", "documentation":
		return true
	}
	switch src {
	case "graph", "tech_debt", "maintainability", "test_gap", "performance", "reliability", "health":
		return true
	}
	return false
}

func scannerFailures(results []scanners.RunResult) []string {
	var failed []string
	for _, r := range results {
		switch r.Status {
		case scanners.StatusFailed, scanners.StatusTimedOut, scanners.StatusParseFailed:
			failed = append(failed, r.Scanner+"="+string(r.Status))
		}
	}
	return failed
}
