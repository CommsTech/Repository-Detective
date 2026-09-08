package issues

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

const maxEvidenceSnippetLen = 1200

// IssueRenderInput carries context for structured issue bodies.
type IssueRenderInput struct {
	Issue        *ai.CodeIssue
	Repository   string
	Owner        string
	RepoName     string
	GiteaBaseURL string
	Context      string
	Commit       string // immutable commit SHA when known
	Ref          string // branch/tag ref (never displayed as Commit)
	PullRequest  int
	ScanID       string
	Now          time.Time
	FindingID      int64
	Provider       string
	ProductVersion string
	ReportOnly     bool
	IssuePolicy    string
	ConfidenceGate string
	SeverityGate   string
	ScanType       string
	PublicBaseURL  string
	FirstSeen      time.Time
	LastSeen       time.Time
}

// RenderIssueBody renders a developer-facing work item (RD-ISSUE-MODEL-2).
// Sections: Summary → Why this matters → Location → Evidence → Recommended action → Verification → details.
func RenderIssueBody(in IssueRenderInput) string {
	issue := in.Issue
	if issue == nil {
		return ""
	}
	if in.Now.IsZero() {
		in.Now = time.Now().UTC()
	}
	firstSeen := in.FirstSeen
	if firstSeen.IsZero() {
		firstSeen = in.Now
	}
	lastSeen := in.LastSeen
	if lastSeen.IsZero() {
		lastSeen = in.Now
	}

	summary := strings.TrimSpace(issue.Title)
	if summary == "" {
		summary = OperatorIssueTitle(issue)
	}

	evidence := evidenceForIssue(issue)
	if len(evidence) > maxEvidenceSnippetLen {
		evidence = evidence[:maxEvidenceSnippetLen] + "\n… (truncated)"
	}

	commitSHA := resolveCommitSHA(in, issue)
	var b strings.Builder

	if issue.SafeForAutoPR && strings.EqualFold(issue.Fixable, "true") {
		b.WriteString("> **Automated fix available** — Repository Detective can open a safe remediation PR for this finding.\n\n")
	}

	b.WriteString("## Summary\n\n")
	b.WriteString(summaryLead(issue, summary) + "\n\n")
	b.WriteString(fmt.Sprintf("**Severity:** %s  \n", capitalize(issue.Severity)))
	b.WriteString(fmt.Sprintf("**Detection confidence:** %.0f%%  \n", issue.Confidence*100))
	b.WriteString(fmt.Sprintf("**First seen:** %s  \n", firstSeen.Format("Jan 2, 2006")))
	b.WriteString(fmt.Sprintf("**Last seen:** %s\n\n", lastSeen.Format("Jan 2, 2006")))

	b.WriteString("## Why this matters\n\n")
	b.WriteString(whyThisMatters(NormalizeCategory(issue.Category, issue.Source), issue) + "\n\n")

	b.WriteString("## Location\n\n")
	b.WriteString(renderLocationCompact(issue, in, commitSHA) + "\n")

	b.WriteString("\n## Evidence\n\n")
	if evidence != "" && !isDuplicateOfDescription(evidence, issue) {
		lang := evidenceLanguage(issue)
		if lang != "" {
			b.WriteString("```" + lang + "\n" + evidence + "\n```\n\n")
		} else {
			b.WriteString("```\n" + evidence + "\n```\n\n")
		}
	} else if loc := locationRef(issue); loc != "" {
		b.WriteString(fmt.Sprintf("_Open `%s` and inspect surrounding lines. Scanner message alone is not evidence._\n\n", loc))
	} else {
		b.WriteString("_No code snippet available — inspect the reported location in the scanned commit._\n\n")
	}

	b.WriteString("## Recommended action\n\n")
	b.WriteString(recommendedFix(issue) + "\n\n")

	b.WriteString("## Verification\n\n")
	b.WriteString(verificationSteps(issue, in) + "\n\n")

	b.WriteString("<details>\n<summary>Repository Detective details</summary>\n\n")
	b.WriteString(fmt.Sprintf("- Scanner: %s\n", displaySource(issue)))
	b.WriteString(fmt.Sprintf("- Rule: `%s`\n", defaultString(issue.RuleID, "unknown")))
	b.WriteString(fmt.Sprintf("- %s `%s`\n", FingerprintBodyMarker, defaultString(issue.Fingerprint, "unknown")))
	if in.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan: `%s`\n", in.ScanID))
	}
	if in.FindingID > 0 {
		b.WriteString(fmt.Sprintf("- Finding ID: `%d`\n", in.FindingID))
	}
	if issue.ReportingAction != "" {
		b.WriteString(fmt.Sprintf("- Reporting action: %s\n", issue.ReportingAction))
	}
	b.WriteString(fmt.Sprintf("- Fix complexity: %s\n", defaultString(issue.FixComplexity, "unknown")))
	b.WriteString(fmt.Sprintf("- Safe for auto PR: %t\n", issue.SafeForAutoPR))
	if issue.SuggestedPatchStrategy != "" {
		b.WriteString(fmt.Sprintf("- Patch strategy: %s\n", issue.SuggestedPatchStrategy))
	}
	if in.ReportOnly {
		b.WriteString("- Report-only: yes\n")
	}
	if extra := renderSpecializedFindingSection(issue, in); extra != "" {
		b.WriteString("\n")
		b.WriteString(extra)
	}
	b.WriteString("\n</details>\n")

	return b.String()
}

