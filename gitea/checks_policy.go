package gitea

import (
	"strings"
)

// EvaluateCommitStatusForPolicy applies per-repo policy level to commit status evaluation.
func EvaluateCommitStatusForPolicy(severities []string, scannerResults []ScannerResultSummary, cfg ChecksConfig, policyLevel, severityGate string) CommitStatusEvaluation {
	policyCfg := cfg
	if severityGate != "" {
		policyCfg.FailOn = strings.ToLower(severityGate)
	}

	eval := EvaluateCommitStatus(severities, scannerResults, policyCfg)

	if isRemediationPolicyLevel(policyLevel) {
		return eval
	}

	if !shouldFailCommitStatus(policyLevel) {
		switch eval.State {
		case CommitStateFailure:
			eval.State = CommitStateWarning
			if !strings.Contains(eval.Description, "non-blocking") {
				eval.Description = "Repository Detective (non-blocking): " + strings.TrimPrefix(eval.Description, "Bugbot ")
			}
		case CommitStateError:
			if !cfg.IncludeScannerFailures {
				eval.State = CommitStateWarning
				eval.Description = "Repository Detective scan completed with warnings"
			}
		}
	}

	return eval
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

func IsRemediationPolicyLevel(policyLevel string) bool {
	return isRemediationPolicyLevel(policyLevel)
}
