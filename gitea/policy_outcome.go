package gitea

import (
	"fmt"
	"strings"
)

// Policy outcome constants (mirrored in store for API/UI; avoid import cycles).
const (
	PolicyOutcomeMet                  = "POLICY_MET"
	PolicyOutcomeActionRequired       = "ACTION_REQUIRED"
	PolicyOutcomeEvaluationIncomplete = "EVALUATION_INCOMPLETE"
	PolicyOutcomeObservationOnly      = "OBSERVATION_ONLY"

	EnforcementObserve = "observe"
	EnforcementWarn    = "warn"
	EnforcementEnforce = "enforce"
)

// PolicyEvaluation is a deterministic, explainable policy decision for a scan.
// It never claims the repository is safe or secure.
type PolicyEvaluation struct {
	Outcome                    string   `json:"outcome"`
	EnforcementMode            string   `json:"enforcement_mode"`
	PolicyLevel                string   `json:"policy_level"`
	PolicyName                 string   `json:"policy_name"`
	PolicyVersion              string   `json:"policy_version"`
	ViolatedConditions         []string `json:"violated_conditions,omitempty"`
	EvaluatedConditions        []string `json:"evaluated_conditions,omitempty"`
	RequiredScannersIncomplete []string `json:"required_scanners_incomplete,omitempty"`
	CoverageSummary            string   `json:"coverage_summary,omitempty"`
	CommitSHA                  string   `json:"commit_sha,omitempty"`
	Description                string   `json:"description"`
}

const (
	defaultPolicyName    = "repository-detective/owner-policy"
	defaultPolicyVersion = "1"
)

func enforcementModeFromPolicyLevel(policyLevel string) string {
	switch strings.ToLower(strings.TrimSpace(policyLevel)) {
	case "monitor_only":
		return EnforcementObserve
	case "gate_pr", "suggest_fix", "auto_pr_with_approval", "auto_pr_low_risk":
		return EnforcementEnforce
	default:
		return EnforcementWarn
	}
}

func classifyCoverageStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "clean", "found", "success":
		return "SUCCESS"
	case "failed", "parse_failed":
		return "FAILED"
	case "timed_out", "timeout":
		return "TIMEOUT"
	case "binary_missing", "scanner_unavailable":
		return "UNAVAILABLE"
	case "disabled":
		return "SKIPPED_BY_POLICY"
	case "no_supported_manifest":
		return "NOT_APPLICABLE"
	default:
		if status == "" {
			return "UNAVAILABLE"
		}
		return "FAILED"
	}
}

func coverageBlocksPolicyMet(state string) bool {
	switch state {
	case "FAILED", "TIMEOUT", "UNAVAILABLE", "SKIPPED_BY_POLICY":
		return true
	default:
		return false
	}
}

