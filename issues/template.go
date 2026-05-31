package issues

import (
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
)

const maxEvidenceSnippetLen = 500

// IssueRenderInput carries context for structured issue bodies.
type IssueRenderInput struct {
	Issue       *ai.CodeIssue
	Repository  string
	Context     string
	Commit      string
	PullRequest int
	ScanID      string
	Now         time.Time
}

// RenderIssueBody renders the structured Bugbot issue template.
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
	if issue.RuleID != "" {
		b.WriteString(fmt.Sprintf("- Rule ID: %s\n", issue.RuleID))
	}
	if in.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan ID: %s\n", in.ScanID))
	}

	b.WriteString("\n## Location\n\n")
	if issue.File != "" {
		b.WriteString(fmt.Sprintf("- File: `%s`\n", issue.File))
	}
	if issue.LineNumber > 0 {
		b.WriteString(fmt.Sprintf("- Line: %d\n", issue.LineNumber))
	}
	if in.Commit != "" {
		b.WriteString(fmt.Sprintf("- Commit: %s\n", in.Commit))
	}
	if in.PullRequest > 0 {
		b.WriteString(fmt.Sprintf("- PR: #%d\n", in.PullRequest))
	}

	b.WriteString("\n## Why this matters\n\n")
	b.WriteString(whyThisMatters(category, issue) + "\n\n")

	b.WriteString("## Evidence\n\n")
	if evidence != "" {
		b.WriteString("```\n" + evidence + "\n```\n\n")
	} else {
		b.WriteString("_No code snippet available._\n\n")
	}

	b.WriteString("## Recommended fix\n\n")
	b.WriteString(recommendedFix(issue) + "\n\n")

	b.WriteString("## Regression risk\n\n")
	b.WriteString(fmt.Sprintf("%s — %s\n\n", capitalize(issue.RegressionRisk), regressionReason(issue)))

	b.WriteString("## Suggested tests\n\n")
	b.WriteString(strings.TrimSpace(issue.RequiredTests) + "\n\n")

	b.WriteString("## Tracking\n\n")
	b.WriteString(fmt.Sprintf("- %s %s\n", FingerprintBodyMarker, issue.Fingerprint))
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
	b.WriteString("*Automated finding from Repository Detective*\n")

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

// AIGeneratedRiskWording returns cautious phrasing for future AI-code findings.
func AIGeneratedRiskWording() string {
	return "Possible AI-generated or low-context code risk"
}
