package gitea

import (
	"fmt"
	"strings"
)

// ChecksConfig controls commit status evaluation.
type ChecksConfig struct {
	Context                string
	TargetURL              string
	FailOn                 string
	WarnOn                 string
	IncludeScannerFailures bool
}

// ScannerResultSummary carries scanner status for commit checks.
type ScannerResultSummary struct {
	Scanner string
	Status  string
}

// CommitStatusEvaluation is the logical commit status outcome.
type CommitStatusEvaluation struct {
	State       string
	Description string
}

// PendingCommitStatusEvaluation returns the pending scan status.
func PendingCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:       CommitStatePending,
		Description: "Repository-Detective scan started",
	}
}

// SkippedCommitStatusEvaluation returns the skip reason when no SHA is available.
func SkippedCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:       CommitStateSuccess,
		Description: "Repository-Detective scan skipped status: no commit SHA",
	}
}

// AnalysisFailedCommitStatusEvaluation returns status when analysis fails.
func AnalysisFailedCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:       CommitStateError,
		Description: "Repository-Detective scan failed",
	}
}

// EvaluateCommitStatus computes final commit status from issue severities and scanner results.
func EvaluateCommitStatus(severities []string, scannerResults []ScannerResultSummary, cfg ChecksConfig) CommitStatusEvaluation {
	if cfg.IncludeScannerFailures && hasBadScannerFailure(scannerResults) {
		return CommitStatusEvaluation{
			State:       CommitStateError,
			Description: "Repository-Detective scan completed with scanner failures",
		}
	}

	counts := countSeverities(severities)
	failOn := normalizeSeverityThreshold(cfg.FailOn, "high")
	warnOn := normalizeSeverityThreshold(cfg.WarnOn, "medium")

	if hasSeverityAtOrAbove(severities, failOn) {
		return CommitStatusEvaluation{
			State:       CommitStateFailure,
			Description: formatFindingDescription(counts),
		}
	}

	if hasSeverityAtOrAbove(severities, warnOn) {
		desc := formatFindingDescription(counts)
		if MapGiteaCommitState(CommitStateWarning) == CommitStateFailure {
			desc = "Repository-Detective warning: " + strings.TrimPrefix(desc, "Repository-Detective ")
		}
		return CommitStatusEvaluation{
			State:       CommitStateWarning,
			Description: desc,
		}
	}

	return CommitStatusEvaluation{
		State:       CommitStateSuccess,
		Description: "Repository-Detective scan passed with no findings",
	}
}

func normalizeSeverityThreshold(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func severityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "crit":
		return 5
	case "high", "error":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low", "info", "note":
		return 2
	case "informational":
		return 1
	default:
		return 0
	}
}

func hasSeverityAtOrAbove(severities []string, threshold string) bool {
	minRank := severityRank(threshold)
	if minRank == 0 {
		return false
	}
	for _, severity := range severities {
		if severityRank(severity) >= minRank {
			return true
		}
	}
	return false
}

func countSeverities(severities []string) map[string]int {
	counts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}
	for _, severity := range severities {
		key := normalizeSeverityBucket(severity)
		counts[key]++
	}
	return counts
}

func normalizeSeverityBucket(severity string) string {
	switch severityRank(severity) {
	case 5:
		return "critical"
	case 4:
		return "high"
	case 3:
		return "medium"
	case 2:
		return "low"
	default:
		return "info"
	}
}

func formatFindingDescription(counts map[string]int) string {
	parts := make([]string, 0, 5)
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		if counts[severity] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[severity], severity))
		}
	}
	if len(parts) == 0 {
		return "Repository-Detective scan passed with no findings"
	}
	return "Repository-Detective found " + strings.Join(parts, ", ") + " findings"
}

func hasBadScannerFailure(results []ScannerResultSummary) bool {
	for _, result := range results {
		switch strings.ToLower(strings.TrimSpace(result.Status)) {
		case "failed", "timed_out", "parse_failed":
			return true
		}
	}
	return false
}
