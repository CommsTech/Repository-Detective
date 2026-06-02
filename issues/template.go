package issues

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
)

const maxEvidenceSnippetLen = 500

// IssueRenderInput carries context for structured issue bodies.
type IssueRenderInput struct {
	Issue       *ai.CodeIssue
	Repository  string
	Owner       string
	RepoName    string
	GiteaBaseURL string
	Context     string
	Commit      string
	Ref         string
	PullRequest int
	ScanID      string
	Now         time.Time
}

// RenderIssueBody renders the structured Repository Detective issue template.
func RenderIssueBody(in IssueRenderInput) string {
	issue := in.Issue
	if issue == nil {
		return ""
	}
	if in.Now.IsZero() {
		in.Now = time.Now()
	}

	category := NormalizeCategory(issue.Category, issue.Source)
	summary := strings.TrimSpace(issue.Title)
	if summary == "" {
		summary = "Potential issue detected by Repository Detective"
	}

	evidence := SanitizeSecretEvidence(issue.CodeSnippet)
	if len(evidence) > maxEvidenceSnippetLen {
		evidence = evidence[:maxEvidenceSnippetLen] + "..."
	}

	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(summary + "\n\n")

	b.WriteString("## Finding Type\n\n")
	b.WriteString(fmt.Sprintf("- Category: %s\n", category))
	b.WriteString(fmt.Sprintf("- Severity: %s\n", strings.ToLower(issue.Severity)))
	b.WriteString(fmt.Sprintf("- Confidence: %.2f\n", issue.Confidence))
	b.WriteString(fmt.Sprintf("- Source: %s\n", displaySource(issue)))
	if issue.SourceType != "" {
		b.WriteString(fmt.Sprintf("- Source type: %s\n", issue.SourceType))
	}
	if issue.ReportingAction != "" {
		b.WriteString(fmt.Sprintf("- Reporting action: %s\n", issue.ReportingAction))
	}
	if issue.FalsePositiveRisk != "" {
		b.WriteString(fmt.Sprintf("- False-positive risk: %s\n", issue.FalsePositiveRisk))
	}
	if issue.RepoProfileSummary != "" {
		b.WriteString(fmt.Sprintf("- Repo profile: %s\n", issue.RepoProfileSummary))
	}
	if issue.RuleID != "" {
		b.WriteString(fmt.Sprintf("- Rule ID: %s\n", issue.RuleID))
	}
	if in.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan ID: %s\n", in.ScanID))
	}

	b.WriteString("\n## Location\n\n")
	if issue.File != "" {
		if link := fileSourceLink(in); link != "" {
			b.WriteString(fmt.Sprintf("- File: [`%s`](%s)\n", locationRef(issue), link))
		} else {
			b.WriteString(fmt.Sprintf("- File: `%s`\n", locationRef(issue)))
		}
	}
	if issue.LineNumber > 0 {
		b.WriteString(fmt.Sprintf("- Line: %d\n", issue.LineNumber))
	}
	if in.Repository != "" {
		b.WriteString(fmt.Sprintf("- Repository: `%s`\n", in.Repository))
	}
	if in.Context != "" {
		b.WriteString(fmt.Sprintf("- Trigger context: %s\n", in.Context))
	}
	if in.Commit != "" {
		b.WriteString(fmt.Sprintf("- Commit / ref: `%s`\n", in.Commit))
	}
	if in.PullRequest > 0 {
		b.WriteString(fmt.Sprintf("- Pull request: #%d\n", in.PullRequest))
	}

	b.WriteString("\n## Why this matters\n\n")
	b.WriteString(whyThisMatters(category, issue) + "\n\n")

	b.WriteString("## Evidence\n\n")
	if evidence != "" {
		lang := evidenceLanguage(issue)
		if lang != "" {
			b.WriteString("```" + lang + "\n" + evidence + "\n```\n\n")
		} else {
			b.WriteString("```\n" + evidence + "\n```\n\n")
		}
	} else {
		b.WriteString("_No code snippet available. Open the file at the location above and inspect surrounding logic._\n\n")
	}

	b.WriteString("## Recommended fix\n\n")
	b.WriteString(recommendedFix(issue) + "\n\n")

	b.WriteString("## Regression risk\n\n")
	b.WriteString(fmt.Sprintf("%s — %s\n\n", capitalize(issue.RegressionRisk), regressionReason(issue)))

	b.WriteString("## Suggested tests\n\n")
	b.WriteString(strings.TrimSpace(issue.RequiredTests) + "\n\n")

	b.WriteString("## Reproduction\n\n")
	b.WriteString(reproductionSteps(issue, in) + "\n\n")

	b.WriteString("## Report flow\n\n")
	b.WriteString(reportFlowTable(issue, in) + "\n\n")

	b.WriteString("## Acceptance criteria\n\n")
	b.WriteString(acceptanceCriteria(issue) + "\n\n")

	b.WriteString("## Links\n\n")
	b.WriteString(reportLinks(in, issue) + "\n\n")

	b.WriteString("## Tracking\n\n")
	b.WriteString(fmt.Sprintf("- %s %s\n", FingerprintBodyMarker, issue.Fingerprint))
	if issue.SuppressionReason != "" {
		b.WriteString(fmt.Sprintf("- Suppression note: %s\n", issue.SuppressionReason))
	}
	b.WriteString(fmt.Sprintf("- First seen: %s\n", in.Now.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Last seen: %s\n", in.Now.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Status: %s\n", lifecycleStatusLabel(issue)))

	if issue.Fixable != "" || issue.FixComplexity != "" || issue.SuggestedPatchStrategy != "" {
		b.WriteString("\n## Remediation readiness\n\n")
		b.WriteString(fmt.Sprintf("- Fixable: %s\n", defaultString(issue.Fixable, "unknown")))
		b.WriteString(fmt.Sprintf("- Fix complexity: %s\n", defaultString(issue.FixComplexity, "unknown")))
		b.WriteString(fmt.Sprintf("- Safe for auto PR: %t\n", issue.SafeForAutoPR))
		if issue.SuggestedPatchStrategy != "" {
			b.WriteString(fmt.Sprintf("- Suggested patch strategy: %s\n", issue.SuggestedPatchStrategy))
		}
	}

	if issue.ProofOfConcept != "" && !issue.FromAI {
		poc := SanitizeSecretEvidence(issue.ProofOfConcept)
		b.WriteString("\n## Proof of concept\n\n")
		b.WriteString("```\n" + poc + "\n```\n")
	}

	b.WriteString("\n---\n")
	b.WriteString("> **Repository Detective** · Inspect · Analyze · Improve\n\n")
	b.WriteString("*Automated finding — review the report flow above before closing.*\n")

	return b.String()
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
	if desc := strings.TrimSpace(issue.Description); desc != "" {
		return desc
	}
	switch NormalizeCategory(category, issue.Source) {
	case CategorySecret:
		return "Exposed secrets can lead to account compromise, data leaks, or unauthorized access if committed or deployed."
	case CategoryDependency:
		return "Known vulnerable dependencies may expose the application to published exploits."
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
		return "This finding may affect security, reliability, or maintainability depending on context."
	}
}

