package remediation

import (
	"fmt"
	"strings"
)

// RenderIssueComment returns a concise Gitea issue comment for a plan.
func RenderIssueComment(plan Plan) string {
	var b strings.Builder
	b.WriteString("## Repository Detective remediation plan\n\n")
	b.WriteString("_Planning only — no code changes are made in this phase._\n\n")
	fmt.Fprintf(&b, "**Fix strategy:** %s\n\n", plan.FixStrategy)
	if len(plan.RequiredTests) > 0 {
		b.WriteString("**Tests required:**\n")
		for _, t := range plan.RequiredTests {
			fmt.Fprintf(&b, "- %s\n", t)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "**Regression risk:** %s\n\n", plan.RegressionRisk)
	fmt.Fprintf(&b, "**Safe for future auto-PR:** %t\n\n", plan.SafeForAutoPR)
	if len(plan.BlockedReasons) > 0 {
		b.WriteString("**Blocked reasons:**\n")
		for _, r := range plan.BlockedReasons {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}
	if plan.Advisory {
		b.WriteString("_Advisory enrichment — verify before acting._\n")
	}
	return b.String()
}

// RenderMarkdown returns a human-readable plan for UI or reports.
func RenderMarkdown(plan Plan) string {
	var b strings.Builder
	b.WriteString("### Remediation plan\n\n")
	b.WriteString("> Planning only — no code changes are made in this phase.\n\n")
	fmt.Fprintf(&b, "**Status:** %s\n\n", plan.Status)
	fmt.Fprintf(&b, "**Summary:** %s\n\n", plan.Summary)
	fmt.Fprintf(&b, "**Fix strategy:** %s\n\n", plan.FixStrategy)
	if len(plan.AffectedFiles) > 0 {
		b.WriteString("**Affected files:**\n")
		for _, f := range plan.AffectedFiles {
			fmt.Fprintf(&b, "- `%s`\n", f)
		}
		b.WriteString("\n")
	}
	if len(plan.RequiredTests) > 0 {
		b.WriteString("**Required tests:**\n")
		for _, t := range plan.RequiredTests {
			fmt.Fprintf(&b, "- %s\n", t)
		}
		b.WriteString("\n")
	}
	if len(plan.ValidationCommands) > 0 {
		b.WriteString("**Validation commands (suggested, not executed):**\n")
		for _, c := range plan.ValidationCommands {
			fmt.Fprintf(&b, "- `%s`\n", c)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "**Regression risk:** %s · **Complexity:** %s\n\n", plan.RegressionRisk, plan.FixComplexity)
	fmt.Fprintf(&b, "**Safe for future auto-PR:** %t · **Requires human review:** %t\n\n", plan.SafeForAutoPR, plan.RequiresHumanReview)
	if len(plan.BlockedReasons) > 0 {
		b.WriteString("**Blocked reasons:**\n")
		for _, r := range plan.BlockedReasons {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// AuditFindingInput converts pre-install audit finding fields to planner context.
func AuditFindingInput(auditID string, category, severity, source, ruleID, title, evidence, filePath, pkg string, confidence float64) FindingContext {
	return FindingContext{
		AuditID:     auditID,
		Category:    category,
		Severity:    severity,
		Source:      source,
		RuleID:      ruleID,
		Title:       title,
		Summary:     evidence,
		Confidence:  confidence,
		FilePath:    filePath,
		PackageName: pkg,
	}
}

// PlanGuidanceForAudit returns markdown guidance for pre-install reports.
func PlanGuidanceForAudit(ctx FindingContext) string {
	plan := ApplyRecipe(ctx)
	plan.AuditID = ctx.AuditID
	plan.SafeForAutoPR = false
	plan.RequiresHumanReview = true
	plan.BlockedReasons = append(plan.BlockedReasons, "pre-install audit — guidance only, no connected repo PR")
	return RenderMarkdown(plan)
}
