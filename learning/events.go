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
)

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