func summaryLead(issue *ai.CodeIssue, title string) string {
	src := displaySource(issue)
	rule := strings.TrimSpace(issue.RuleID)
	file := filepath.Base(issue.File)
	if rule != "" && file != "" && file != "." {
		return fmt.Sprintf("%s `%s` detected in `%s`.", src, rule, file)
	}
	if rule != "" {
		return fmt.Sprintf("%s `%s`: %s", src, rule, title)
	}
	return title
}

func evidenceForIssue(issue *ai.CodeIssue) string {
	snippet := strings.TrimSpace(SanitizeSecretEvidence(issue.CodeSnippet))
	if snippet != "" {
		return snippet
	}
	// Prefer structured evidence fields over repeating the scanner one-liner.
	meta := parseIssueEvidence(issue)
	if code := evidenceField(meta, "code", "snippet", "source_line", "line_text"); code != "" {
		return SanitizeSecretEvidence(code)
	}
	return ""
}

func isDuplicateOfDescription(evidence string, issue *ai.CodeIssue) bool {
	e := strings.TrimSpace(strings.ToLower(evidence))
	if e == "" {
		return true
	}
	for _, cand := range []string{issue.Description, issue.Title, issue.Remediation} {
		c := strings.TrimSpace(strings.ToLower(cand))
		if c != "" && (e == c || strings.Contains(c, e) || strings.Contains(e, c)) {
			return true
		}
	}
	return false
}

func resolveCommitSHA(in IssueRenderInput, issue *ai.CodeIssue) string {
	for _, c := range []string{in.Commit, issue.CommitSHA} {
		c = strings.TrimSpace(c)
		if looksLikeCommitSHA(c) {
			return c
		}
	}
	return ""
}