func recommendedFix(issue *ai.CodeIssue) string {
	if issue.FromAI {
		return "Review the finding manually. AI-generated guidance should be validated against project conventions and tests."
	}
	switch NormalizeCategory(issue.Category, issue.Source) {
	case CategorySecret:
		return "Remove the secret from source control, rotate the exposed credential, and load secrets from a secure secret manager or environment injection."
	case CategoryDependency:
		return "Upgrade or replace the affected dependency to a patched version and rerun dependency scans."
	case CategorySecurity:
		return "Apply the recommended secure pattern for this rule (parameterization, validation, safer API usage) and add a regression test."
	case CategoryCodeQuality, CategoryMaintainability:
		return "Refactor or adjust the code to follow project lint/security conventions and keep behavior unchanged."
	default:
		return "Review the evidence, apply a minimal fix, and add or run tests that cover the affected path."
	}
}

func regressionReason(issue *ai.CodeIssue) string {
	switch strings.ToLower(strings.TrimSpace(issue.RegressionRisk)) {
	case "low":
		return "localized change with clear scanner or lint guidance"
	case "high":
		return "may affect security-sensitive behavior or shared code paths"
	default:
		return "verify behavior with existing or new tests before merging"
	}
}

func lifecycleStatusLabel(issue *ai.CodeIssue) string {
	if issue.LifecycleState != "" {
		return issue.LifecycleState
	}
	if ConfidenceNeedsHumanReview(issue.Confidence) {
		return LifecycleNeedsHumanReview
	}
	return LifecycleOpen
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
	ref := strings.TrimSpace(in.Ref)
	if ref == "" {
		ref = strings.TrimSpace(in.Commit)
	}
	if ref == "" {
		ref = "main"
	}
	base := strings.TrimRight(in.GiteaBaseURL, "/")
	var link string
	if strings.Contains(base, "github.com") {
		link = fmt.Sprintf("%s/%s/%s/blob/%s/%s", base, in.Owner, in.RepoName, ref, in.Issue.File)
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
	case ".sh":
		return "bash"
	case ".sql":
		return "sql"
	default:
		return ""
	}
}

