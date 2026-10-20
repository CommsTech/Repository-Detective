package patcher

import (
	"fmt"
	"strings"
)

// PRRequest describes a pull request to open.
type PRRequest struct {
	Title      string
	Body       string
	HeadBranch string
	BaseBranch string
}

// RenderPRBody builds the PR description for a remediation attempt.
func RenderPRBody(planSummary, diffSummary, validationSummary string) string {
	var b strings.Builder
	b.WriteString("## Repository Detective remediation PR\n\n")
	b.WriteString("_This PR was opened by Repository Detective. It will **not** be merged automatically._\n\n")
	b.WriteString("### Summary\n\n")
	b.WriteString(planSummary)
	b.WriteString("\n\n### Patch\n\n")
	b.WriteString(diffSummary)
	b.WriteString("\n\n### Validation\n\n")
	b.WriteString(validationSummary)
	b.WriteString("\n\n---\n")
	b.WriteString("The linked issue remains open until the PR is merged and a rescan confirms the finding is resolved.\n")
	return b.String()
}

// RenderIssuePRComment builds a Gitea issue comment when a remediation PR is opened.
func RenderIssuePRComment(branchName, prURL, validationSummary string) string {
	var b strings.Builder
	b.WriteString("## Repository Detective remediation PR opened\n\n")
	b.WriteString("_Repository Detective created a branch and pull request only. It will not merge or close this issue._\n\n")
	fmt.Fprintf(&b, "**Branch:** `%s`\n\n", branchName)
	if prURL != "" {
		fmt.Fprintf(&b, "**Pull request:** %s\n\n", prURL)
	}
	b.WriteString("**Validation commands run:**\n\n")
	b.WriteString(validationSummary)
	b.WriteString("\n\nThe issue remains open until the PR is merged and a rescan shows the fingerprint is gone.\n")
	return b.String()
}

// SummarizeValidationResults formats test results for comments.
func SummarizeValidationResults(results []TestResult) string {
	if len(results) == 0 {
		return "_No validation commands were run._"
	}
	var lines []string
	for _, r := range results {
		status := r.Status
		if status == "" {
			status = "unknown"
		}
		lines = append(lines, fmt.Sprintf("- `%s`: **%s**", r.Command, status))
	}
	return strings.Join(lines, "\n")
}
