package issues

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// PRPolicySummaryMarker uniquely identifies Repository Detective PR summary comments.
const PRPolicySummaryMarker = "<!-- repository-detective-policy-summary -->"

// PRPolicySummaryInput feeds a compact PR comment (not per-finding inline comments).
type PRPolicySummaryInput struct {
	Outcome         string
	EnforcementMode string
	Description     string
	IssueCount      int
	ScannerCoverage string
	ScanID          string
	CommitSHA       string
	UIBase          string
	Timestamp       string
}

// RenderPRPolicySummary builds a single navigation/summary comment for a PR.
// Canonical finding lifecycle remains in forge issues / RD finding records.
func RenderPRPolicySummary(in PRPolicySummaryInput) string {
	var b strings.Builder
	b.WriteString(PRPolicySummaryMarker + "\n")
	b.WriteString("### Repository Detective\n\n")
	if in.Outcome != "" {
		fmt.Fprintf(&b, "- **Policy:** `%s`\n", in.Outcome)
	}
	if in.EnforcementMode != "" {
		fmt.Fprintf(&b, "- **Mode:** %s\n", in.EnforcementMode)
	}
	if in.ScannerCoverage != "" {
		fmt.Fprintf(&b, "- **Analysis:** %s\n", in.ScannerCoverage)
	}
	fmt.Fprintf(&b, "- **Findings linked to this scan:** %d (canonical issues — not duplicated as inline review comments)\n", in.IssueCount)
	if in.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", in.Description)
	}
	b.WriteString("\nPolicy outcomes describe compliance with the **owner-configured repository policy**, not that the code is safe or secure.\n")
	if in.UIBase != "" && in.ScanID != "" {
		fmt.Fprintf(&b, "\n[Open scan in Repository Detective](%s/ui/scans/%s)\n", in.UIBase, in.ScanID)
	} else if in.ScanID != "" {
		fmt.Fprintf(&b, "\nScan ID: `%s`\n", in.ScanID)
	}
	if in.CommitSHA != "" {
		fmt.Fprintf(&b, "\nCommit: `%s`\n", in.CommitSHA)
	}
	if in.Timestamp != "" {
		fmt.Fprintf(&b, "\nUpdated: `%s`\n", in.Timestamp)
	}
	return b.String()
}

// IsPRPolicySummaryBody reports whether a comment body is an RD-owned policy summary.
// Requires the exact canonical marker — similar-looking HTML comments do not match.
func IsPRPolicySummaryBody(body string) bool {
	return strings.Contains(body, PRPolicySummaryMarker)
}

// CommentRef is a forge comment identity for upsert/dedupe.
type CommentRef struct {
	ID   int64
	Body string
}

// PRCommentAPI is the forge surface used for idempotent PR summaries.
type PRCommentAPI interface {
	ListIssueComments(ctx context.Context, owner, repo string, issueNumber int) ([]CommentRef, error)
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) (int64, error)
	EditIssueComment(ctx context.Context, owner, repo string, commentID int64, body string) error
	DeleteIssueComment(ctx context.Context, owner, repo string, commentID int64) error
}

// UpsertResult describes what UpsertPRPolicySummary did.
type UpsertResult struct {
	Action            string // created | updated | skipped_duplicate_in_flight | failed
	CanonicalID       int64
	DuplicatesRemoved int
	Err               error
}

// inFlightPRSummary guards concurrent webhook retries for the same PR.
var inFlightPRSummary sync.Map // key owner/repo#n → struct{}

func prSummaryLockKey(owner, repo string, prNumber int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, prNumber)
}

// UpsertPRPolicySummary creates or updates exactly one RD policy summary on a PR.
// It never modifies comments that lack the exact marker.
func UpsertPRPolicySummary(ctx context.Context, api PRCommentAPI, owner, repo string, prNumber int, body string) UpsertResult {
	if api == nil || prNumber <= 0 || !IsPRPolicySummaryBody(body) {
		return UpsertResult{Action: "failed", Err: fmt.Errorf("invalid upsert request")}
	}
	key := prSummaryLockKey(owner, repo, prNumber)
	if _, loaded := inFlightPRSummary.LoadOrStore(key, struct{}{}); loaded {
		return UpsertResult{Action: "skipped_duplicate_in_flight"}
	}
	defer inFlightPRSummary.Delete(key)

	comments, err := api.ListIssueComments(ctx, owner, repo, prNumber)
	if err != nil {
		// Fail closed on list: do not create a second comment when we cannot see existing ones.
		return UpsertResult{Action: "failed", Err: fmt.Errorf("list comments: %w", err)}
	}

	var owned []CommentRef
	for _, c := range comments {
		if IsPRPolicySummaryBody(c.Body) {
			owned = append(owned, c)
		}
	}

	if len(owned) == 0 {
		id, cerr := api.CreateIssueComment(ctx, owner, repo, prNumber, body)
		if cerr != nil {
			return UpsertResult{Action: "failed", Err: cerr}
		}
		return UpsertResult{Action: "created", CanonicalID: id}
	}

	canonical := owned[0]
	if err := api.EditIssueComment(ctx, owner, repo, canonical.ID, body); err != nil {
		return UpsertResult{Action: "failed", CanonicalID: canonical.ID, Err: fmt.Errorf("edit comment: %w", err)}
	}

	removed := 0
	for _, dup := range owned[1:] {
		if derr := api.DeleteIssueComment(ctx, owner, repo, dup.ID); derr != nil {
			// Best-effort cleanup; canonical update already succeeded.
			continue
		}
		removed++
	}
	return UpsertResult{Action: "updated", CanonicalID: canonical.ID, DuplicatesRemoved: removed}
}