func reproductionSteps(issue *ai.CodeIssue, in IssueRenderInput) string {
	var steps []string
	if issue.File != "" {
		loc := locationRef(issue)
		if link := fileSourceLink(in); link != "" {
			steps = append(steps, fmt.Sprintf("1. Open [`%s`](%s).", loc, link))
		} else {
			steps = append(steps, fmt.Sprintf("1. Open `%s` in the repository.", loc))
		}
	} else {
		steps = append(steps, "1. Locate the affected code path referenced in the summary.")
	}
	if issue.RuleID != "" {
		steps = append(steps, fmt.Sprintf("2. Search for rule `%s` / pattern from scanner `%s`.", issue.RuleID, displaySource(issue)))
	} else {
		steps = append(steps, fmt.Sprintf("2. Review the logic flagged by `%s`.", displaySource(issue)))
	}
	if strings.TrimSpace(issue.CodeSnippet) != "" {
		steps = append(steps, "3. Confirm the evidence snippet matches current code (may drift after refactors).")
	} else {
		steps = append(steps, "3. Re-run the relevant scanner or manual review to confirm the finding still applies.")
	}
	steps = append(steps, "4. Document whether the issue is a true positive, false positive, or accepted risk.")
	return strings.Join(steps, "\n")
}

func reportFlowTable(issue *ai.CodeIssue, in IssueRenderInput) string {
	var b strings.Builder
	b.WriteString("Use this checklist to triage, fix, and verify the finding:\n\n")
	b.WriteString("| Step | Action | Done |\n")
	b.WriteString("| --- | --- | --- |\n")
	b.WriteString("| 1. Triage | Validate severity, category, and whether this is a true positive | [ ] |\n")
	b.WriteString("| 2. Assign | Set an owner and target milestone | [ ] |\n")
	b.WriteString("| 3. Reproduce | Follow reproduction steps above | [ ] |\n")
	b.WriteString("| 4. Fix | Apply the recommended fix with minimal scope | [ ] |\n")
	b.WriteString("| 5. Test | Run suggested tests and CI | [ ] |\n")
	b.WriteString("| 6. Verify | Re-scan or manually confirm; close when fingerprint no longer reproduces | [ ] |\n")
	if ConfidenceNeedsHumanReview(issue.Confidence) {
		b.WriteString("\n> **Needs human review** — confidence is below the auto-remediation threshold.\n")
	}
	if in.ScanID != "" {
		b.WriteString(fmt.Sprintf("\nTrack verification against scan `%s`.\n", in.ScanID))
	}
	return b.String()
}

func acceptanceCriteria(issue *ai.CodeIssue) string {
	var items []string
	items = append(items, "- The vulnerable or problematic pattern no longer exists at the reported location.")
	items = append(items, "- Suggested tests pass and no related regressions are introduced.")
	if issue.Fingerprint != "" {
		items = append(items, fmt.Sprintf("- Re-scan does not reopen fingerprint `%s`.", issue.Fingerprint))
	}
	if issue.FromAI {
		items = append(items, "- A human reviewer confirms the AI finding matches project context.")
	}
	return strings.Join(items, "\n")
}

func reportLinks(in IssueRenderInput, issue *ai.CodeIssue) string {
	var links []string
	if in.Repository != "" {
		links = append(links, fmt.Sprintf("- Repository: `%s`", in.Repository))
	}
	if link := fileSourceLink(in); link != "" {
		links = append(links, fmt.Sprintf("- Source file: %s", link))
	}
	if in.ScanID != "" {
		links = append(links, fmt.Sprintf("- Scan ID: `%s`", in.ScanID))
	}
	if issue.RuleID != "" {
		links = append(links, fmt.Sprintf("- Rule ID: `%s`", issue.RuleID))
	}
	return strings.Join(links, "\n")
}

// AIGeneratedRiskWording returns cautious phrasing for future AI-code findings.
func AIGeneratedRiskWording() string {
	return "Possible AI-generated or low-context code risk"
}
