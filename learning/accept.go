package learning

import (
	"fmt"
	"strings"
)

// ValidateCalibrationAccept checks whether a calibration recommendation may be accepted.
// Severity is not used here: high/critical protection is enforced again at scan persist time.
// Category protection blocks secrets/security classes from becoming suppressions via accept.
// Global-scope accepts are blocked in community beta — operators use repo-scoped recommendations.
func ValidateCalibrationAccept(category, scope string) error {
	if IsProtectedFromAutoDowngrade("", category) {
		return fmt.Errorf("recommendation affects protected security category — mark findings false-positive individually or use an explicit operator override")
	}
	if strings.EqualFold(strings.TrimSpace(scope), "global") {
		return fmt.Errorf("global calibration recommendations require multi-repo evidence review — use repo-scoped recommendations")
	}
	return nil
}
