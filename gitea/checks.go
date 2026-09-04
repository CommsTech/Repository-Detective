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
	Scanner  string
	Status   string
	Required bool // when true, incomplete/unavailable blocks POLICY_MET
}

// CommitStatusEvaluation is the logical commit status outcome.
type CommitStatusEvaluation struct {
	State           string
	Description     string
	PolicyOutcome   string
	EnforcementMode string
}

// PendingCommitStatusEvaluation returns the pending scan status.
func PendingCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:         CommitStatePending,
		Description:   "Repository-Detective scan started",
		PolicyOutcome: "",
	}
}

// SkippedCommitStatusEvaluation returns the skip reason when no SHA is available.
func SkippedCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:         CommitStateSuccess,
		Description:   "Repository Detective policy evaluation skipped: no commit SHA",
		PolicyOutcome: "",
	}
}

// AnalysisFailedCommitStatusEvaluation returns status when analysis fails.
func AnalysisFailedCommitStatusEvaluation() CommitStatusEvaluation {
	return CommitStatusEvaluation{
		State:         CommitStateError,
		Description:   "Policy evaluation incomplete — analysis did not finish",
		PolicyOutcome: "EVALUATION_INCOMPLETE",
	}
}

// EvaluateCommitStatus computes final commit status from issue severities and scanner results.
// Uses Enforce semantics for raw evaluation; prefer EvaluateCommitStatusForPolicy for repo modes.
func EvaluateCommitStatus(severities []string, scannerResults []ScannerResultSummary, cfg ChecksConfig) CommitStatusEvaluation {
	eval, _ := EvaluatePolicyOutcome(severities, scannerResults, cfg, "gate_pr", cfg.FailOn, "")
	return eval
}

// EvaluateCommitStatusForPolicy applies per-repo policy level (Observe/Warn/Enforce) to evaluation.
func EvaluateCommitStatusForPolicy(severities []string, scannerResults []ScannerResultSummary, cfg ChecksConfig, policyLevel, severityGate string) CommitStatusEvaluation {
	eval, _ := EvaluatePolicyOutcome(severities, scannerResults, cfg, policyLevel, severityGate, "")
	if isRemediationPolicyLevel(policyLevel) {
		return eval
	}
	return eval
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
		return "Policy met — no gated findings"
	}
	return "Action required — findings: " + strings.Join(parts, ", ")
}

func hasBadScannerFailure(results []ScannerResultSummary) bool {
	for _, result := range results {
		status := strings.ToLower(strings.TrimSpace(result.Status))
		if result.Required {
			switch status {
			case "failed", "timed_out", "parse_failed", "binary_missing", "scanner_unavailable":
				return true
			}
			continue
		}
		switch status {
		case "failed", "timed_out", "parse_failed":
			return true
		}
	}
	return false
}

func shouldFailCommitStatus(policyLevel string) bool {
	switch strings.ToLower(strings.TrimSpace(policyLevel)) {
	case "gate_pr", "suggest_fix", "auto_pr_with_approval", "auto_pr_low_risk":
		return true
	default:
		return false
	}
}

func isRemediationPolicyLevel(policyLevel string) bool {
	switch strings.ToLower(strings.TrimSpace(policyLevel)) {
	case "suggest_fix", "auto_pr_with_approval", "auto_pr_low_risk":
		return true
	default:
		return false
	}
}

// IsRemediationPolicyLevel reports reserved remediation policy levels.
func IsRemediationPolicyLevel(policyLevel string) bool {
	return isRemediationPolicyLevel(policyLevel)
}
