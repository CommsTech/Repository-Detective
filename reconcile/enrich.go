package reconcile

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// EnrichmentComment builds a structured update for an existing forge issue.
func EnrichmentComment(f store.Finding, latestScanID, basePath string, repoID int64, graphDetail string) string {
	var b strings.Builder
	b.WriteString("## Repository Detective enrichment\n\n")
	b.WriteString(fmt.Sprintf("**Severity:** %s | **Category:** %s | **Source:** %s\n", f.Severity, f.Category, f.Source))
	if f.RuleID != "" {
		b.WriteString(fmt.Sprintf("**Rule ID:** `%s`\n", f.RuleID))
	}
	b.WriteString(fmt.Sprintf("**Fingerprint:** `%s`\n", f.Fingerprint))
	if latestScanID != "" {
		b.WriteString(fmt.Sprintf("**Latest scan:** `%s`\n", latestScanID))
	}
	b.WriteString(fmt.Sprintf("**Finding status:** %s\n", f.Status))
	if f.FilePath != "" {
		b.WriteString(fmt.Sprintf("**Location:** `%s`", f.FilePath))
		if f.Line > 0 {
			b.WriteString(fmt.Sprintf(":%d", f.Line))
		}
		b.WriteString("\n")
	}
	if basePath != "" && repoID > 0 {
		b.WriteString(fmt.Sprintf("\n- [Repository map](%s/repos/%d/graph)\n", strings.TrimSuffix(basePath, "/"), repoID))
		b.WriteString(fmt.Sprintf("- [Finding detail](%s/findings/%d)\n", strings.TrimSuffix(basePath, "/"), f.ID))
	}
	if graphDetail != "" {
		b.WriteString("\n**Graph context:** " + graphDetail + "\n")
	}
	b.WriteString("\n_Suppress or mark false positive from the finding detail page if this is bot bloat._")
	return b.String()
}
