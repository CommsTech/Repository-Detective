package issues

import (
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// Lifecycle label constants (Repository Detective namespace).
const (
	LifecycleOpen                 = "repository-detective/open"
	LifecycleStillPresent         = "repository-detective/still-present"
	LifecycleNotReproduced        = "repository-detective/not-reproduced"
	LifecycleFixed                = "repository-detective/fixed"
	LifecycleFalsePositive        = "repository-detective/false-positive"
	LifecycleSuppressed           = "repository-detective/suppressed"
	LifecycleNeedsHumanReview     = "repository-detective/needs-human-review"
	LifecycleRemediationCandidate = "repository-detective/remediation-candidate"
	LifecycleFixPROpened          = "repository-detective/fix-pr-opened"
	LifecycleFixPRMerged          = "repository-detective/fix-pr-merged"
	LifecyclePendingRescan        = "repository-detective/pending-rescan"
	LifecycleResolvedVerified     = "repository-detective/resolved-verified"
	LifecycleClosureBlocked       = "repository-detective/closure-blocked"
	LifecycleDuplicate            = "repository-detective/duplicate"
)

// StillPresentCommentBody formats an update when a fingerprint is detected again.
func StillPresentCommentBody(issue *ai.CodeIssue, scanID string) string {
	var b strings.Builder
	b.WriteString("Repository Detective detected this finding again in scan `" + scanID + "`.\n\n")
	b.WriteString("**Status:** still present\n")
	b.WriteString(fmt.Sprintf("**Severity:** %s\n", issue.Severity))
	b.WriteString(fmt.Sprintf("**Confidence:** %.2f\n", issue.Confidence))
	if issue.File != "" {
		b.WriteString(fmt.Sprintf("**Location:** `%s`", issue.File))
		if issue.LineNumber > 0 {
			b.WriteString(fmt.Sprintf(" line %d", issue.LineNumber))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nLast seen: " + time.Now().Format(time.RFC3339))
	return b.String()
}

// NotReproducedCommentBody formats a comment when a prior finding was absent in a scan.
func NotReproducedCommentBody(scanID string) string {
	return fmt.Sprintf(
		"Repository Detective did not reproduce this finding in scan `%s`.\n\nThe issue remains open for manual verification.",
		scanID,
	)
}

// NeedsHumanReviewCommentBody formats a low-confidence update without creating noise.
func NeedsHumanReviewCommentBody(issue *ai.CodeIssue, scanID string) string {
	return fmt.Sprintf(
		"Repository Detective confidence dropped or is borderline (%.2f) in scan `%s`. Please review manually before acting.\n\n**Finding:** %s",
		issue.Confidence,
		scanID,
		issue.Title,
	)
}

// ConfidenceNeedsHumanReview reports whether confidence is low enough to flag manual review.
func ConfidenceNeedsHumanReview(confidence float64) bool {
	return confidence > 0 && confidence < 0.65
}
