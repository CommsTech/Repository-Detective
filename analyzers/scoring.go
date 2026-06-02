package analyzers

import (
	"strings"

	"git.commsnet.org/commstech/bugbot/ai"
)

// ComputeOverallScore returns a 0–1 repository health score from validated findings.
// Higher is better. Deterministic scans use this for reports and issue summaries.
func ComputeOverallScore(issues []ai.CodeIssue) float64 {
	if len(issues) == 0 {
		return 1.0
	}
	var penalty float64
	for _, issue := range issues {
		penalty += severityPenalty(issue.Severity)
		if issue.Confidence > 0 && issue.Confidence < 1 {
			penalty += (1 - issue.Confidence) * 0.02
		}
	}
	score := 1.0 - penalty
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func severityPenalty(severity string) float64 {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return 0.18
	case "high", "error":
		return 0.09
	case "medium", "warning", "warn":
		return 0.04
	case "low", "info", "note":
		return 0.015
	default:
		return 0.03
	}
}
