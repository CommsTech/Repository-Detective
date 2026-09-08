package issues

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// Lifecycle label constants (legacy Repository Detective namespace — mapped via ExpandLifecycleLabels).
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

const agingReminderDays = 14

var (
	bodySeverityRe = regexp.MustCompile(`(?i)\*\*Severity:\*\*\s*([A-Za-z]+)`)
	bodyLocationRe = regexp.MustCompile("(?m)^`([^`]+)`\\s*$")
	bodyLastSeenRe = regexp.MustCompile(`(?i)\*\*Last seen:\*\*\s*([^\n]+)`)
)

// UpdateDecision describes whether a rescan should comment on an existing forge issue.
type UpdateDecision struct {
	Comment bool
	Body    string
	Labels  []string
	Reason  string // still_present_silent | severity_changed | location_changed | needs_review | aging_reminder
}

// DecideExistingIssueUpdate suppresses routine "still present" noise.
// Comments only for meaningful changes (severity/location/confidence/aging).
func DecideExistingIssueUpdate(issue *ai.CodeIssue, match *ExistingIssueMatch, scanID string, now time.Time) UpdateDecision {
	if issue == nil {
		return UpdateDecision{Reason: "still_present_silent"}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	prevBody := ""
	if match != nil {
		prevBody = match.Body
	}

	if ConfidenceNeedsHumanReview(issue.Confidence) {
		return UpdateDecision{
			Comment: true,
			Body:    NeedsHumanReviewCommentBody(issue, scanID),
			Labels:  ExpandLifecycleLabels(LifecycleNeedsHumanReview),
			Reason:  "needs_review",
		}
	}

	prevSev := strings.ToLower(extractBodySeverity(prevBody))
	curSev := strings.ToLower(strings.TrimSpace(issue.Severity))
	if prevSev != "" && curSev != "" && prevSev != curSev {
		return UpdateDecision{
			Comment: true,
			Body:    MeaningfulChangeCommentBody(issue, scanID, fmt.Sprintf("Severity changed from **%s** to **%s**.", prevSev, curSev)),
			Labels:  []string{SeverityLabel(issue.Severity)},
			Reason:  "severity_changed",
		}
	}

	prevLoc := extractBodyLocation(prevBody)
	curLoc := locationRef(issue)
	if prevLoc != "" && curLoc != "" && prevLoc != curLoc {
		return UpdateDecision{
			Comment: true,
			Body:    MeaningfulChangeCommentBody(issue, scanID, fmt.Sprintf("Location changed from `%s` to `%s`.", prevLoc, curLoc)),
			Reason:  "location_changed",
		}
	}

	if lastSeen, ok := extractBodyLastSeen(prevBody); ok {
		if now.Sub(lastSeen) >= agingReminderDays*24*time.Hour {
			return UpdateDecision{
				Comment: true,
				Body: fmt.Sprintf(
					"Repository Detective aging reminder: this finding has remained open for %d+ days (scan `%s`).\n\n**Location:** `%s`\n**Severity:** %s",
					agingReminderDays, scanID, curLoc, issue.Severity,
				),
				Reason: "aging_reminder",
			}
		}
	}

	// Silent: last-seen is tracked in Repository Detective; do not comment every nightly scan.
	return UpdateDecision{Reason: "still_present_silent"}
}

func extractBodySeverity(body string) string {
	m := bodySeverityRe.FindStringSubmatch(body)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractBodyLocation(body string) string {
	m := bodyLocationRe.FindStringSubmatch(body)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractBodyLastSeen(body string) (time.Time, bool) {
	m := bodyLastSeenRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(m[1])
	for _, layout := range []string{"Jan 2, 2006", time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// StillPresentCommentBody formats an update when a fingerprint is detected again (legacy; prefer DecideExistingIssueUpdate).
func StillPresentCommentBody(issue *ai.CodeIssue, scanID string) string {
	return MeaningfulChangeCommentBody(issue, scanID, "Finding still present on re-scan.")
}

// MeaningfulChangeCommentBody posts only when something operator-relevant changed.
func MeaningfulChangeCommentBody(issue *ai.CodeIssue, scanID, change string) string {
	var b strings.Builder
	b.WriteString(change + "\n\n")
	b.WriteString(fmt.Sprintf("Scan: `%s`\n", scanID))
	if issue != nil {
		b.WriteString(fmt.Sprintf("**Severity:** %s\n", issue.Severity))
		if loc := locationRef(issue); loc != "" {
			b.WriteString(fmt.Sprintf("**Location:** `%s`\n", loc))
		}
	}
	b.WriteString("\nLast seen: " + time.Now().UTC().Format(time.RFC3339))
	return b.String()
}

// NotReproducedCommentBody formats a comment when a prior finding was absent in a scan.
func NotReproducedCommentBody(scanID string) string {
	return fmt.Sprintf(
		"Repository Detective did not reproduce this finding in scan `%s`.\n\nThe issue remains open for manual verification. "+
			"Repository Detective will only close when the fingerprint stays absent or you triage it as fixed/false positive.",
		scanID,
	)
}

// NeedsHumanReviewCommentBody formats a low-confidence update without creating noise.
func NeedsHumanReviewCommentBody(issue *ai.CodeIssue, scanID string) string {
	title := ""
	if issue != nil {
		title = issue.Title
	}
	conf := 0.0
	if issue != nil {
		conf = issue.Confidence
	}
	return fmt.Sprintf(
		"Repository Detective confidence is borderline (%.0f%%) in scan `%s`. Please triage manually before acting.\n\n**Finding:** %s",
		conf*100,
		scanID,
		title,
	)
}

// ConfidenceNeedsHumanReview reports whether confidence is low enough to flag manual review.
func ConfidenceNeedsHumanReview(confidence float64) bool {
	return confidence > 0 && confidence < 0.65
}
