package store

import "strings"

// AllowedNotificationEvents lists valid per-repo notification event filters.
var AllowedNotificationEvents = []string{
	"scan_failed",
	"scan_completed_with_findings",
	"critical_finding",
	"high_finding",
	"pr_gate_failed",
	"scheduled_scan_failed",
	"runner_job_failed",
	"runner_job_expired",
	"preinstall_do_not_install",
	"preinstall_caution",
	"disclosure_report_generated",
}

// ValidateNotificationEventsCSV validates comma-separated notification event names.
func ValidateNotificationEventsCSV(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if !containsString(AllowedNotificationEvents, p) {
			return errInvalidNotificationEvent(p)
		}
	}
	return nil
}

func errInvalidNotificationEvent(name string) error {
	return &notificationEventError{name: name}
}

type notificationEventError struct{ name string }

func (e *notificationEventError) Error() string {
	return "invalid notification_events entry " + quote(e.name)
}

func quote(s string) string { return `"` + s + `"` }