// EvaluatePolicyOutcome derives POLICY_* from commit-status inputs and enforcement mode.
func EvaluatePolicyOutcome(severities []string, scannerResults []ScannerResultSummary, cfg ChecksConfig, policyLevel, severityGate, commitSHA string) (CommitStatusEvaluation, PolicyEvaluation) {
	mode := enforcementModeFromPolicyLevel(policyLevel)
	policyCfg := cfg
	if severityGate != "" {
		policyCfg.FailOn = strings.ToLower(severityGate)
	}

	pe := PolicyEvaluation{
		EnforcementMode: mode,
		PolicyLevel:     policyLevel,
		PolicyName:      defaultPolicyName,
		PolicyVersion:   defaultPolicyVersion,
		CommitSHA:       commitSHA,
		EvaluatedConditions: []string{
			"required_scanners_completed",
			"severity_gate:" + normalizeSeverityThreshold(policyCfg.FailOn, "high"),
			"warn_gate:" + normalizeSeverityThreshold(policyCfg.WarnOn, "medium"),
			"enforcement_mode:" + mode,
		},
	}

	incomplete := requiredScannerIncomplete(scannerResults)
	pe.RequiredScannersIncomplete = incomplete
	pe.CoverageSummary = formatRequiredCoverage(scannerResults)

	if len(incomplete) > 0 && (cfg.IncludeScannerFailures || hasRequiredUnavailable(scannerResults)) {
		pe.Outcome = PolicyOutcomeEvaluationIncomplete
		pe.ViolatedConditions = append(pe.ViolatedConditions, "required_scanner_incomplete")
		pe.Description = fmt.Sprintf("Policy evaluation incomplete — %s", strings.Join(incomplete, ", "))
		eval := CommitStatusEvaluation{
			State:           CommitStateError,
			Description:     pe.Description,
			PolicyOutcome:   pe.Outcome,
			EnforcementMode: mode,
		}
		if mode == EnforcementObserve {
			eval.State = CommitStateSuccess
			pe.Outcome = PolicyOutcomeObservationOnly
			pe.Description = "Observation only — required analyzer coverage incomplete (non-blocking): " + strings.Join(incomplete, ", ")
			eval.Description = pe.Description
			eval.PolicyOutcome = pe.Outcome
		} else if mode == EnforcementWarn {
			eval.State = CommitStateWarning
			eval.Description = "Policy evaluation incomplete (non-blocking): " + strings.Join(incomplete, ", ")
		}
		return eval, pe
	}

	if cfg.IncludeScannerFailures && hasOptionalBadScannerFailure(scannerResults) {
		pe.Outcome = PolicyOutcomeEvaluationIncomplete
		pe.ViolatedConditions = append(pe.ViolatedConditions, "optional_scanner_failure_included")
		pe.Description = "Policy evaluation incomplete — scanner failures reported"
		eval := CommitStatusEvaluation{
			State:           CommitStateError,
			Description:     pe.Description,
			PolicyOutcome:   pe.Outcome,
			EnforcementMode: mode,
		}
		if mode != EnforcementEnforce {
			eval.State = CommitStateWarning
			eval.Description = "Repository Detective (non-blocking): scanner failures reported"
		}
		return eval, pe
	}

	counts := countSeverities(severities)
	failOn := normalizeSeverityThreshold(policyCfg.FailOn, "high")
	warnOn := normalizeSeverityThreshold(policyCfg.WarnOn, "medium")

	if mode == EnforcementObserve {
		pe.Outcome = PolicyOutcomeObservationOnly
		pe.Description = "Observation only — policy not enforced; findings recorded when present"
		if hasSeverityAtOrAbove(severities, failOn) || hasSeverityAtOrAbove(severities, warnOn) {
			pe.Description = "Observation only — findings present; workflow not blocked by Repository Detective"
			pe.ViolatedConditions = append(pe.ViolatedConditions, "findings_observed")
		}
		return CommitStatusEvaluation{
			State:           CommitStateSuccess,
			Description:     pe.Description,
			PolicyOutcome:   pe.Outcome,
			EnforcementMode: mode,
		}, pe
	}

	if hasSeverityAtOrAbove(severities, failOn) {
		pe.Outcome = PolicyOutcomeActionRequired
		pe.ViolatedConditions = append(pe.ViolatedConditions, "severity_gate_violated:"+failOn)
		pe.Description = "Action required — owner policy conditions violated: " + formatFindingCounts(counts)
		eval := CommitStatusEvaluation{
			State:           CommitStateFailure,
			Description:     pe.Description,
			PolicyOutcome:   pe.Outcome,
			EnforcementMode: mode,
		}
		if mode == EnforcementWarn {
			eval.State = CommitStateWarning
			eval.Description = "Action required (non-blocking): " + formatFindingCounts(counts)
		}
		return eval, pe
	}

	if hasSeverityAtOrAbove(severities, warnOn) {
		pe.Outcome = PolicyOutcomeActionRequired
		pe.ViolatedConditions = append(pe.ViolatedConditions, "warn_gate_violated:"+warnOn)
		pe.Description = "Action required — advisory policy conditions: " + formatFindingCounts(counts)
		return CommitStatusEvaluation{
			State:           CommitStateWarning,
			Description:     pe.Description,
			PolicyOutcome:   pe.Outcome,
			EnforcementMode: mode,
		}, pe
	}

	pe.Outcome = PolicyOutcomeMet
	pe.Description = "Policy met — required analysis completed; configured policy conditions were not violated"
	return CommitStatusEvaluation{
		State:           CommitStateSuccess,
		Description:     pe.Description,
		PolicyOutcome:   pe.Outcome,
		EnforcementMode: mode,
	}, pe
}

func requiredScannerIncomplete(results []ScannerResultSummary) []string {
	var out []string
	for _, result := range results {
		if !result.Required {
			continue
		}
		state := classifyCoverageStatus(result.Status)
		if coverageBlocksPolicyMet(state) {
			out = append(out, fmt.Sprintf("%s (%s)", result.Scanner, state))
		}
	}
	return out
}

func hasRequiredUnavailable(results []ScannerResultSummary) bool {
	for _, result := range results {
		if result.Required && coverageBlocksPolicyMet(classifyCoverageStatus(result.Status)) {
			return true
		}
	}
	return false
}

func hasOptionalBadScannerFailure(results []ScannerResultSummary) bool {
	for _, result := range results {
		if result.Required {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(result.Status)) {
		case "failed", "timed_out", "parse_failed":
			return true
		}
	}
	return false
}

func formatRequiredCoverage(results []ScannerResultSummary) string {
	total := 0
	ok := 0
	for _, result := range results {
		if !result.Required {
			continue
		}
		total++
		if !coverageBlocksPolicyMet(classifyCoverageStatus(result.Status)) {
			ok++
		}
	}
	if total == 0 {
		return "0/0 required analyzers completed"
	}
	return fmt.Sprintf("%d/%d required analyzers completed", ok, total)
}

func formatFindingCounts(counts map[string]int) string {
	parts := make([]string, 0, 5)
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		if counts[severity] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[severity], severity))
		}
	}
	if len(parts) == 0 {
		return "no gated findings"
	}
	return strings.Join(parts, ", ")
}
