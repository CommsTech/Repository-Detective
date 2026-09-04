package store

import "strings"

// Enforcement modes are operator-facing aliases over policy_level.
// Stored values remain monitor_only / issue_only / gate_pr for compatibility.
const (
	EnforcementObserve = "observe"
	EnforcementWarn    = "warn"
	EnforcementEnforce = "enforce"
)

// EnforcementModeFromPolicyLevel maps stored policy_level to Observe/Warn/Enforce.
func EnforcementModeFromPolicyLevel(policyLevel string) string {
	switch normalizePolicyLevel(policyLevel) {
	case PolicyMonitorOnly:
		return EnforcementObserve
	case PolicyGatePR, PolicySuggestFix, PolicyAutoPRWithApproval, PolicyAutoPRLowRisk:
		return EnforcementEnforce
	case PolicyIssueOnly:
		return EnforcementWarn
	default:
		return EnforcementWarn
	}
}

// PolicyLevelFromEnforcementMode maps Observe/Warn/Enforce to stored policy_level.
func PolicyLevelFromEnforcementMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case EnforcementObserve, "observation", "observation_only":
		return PolicyMonitorOnly
	case EnforcementEnforce, "gate", "blocking":
		return PolicyGatePR
	case EnforcementWarn, "warning":
		return PolicyIssueOnly
	default:
		return ""
	}
}

// EnforcementModeLabel returns a short UI label for a policy_level value.
func EnforcementModeLabel(policyLevel string) string {
	switch EnforcementModeFromPolicyLevel(policyLevel) {
	case EnforcementObserve:
		return "Observe — findings only; never blocks workflow"
	case EnforcementEnforce:
		return "Enforce — commit status may block when branch protection requires it"
	default:
		return "Warn — surface policy violations; merge stays possible unless forge rules say otherwise"
	}
}

// PolicyOutcome values describe owner-defined policy evaluation — never "secure"/"safe".
const (
	PolicyOutcomeMet               = "POLICY_MET"
	PolicyOutcomeActionRequired    = "ACTION_REQUIRED"
	PolicyOutcomeEvaluationIncomplete = "EVALUATION_INCOMPLETE"
	PolicyOutcomeObservationOnly   = "OBSERVATION_ONLY"
)
