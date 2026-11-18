package notify

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// ValidateEventCSV returns an error if any token is unknown.
func ValidateEventCSV(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if !IsValidEventType(p) {
			return fmt.Errorf("invalid notification event %q", p)
		}
	}
	return nil
}

// ValidateMinSeverity returns an error for invalid severity thresholds.
func ValidateMinSeverity(sev string) error {
	sev = strings.ToLower(strings.TrimSpace(sev))
	if sev == "" {
		return nil
	}
	if !storeContainsSeverity(sev) {
		return fmt.Errorf("invalid notification_min_severity %q", sev)
	}
	return nil
}

func storeContainsSeverity(sev string) bool {
	for _, s := range store.AllowedSeverities {
		if s == sev {
			return true
		}
	}
	return false
}

// FormatEventCSV serializes enabled events to CSV for storage.
func FormatEventCSV(events map[string]bool) string {
	if len(events) == 0 {
		return ""
	}
	var parts []string
	for _, e := range AllEvents {
		if events[e] {
			parts = append(parts, e)
		}
	}
	return strings.Join(parts, ",")
}