func looksLikeCommitSHA(ref string) bool {
	if len(ref) < 7 || len(ref) > 64 {
		return false
	}
	for _, r := range ref {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func renderLocationCompact(issue *ai.CodeIssue, in IssueRenderInput, commitSHA string) string {
	var b strings.Builder
	if issue.File != "" {
		loc := locationRef(issue)
		if link := fileSourceLink(in); link != "" {
			b.WriteString(fmt.Sprintf("`%s`\n\n[Open in forge](%s)\n\n", loc, link))
		} else {
			b.WriteString(fmt.Sprintf("`%s`\n\n", loc))
		}
	}
	if commitSHA != "" {
		b.WriteString(fmt.Sprintf("Commit: `%s`\n", commitSHA))
	} else if ref := strings.TrimSpace(in.Ref); ref != "" && !looksLikeCommitSHA(ref) {
		b.WriteString(fmt.Sprintf("Ref: `%s` _(commit SHA unavailable — re-scan with a pinned commit)_\n", ref))
	}
	return strings.TrimRight(b.String(), "\n")
}

func displaySource(issue *ai.CodeIssue) string {
	source := strings.TrimSpace(issue.Source)
	if source == "" {
		return "unknown"
	}
	if issue.FromAI {
		return source + " (AI auditor)"
	}
	return source
}

func whyThisMatters(category string, issue *ai.CodeIssue) string {
	if specific := ruleSpecificImpact(issue); specific != "" {
		return specific
	}
	desc := strings.TrimSpace(issue.Description)
	// Reject useless one-liners that just restate the rule short name.
	if desc != "" && !isGenericScannerBlurb(desc, issue) {
		return desc
	}
	switch NormalizeCategory(category, issue.Source) {
	case CategorySecret:
		return "Exposed secrets can lead to account compromise, data leaks, or unauthorized access if committed or deployed."
	case CategoryDependency:
		return "Known vulnerable dependencies may expose the application to published exploits depending on reachability and deployment."
	case CategoryCodeQuality, CategoryMaintainability:
		return "This pattern can make the code harder to maintain and increases the chance of future bugs."
	case CategoryTechDebt:
		return "Technical debt markers and workarounds can accumulate risk if left unresolved."
	case CategoryReliability:
		return "This pattern may reduce reliability under failure or load; review error handling and timeouts."
	case CategoryTestGap:
		return "Missing or weak tests increase the chance regressions go undetected before release."
	case CategoryPerformance:
		return "This pattern may cause unnecessary work at runtime; review hot paths and resource use."
	case CategoryAIGeneratedRisk:
		return "Possible AI-generated or low-context code risk. Review for hallucinated APIs, weak error handling, or boilerplate that does not match behavior."
	default:
		return "This finding may affect security, reliability, or maintainability depending on context. Confirm exploitability before prioritizing."
	}
}

func isGenericScannerBlurb(desc string, issue *ai.CodeIssue) bool {
	d := strings.ToLower(strings.TrimSpace(desc))
	if len(d) < 40 {
		return true
	}
	rule := strings.ToLower(strings.TrimSpace(issue.RuleID))
	title := strings.ToLower(strings.TrimSpace(issue.Title))
	if rule != "" && d == rule {
		return true
	}
	if title != "" && d == title {
		return true
	}
	generics := []string{
		"sql string formatting",
		"potential hardcoded credentials",
		"use of unsafe",
		"error return value not checked",
	}
	for _, g := range generics {
		if d == g {
			return true
		}
	}
	return false
}

func ruleSpecificImpact(issue *ai.CodeIssue) string {
	rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
	src := strings.ToLower(strings.TrimSpace(issue.Source))
	switch {
	case rule == "G201" || strings.Contains(rule, "SQL_FORMAT") || (src == "gosec" && strings.Contains(strings.ToLower(issue.Title), "sql")):
		return "Formatting values into a SQL string (for example with `fmt.Sprintf`) can permit SQL injection if any formatted fragment can contain untrusted syntax.\n\n" +
			"In many Go codebases the formatted fragment is only a list of `?` placeholders while runtime values stay in `ExecContext`/`QueryContext` args. " +
			"Verify how the formatted fragment is generated before treating this as exploitable. " +
			"If it is only trusted placeholders, classify as a false positive and feed that into Repository Detective calibration."
	case src == "shellcheck" && (strings.Contains(strings.ToLower(issue.Title), "unused") || strings.Contains(rule, "SC2034")):
		return "ShellCheck reports an unused variable. Confirm the variable is truly unused (or export it if it is consumed externally). " +
			"If Repository Detective marks this as auto-fixable, prefer the remediation PR path instead of hand-editing."
	case strings.HasPrefix(rule, "G1") && src == "gosec":
		return fmt.Sprintf("gosec rule `%s` indicates a potential security defect. Confirm the match is reachable with untrusted input before prioritizing as exploitable.", defaultString(issue.RuleID, rule))
	default:
		return ""
	}
}

func recommendedFix(issue *ai.CodeIssue) string {
	if rem := strings.TrimSpace(issue.Remediation); rem != "" && !isGenericScannerBlurb(rem, issue) {
		return rem
	}
	if issue.FromAI {
		return "Review the finding manually. AI-generated guidance should be validated against project conventions and tests."
	}
	rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
	if rule == "G201" {
		return "Determine whether the formatted SQL fragment can contain anything other than internally generated placeholders.\n\n" +
			"* If untrusted content can reach that fragment, replace the construction with a safe parameterized approach.\n" +
			"* If it consists only of trusted placeholder tokens and all values remain bound through query args, " +
			"classify this scanner result as a false positive and feed that outcome into Repository Detective calibration."
	}
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategorySecret:
		return "Remove the secret from source control, rotate the exposed credential, and load secrets from a secure secret manager or environment injection."
	case CategoryDependency:
		return "Upgrade or replace the affected dependency to a patched version and re-scan."
	case CategorySecurity:
		return "Apply the recommended secure pattern for this rule (parameterization, validation, safer API usage) and add a regression test."
	case CategoryCodeQuality, CategoryMaintainability:
		if issue.SafeForAutoPR {
			return "Apply the lint-guided fix (or let Repository Detective open a remediation PR), keep behavior unchanged, and re-scan."
		}
		return "Refactor or adjust the code to follow project lint conventions and keep behavior unchanged."
	default:
		return "Review the evidence, apply a minimal fix, and add or run tests that cover the affected path."
	}
}

func capitalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func locationRef(issue *ai.CodeIssue) string {
	if issue.File == "" {
		return ""
	}
	if issue.LineNumber > 0 {
		return fmt.Sprintf("%s:%d", issue.File, issue.LineNumber)
	}
	return issue.File
}

func fileSourceLink(in IssueRenderInput) string {
	if in.GiteaBaseURL == "" || in.Owner == "" || in.RepoName == "" || in.Issue == nil || in.Issue.File == "" {
		return ""
	}
	commitSHA := resolveCommitSHA(in, in.Issue)
	ref := commitSHA
	useCommitPath := commitSHA != ""
	if ref == "" {
		ref = strings.TrimSpace(in.Ref)
	}
	if ref == "" {
		return "" // never invent "main" as proof of what was scanned
	}
	base := strings.TrimRight(in.GiteaBaseURL, "/")
	var link string
	if strings.Contains(base, "github.com") {
		link = fmt.Sprintf("%s/%s/%s/blob/%s/%s", base, in.Owner, in.RepoName, ref, in.Issue.File)
	} else if useCommitPath {
		link = fmt.Sprintf("%s/%s/%s/src/commit/%s/%s", base, in.Owner, in.RepoName, ref, in.Issue.File)
	} else {
		link = fmt.Sprintf("%s/%s/%s/src/branch/%s/%s", base, in.Owner, in.RepoName, ref, in.Issue.File)
	}
	if in.Issue.LineNumber > 0 {
		link += fmt.Sprintf("#L%d", in.Issue.LineNumber)
	}
	return link
}

func evidenceLanguage(issue *ai.CodeIssue) string {
	if issue == nil || issue.File == "" {
		return ""
	}
	switch strings.ToLower(filepath.Ext(issue.File)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".yaml", ".yml":
		return "yaml"
	case ".sh", ".bash":
		return "bash"
	case ".sql":
		return "sql"
	default:
		return ""
	}
}

func verificationSteps(issue *ai.CodeIssue, in IssueRenderInput) string {
	category := NormalizeCategory(issue.Category, issue.Source)
	switch category {
	case CategorySecret:
		return "1. Confirm the secret is revoked/rotated.\n2. Remove it from source and history if exposed.\n3. Re-scan; Repository Detective closes only when the fingerprint disappears or is explicitly triaged."
	case CategoryDependency:
		return "1. Upgrade the dependency to a fixed version.\n2. Run tests and dependency scan.\n3. Re-scan; the CVE fingerprint should clear."
	default:
		return "Run the relevant project tests and re-scan with the same scanner.\n\n" +
			"Repository Detective will consider this resolved only when the fingerprint no longer appears, " +
			"or when it has been explicitly triaged as false positive or accepted risk."
	}
}
