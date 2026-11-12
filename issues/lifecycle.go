package issues

import (
	"fmt"
	"math"
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

// Aging stages advance once each; after long-lived, stay silent unless other signals fire.
const (
	AgingStageNone      = ""
	AgingStageAging     = "aging"      // 14d
	AgingStageOverdue   = "overdue"    // 30d
	AgingStageStale     = "stale"      // 60d
	AgingStageLongLived = "long-lived" // 90d — final escalation
)

const (
	agingDaysAging     = 14
	agingDaysOverdue   = 30
	agingDaysStale     = 60
	agingDaysLongLived = 90
	// ConfidenceCommentDelta is the minimum absolute confidence change that warrants a comment.
	ConfidenceCommentDelta = 0.15
)

var (
	bodySeverityRe   = regexp.MustCompile(`(?i)\*\*Severity:\*\*\s*([A-Za-z]+)`)
	bodyLocationRe   = regexp.MustCompile("(?m)^`([^`]+)`\\s*$")
	bodyLastSeenRe   = regexp.MustCompile(`(?i)\*\*Last seen:\*\*\s*([^\n]+)`)
	bodyFirstSeenRe  = regexp.MustCompile(`(?i)\*\*First seen:\*\*\s*([^\n]+)`)
	bodyAgingStageRe = regexp.MustCompile(`(?i)\*\*Aging:\*\*\s*([a-z-]+)`)
	bodyConfidenceRe = regexp.MustCompile(`(?i)\*\*Detection confidence:\*\*\s*([0-9.]+)%`)
)

// UpdateDecision describes whether a rescan should comment on an existing forge issue.
type UpdateDecision struct {
	Comment   bool
	Body      string
	BodyPatch string // when set, replace the forge issue body (persist aging markers)
	Labels    []string
	Reason    string
}

// DecideExistingIssueUpdate suppresses routine "still present" noise.
// Comments only when the developer should care that RD is talking again.
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

	if d := decideConfidenceComment(issue, prevBody, scanID); d.Comment {
		return d
	}

	if d := decideAgingComment(issue, prevBody, curLoc, scanID, now); d.Comment {
		return d
	}

	return UpdateDecision{Reason: "still_present_silent"}
}

func decideConfidenceComment(issue *ai.CodeIssue, prevBody, scanID string) UpdateDecision {
	prevConf, hasPrev := extractBodyConfidence(prevBody)
	cur := issue.Confidence
	crossedIntoReview := ConfidenceNeedsHumanReview(cur) && (!hasPrev || !ConfidenceNeedsHumanReview(prevConf))
	materialDrop := hasPrev && prevConf-cur >= ConfidenceCommentDelta
	if crossedIntoReview || (materialDrop && ConfidenceNeedsHumanReview(cur)) {
		return UpdateDecision{
			Comment: true,
			Body:    NeedsHumanReviewCommentBody(issue, scanID),
			Labels:  ExpandLifecycleLabels(LifecycleNeedsHumanReview),
			Reason:  "needs_review",
		}
	}
	return UpdateDecision{}
}

func decideAgingComment(issue *ai.CodeIssue, prevBody, curLoc, scanID string, now time.Time) UpdateDecision {
	firstSeen, ok := extractBodyFirstSeen(prevBody)
	if !ok {
		// Fall back to last-seen as open-age proxy for legacy bodies.
		firstSeen, ok = extractBodyLastSeen(prevBody)
	}
	if !ok {
		return UpdateDecision{}
	}
	ageDays := int(now.Sub(firstSeen).Hours() / 24)
	want := agingStageForDays(ageDays)
	if want == AgingStageNone {
		return UpdateDecision{}
	}
	prevStage := extractBodyAgingStage(prevBody)
	if !agingStageAdvances(prevStage, want) {
		return UpdateDecision{} // already at or past this stage — no spam
	}
	return UpdateDecision{
		Comment: true,
		Body: fmt.Sprintf(
			"Repository Detective aging transition: **%s** (%d days open).\n\n**Aging:** %s\nScan: `%s`\n**Location:** `%s`\n**Severity:** %s\n\nWhy you should care: this finding has remained unresolved long enough to warrant another look.",
			want, ageDays, want, scanID, curLoc, issue.Severity,
		),
		BodyPatch: UpsertAgingMarker(prevBody, want),
		Reason:    "aging_" + want,
	}
}

// UpsertAgingMarker writes or replaces **Aging:** in the issue body so stage advances persist.
func UpsertAgingMarker(body, stage string) string {
	stage = strings.TrimSpace(strings.ToLower(stage))
	if stage == "" {
		return body
	}
	line := "**Aging:** " + stage
	if bodyAgingStageRe.MatchString(body) {
		return bodyAgingStageRe.ReplaceAllString(body, line)
	}
	if bodyFirstSeenRe.MatchString(body) {
		return bodyFirstSeenRe.ReplaceAllString(body, "${0}\n"+line)
	}
	if bodyLastSeenRe.MatchString(body) {
		return bodyLastSeenRe.ReplaceAllString(body, "${0}\n"+line)
	}
	return strings.TrimRight(body, "\n") + "\n\n" + line + "\n"
}

func agingStageForDays(days int) string {
	switch {
	case days >= agingDaysLongLived:
		return AgingStageLongLived
	case days >= agingDaysStale:
		return AgingStageStale
	case days >= agingDaysOverdue:
		return AgingStageOverdue
	case days >= agingDaysAging:
		return AgingStageAging
	default:
		return AgingStageNone
	}
}

func agingStageRank(stage string) int {
	switch stage {
	case AgingStageAging:
		return 1
	case AgingStageOverdue:
		return 2
	case AgingStageStale:
		return 3
	case AgingStageLongLived:
		return 4
	default:
		return 0
	}
}

func agingStageAdvances(prev, want string) bool {
	return agingStageRank(want) > agingStageRank(prev)
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
	return parseBodyDate(bodyLastSeenRe, body)
}

func extractBodyFirstSeen(body string) (time.Time, bool) {
	return parseBodyDate(bodyFirstSeenRe, body)
}

func parseBodyDate(re *regexp.Regexp, body string) (time.Time, bool) {
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(m[1])
	for _, layout := range []string{"Jan 2, 2006", time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func extractBodyAgingStage(body string) string {
	matches := bodyAgingStageRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	if len(last) > 1 {
		return strings.ToLower(strings.TrimSpace(last[1]))
	}
	return ""
}

func extractBodyConfidence(body string) (float64, bool) {
	m := bodyConfidenceRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return 0, false
	}
	var pct float64
	if _, err := fmt.Sscanf(m[1], "%f", &pct); err != nil {
		return 0, false
	}
	return pct / 100.0, true
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
	conf := 0.0
	if issue != nil {
		title = issue.Title
		conf = issue.Confidence
	}
	return fmt.Sprintf(
		"Repository Detective confidence warrants human triage (%.0f%%) in scan `%s`.\n\n**Finding:** %s",
		conf*100,
		scanID,
		title,
	)
}

// ConfidenceNeedsHumanReview reports whether confidence is low enough to flag manual review.
func ConfidenceNeedsHumanReview(confidence float64) bool {
	return confidence > 0 && confidence < 0.65
}

// ConfidenceDelta reports absolute change (exported for tests).
func ConfidenceDelta(a, b float64) float64 {
	return math.Abs(a - b)
}
