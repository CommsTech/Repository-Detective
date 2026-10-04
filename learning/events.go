package learning

import "strings"

// Learning event types (append-only audit log).
const (
	EventUserMarkedFalsePositive = "user_marked_false_positive"
	EventUserMarkedTruePositive  = "user_marked_true_positive"
	EventResolvedVerified        = "resolved_verified"
	EventDuplicateLinked         = "duplicate_linked"
	EventScannerFailed           = "scanner_failed"
	EventScannerRecovered        = "scanner_recovered"
	EventFindingReappeared       = "finding_reappeared"
	EventRemediationFailed       = "remediation_failed"
	EventRemediationSucceeded    = "remediation_succeeded"
	EventReportOnlyDryRun        = "report_only_dry_run"
	EventIssueClosed             = "issue_closed"
	EventIssueReopened           = "issue_reopened"
	EventOperatorOverride        = "operator_override"
	EventRecommendationAccepted  = "recommendation_accepted"
	EventRecommendationRejected  = "recommendation_rejected"
	// Cost / reliability signals learned from the 2026-10-02 OpenClaw auditor flood.
	EventLLMAuditorTimeout           = "llm_auditor_timeout"
	EventLLMAuditorEmptyBatch        = "llm_auditor_empty_batch"
	EventAIAdvisoryFailed            = "ai_advisory_failed"
	EventAIAdvisoryCalibrateSuggested = "ai_advisory_calibrate_suggested"
)

// DispositionEventTypes are events that count toward FP/TP calibration math.
// Operational noise (scanner_failed, dry-run, auditor timeouts) must not dilute rates.
var DispositionEventTypes = []string{
	EventUserMarkedFalsePositive,
	EventUserMarkedTruePositive,
	EventResolvedVerified,
	EventAIAdvisoryCalibrateSuggested,
}

// IsDispositionEvent reports whether eventType should count in FP/TP denominators.
func IsDispositionEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case EventUserMarkedFalsePositive, EventUserMarkedTruePositive,
		EventResolvedVerified, EventAIAdvisoryCalibrateSuggested:
		return true
	default:
		return false
	}
}

// IsSoftFalsePositiveEvidence reports weak FP evidence (AI calibrate suggestions).
func IsSoftFalsePositiveEvidence(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case EventUserMarkedFalsePositive, EventAIAdvisoryCalibrateSuggested:
		return true
	default:
		return false
	}
}

var protectedSeverities = map[string]bool{"critical": true, "high": true}

var protectedCategories = map[string]bool{
	"secrets": true, "secret": true, "hardcoded_secret": true,
	"dependency_vulnerability": true, "security": true,
}

// IsProtectedFromAutoDowngrade reports whether automatic calibration must not apply.
func IsProtectedFromAutoDowngrade(severity, category string) bool {
	sev := strings.ToLower(strings.TrimSpace(severity))
	if sev != "" && protectedSeverities[sev] {
		return true
	}
	cat := strings.ToLower(strings.TrimSpace(category))
	if cat == "" {
		return false
	}
	for k := range protectedCategories {
		if strings.Contains(cat, k) {
			return true
		}
	}
	return false
}
