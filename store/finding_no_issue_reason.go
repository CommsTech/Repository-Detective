package store

// FindingNoIssueReason explains why an open finding has no mapped forge issue.
type FindingNoIssueReason string

const (
	NoIssueReasonAlreadyMapped   FindingNoIssueReason = "already_mapped"
	NoIssueReasonReportOnly      FindingNoIssueReason = "report_only"
	NoIssueReasonFilingDisabled  FindingNoIssueReason = "filing_disabled"
	NoIssueReasonNoIssuePolicy   FindingNoIssueReason = "no_issue_policy"
	NoIssueReasonBelowThreshold  FindingNoIssueReason = "below_threshold"
	NoIssueReasonSuppressed      FindingNoIssueReason = "suppressed"
	NoIssueReasonDuplicate       FindingNoIssueReason = "duplicate"
	NoIssueReasonForgeError      FindingNoIssueReason = "forge_error"
	NoIssueReasonUnknown         FindingNoIssueReason = "unknown"
)

// ClassifyFindingNoIssueReason returns why a finding lacks a mapped open forge issue.
// When hasOpenMappedIssue is true, the finding is already linked.
func ClassifyFindingNoIssueReason(
	findingStatus string,
	hasOpenMappedIssue bool,
	effective EffectiveSettings,
	scanDryRun bool,
	issueSyncStatus string,
) FindingNoIssueReason {
	if hasOpenMappedIssue {
		return NoIssueReasonAlreadyMapped
	}
	if findingStatus == FindingStatusSuppressed || findingStatus == FindingStatusFalsePositive {
		return NoIssueReasonSuppressed
	}
	if scanDryRun {
		return NoIssueReasonReportOnly
	}
	if !ShouldCreateForgeIssues(effective) {
		if effective.PolicyLevel == PolicyMonitorOnly || effective.IssuePolicy == IssuePolicyOff {
			return NoIssueReasonNoIssuePolicy
		}
		return NoIssueReasonFilingDisabled
	}
	if issueSyncStatus == IssueSyncStatusFailed {
		return NoIssueReasonForgeError
	}
	if issueSyncStatus == IssueSyncStatusSkipped {
		return NoIssueReasonReportOnly
	}
	return NoIssueReasonUnknown
}

// ScheduleEligible reports whether a repo can be picked up by the in-process scheduler.
func ScheduleEligible(connected bool, effective EffectiveSettings) (eligible bool, reason string) {
	if !connected {
		return false, "not_connected"
	}
	if !effective.Enabled {
		return false, "scan_disabled"
	}
	if !effective.ScheduleEnabled {
		return false, "schedule_disabled"
	}
	cron := effective.ScheduleCron
	if cron == "" {
		return false, "missing_schedule_cron"
	}
	return true, ""
}
