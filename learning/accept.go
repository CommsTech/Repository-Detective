package learning

import (
	"fmt"
	"strings"
)

// ValidateCalibrationAccept checks whether a calibration recommendation may be accepted.
// Severity is not used here: high/critical protection is enforced again at scan persist time.
// Category protection blocks secrets/security classes from becoming suppressions via accept.
// Global-scope recommendations are allowed: accept expands them into repo-scoped rules
// (never a fleet-wide global suppression).
func ValidateCalibrationAccept(category, scope string) error {
	_ = scope // retained for callers / future scope policy
	if IsProtectedFromAutoDowngrade("", category) {
		return fmt.Errorf("recommendation affects protected security category — mark findings false-positive individually or use an explicit operator override")
	}
	if strings.TrimSpace(scope) == "" {
		return fmt.Errorf("recommendation scope is required")
	}
	return nil
}
