package issues

import (
	"fmt"
	"strings"
)

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
}

// RenderPRPolicySummary builds a single navigation/summary comment for a PR.
// Canonical finding lifecycle remains in forge issues / RD finding records.
func RenderPRPolicySummary(in PRPolicySummaryInput) string {
	var b strings.Builder
	b.WriteString("<!-- repository-detective-policy-summary -->\n")
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
	return b.String()
}
