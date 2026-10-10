package notify

import (
	"context"
	"time"
)

const (
	EventScanFailed                = "scan_failed"
	EventScanCompletedWithFindings = "scan_completed_with_findings"
	EventCriticalFinding           = "critical_finding"
	EventHighFinding               = "high_finding"
	EventPRGateFailed              = "pr_gate_failed"
	EventScheduledScanFailed       = "scheduled_scan_failed"
	EventRunnerJobFailed           = "runner_job_failed"
	EventRunnerJobExpired          = "runner_job_expired"
	EventPreinstallDoNotInstall    = "preinstall_do_not_install"
	EventPreinstallCaution         = "preinstall_caution"
	EventDisclosureReportGenerated = "disclosure_report_generated"
	EventFixPRMerged               = "fix_pr_merged"
	EventClosureVerified           = "closure_verified"
	EventClosureBlocked            = "closure_blocked"
	EventRemediationStillPresent   = "remediation_still_present"
	EventTest                      = "test"
)

// DefaultEnabledEvents are notification types enabled when no explicit filter is set.
var DefaultEnabledEvents = []string{
	EventCriticalFinding,
	EventHighFinding,
	EventPRGateFailed,
	EventScanFailed,
	EventRunnerJobFailed,
	EventPreinstallDoNotInstall,
}

// AllEvents lists every supported notification event type.
var AllEvents = []string{
	EventScanFailed,
	EventScanCompletedWithFindings,
	EventCriticalFinding,
	EventHighFinding,
	EventPRGateFailed,
	EventScheduledScanFailed,
	EventRunnerJobFailed,
	EventRunnerJobExpired,
	EventPreinstallDoNotInstall,
	EventPreinstallCaution,
	EventDisclosureReportGenerated,
	EventFixPRMerged,
	EventClosureVerified,
	EventClosureBlocked,
	EventRemediationStillPresent,
}

// Event is a sanitized notification payload.
type Event struct {
	Type       string
	Severity   string
	Repository string
	ScanID     string
	FindingID  string
	Title      string
	Summary    string
	Counts     map[string]int
	URL        string
	CreatedAt  time.Time
}

// Message is the formatted text sent to channels.
type Message struct {
	Text    string
	Event   Event
	Channel string
}

// Channel delivers notifications to an external service.
type Channel interface {
	Name() string
	Enabled() bool
	Send(ctx context.Context, msg Message) error
}

// HTTPPoster performs HTTP POST requests (injectable for tests).
type HTTPPoster interface {
	Post(ctx context.Context, url string, contentType string, body []byte, headers map[string]string) (int, error)
}

// SettingsResolver returns effective notification settings for a repository.
type SettingsResolver func(repositoryID int64) EffectiveSettings
