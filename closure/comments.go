package closure

import (
	"fmt"
	"strings"
)

// PRMergedComment returns the issue comment when a remediation PR is merged.
func PRMergedComment() string {
	return "Remediation PR merged. Repository Detective is waiting for a follow-up scan to verify the finding is gone."
}

// VerifiedComment returns the issue comment when closure is verified.
func VerifiedComment(scanID, fingerprint, scanner string) string {
	return fmt.Sprintf(
		"Resolution verified by scan `%s`. Fingerprint `%s` was not reproduced, and `%s` completed successfully.",
		sanitize(scanID), sanitize(fingerprint), sanitize(scanner),
	)
}

// BlockedComment returns the issue comment when closure is blocked.
func BlockedComment(scanner, scanID string) string {
	return fmt.Sprintf(
		"Closure blocked: fingerprint was absent, but the original scanner `%s` did not complete successfully in scan `%s`.",
		sanitize(scanner), sanitize(scanID),
	)
}

// StillPresentComment returns the issue comment when remediation did not resolve the finding.
func StillPresentComment(fingerprint, scanID string) string {
	return fmt.Sprintf(
		"Remediation did not resolve the finding. Fingerprint `%s` was still present in scan `%s`.",
		sanitize(fingerprint), sanitize(scanID),
	)
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 256 {
		s = s[:256] + "…"
	}
	return s
}
