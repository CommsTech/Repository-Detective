package closure

import (
	"database/sql"
	"errors"
	"strings"
)

const directScanMergeMarker = "direct-scan-verified"

// IsEvidenceNotFound reports whether closure evidence has never been recorded for a finding.
func IsEvidenceNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "not found")
}
