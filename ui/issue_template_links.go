package ui

import (
	"fmt"
	"net/url"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

const defaultProductIssueBase = "https://git.commsnet.org/commstech/Repository-Detective"

// IssueTemplateLinks surfaces Gitea issue template URLs and copy guidance for beta feedback.
type IssueTemplateLinks struct {
	FalsePositiveURL    string
	BetaFeedbackURL     string
	MissedDetectionURL  string
	ScannerBugURL       string
	UIIssueURL          string
	DocsGapURL          string
	SecurityReviewURL   string
	TemplateGuidance    string
	ProductIssueBase    string
}

// BuildFindingIssueTemplateLinks builds safe template links for a finding detail page.
func BuildFindingIssueTemplateLinks(detail store.FindingDetail, publicBaseURL string) IssueTemplateLinks {
	base := defaultProductIssueBase
	scanID := detail.LastSeenScanID
	if scanID == "" {
		scanID = detail.FirstSeenScanID
	}
	if len(detail.Instances) > 0 && detail.Instances[0].ScanID != "" {
		scanID = detail.Instances[0].ScanID
	}

	title := fmt.Sprintf("[FP] %s — %s", detail.RuleID, detail.RepoFullName)
	links := IssueTemplateLinks{
		FalsePositiveURL:   giteaNewIssueURL(base, "scanner_false_positive", title, ""),
		MissedDetectionURL: giteaNewIssueURL(base, "missed_detection", "", ""),
		ScannerBugURL:      giteaNewIssueURL(base, "scanner_parser_bug", "", ""),
		UIIssueURL:         giteaNewIssueURL(base, "ui_ux_issue", "", ""),
		DocsGapURL:         giteaNewIssueURL(base, "docs_gap", "", ""),
		SecurityReviewURL:  giteaNewIssueURL(base, "security_triage", "", ""),
		ProductIssueBase:   base,
		TemplateGuidance:   findingTemplateGuidance(detail, scanID, publicBaseURL),
	}
	return links
}

// BuildScanBetaFeedbackLink returns a Gitea URL for scan-level beta feedback.
func BuildScanBetaFeedbackLink(scanID, repoFullName string) string {
	title := fmt.Sprintf("[Beta feedback] scan %s", shortScanID(scanID))
	if repoFullName != "" {
		title = fmt.Sprintf("[Beta feedback] %s — %s", repoFullName, shortScanID(scanID))
	}
	return giteaNewIssueURL(defaultProductIssueBase, "beta_feedback", title, "")
}

func giteaNewIssueURL(base, template, title, body string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	u, err := url.Parse(base + "/issues/new")
	if err != nil {
		return base + "/issues/new"
	}
	q := u.Query()
	if template != "" {
		q.Set("template", template)
	}
	if title != "" {
		q.Set("title", title)
	}
	if body != "" {
		q.Set("body", body)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func findingTemplateGuidance(detail store.FindingDetail, scanID, publicBaseURL string) string {
	var b strings.Builder
	b.WriteString("## Repository Detective false-positive report\n\n")
	b.WriteString("Do **not** paste raw secrets, tokens, `.env`, PHI/PII, or customer data.\n\n")
	b.WriteString("- Repository Detective version/commit: (from `/api/v1/about`)\n")
	b.WriteString(fmt.Sprintf("- Scan ID: `%s`\n", scanID))
	b.WriteString(fmt.Sprintf("- Repository: `%s`\n", detail.RepoFullName))
	b.WriteString(fmt.Sprintf("- Finding ID: `%d`\n", detail.ID))
	b.WriteString(fmt.Sprintf("- Fingerprint: `%s`\n", detail.Fingerprint))
	b.WriteString(fmt.Sprintf("- Rule/source: `%s` / `%s`\n", detail.RuleID, detail.Source))
	b.WriteString(fmt.Sprintf("- Severity/confidence: %s / %.2f\n", detail.Severity, detail.Confidence))
	if detail.FilePath != "" {
		b.WriteString(fmt.Sprintf("- File/path/line: `%s`", detail.FilePath))
		if detail.Line > 0 {
			b.WriteString(fmt.Sprintf(":%d", detail.Line))
		}
		b.WriteString("\n")
	}
	b.WriteString("- Expected behavior: \n")
	b.WriteString("- Actual behavior: \n")
	b.WriteString("- Why false positive: \n")
	if publicBaseURL != "" && scanID != "" {
		base := strings.TrimRight(publicBaseURL, "/")
		b.WriteString(fmt.Sprintf("- Finding URL: %s/findings/%d\n", base, detail.ID))
		b.WriteString(fmt.Sprintf("- Scan URL: %s/scans/%s\n", base, scanID))
	}
	return b.String()
}

func shortScanID(scanID string) string {
	scanID = strings.TrimSpace(scanID)
	if len(scanID) > 12 {
		return scanID[:12]
	}
	return scanID
}
