package issues

import (
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
)

// Lifecycle label constants.
const (
	LifecycleOpen                = "bugbot/open"
	LifecycleStillPresent        = "bugbot/still-present"
	LifecycleNotReproduced       = "bugbot/not-reproduced"
	LifecycleFixed               = "bugbot/fixed"
	LifecycleFalsePositive       = "bugbot/false-positive"
	LifecycleNeedsHumanReview    = "bugbot/needs-human-review"
	LifecycleRemediationCandidate = "bugbot/remediation-candidate"
	LifecycleFixPROpened         = "bugbot/fix-pr-opened"
	LifecycleFixPRMerged         = "bugbot/fix-pr-merged"
	LifecyclePendingRescan       = "bugbot/pending-rescan"
	LifecycleResolvedVerified    = "bugbot/resolved-verified"
	LifecycleClosureBlocked      = "bugbot/closure-blocked"
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
