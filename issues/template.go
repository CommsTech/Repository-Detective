package issues

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
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
	// Extended metadata for actionable forge issues
	FindingID      int64
	Provider       string
	ProductVersion string
	ReportOnly     bool
	IssuePolicy    string
	ConfidenceGate string
	SeverityGate   string
	ScanType       string
	PublicBaseURL  string
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

	b.WriteString("## Finding\n\n")
	b.WriteString(renderFindingSection(issue, in) + "\n")

	b.WriteString("\n## Location\n\n")
	b.WriteString(renderLocationSection(issue, in) + "\n")

	if extra := renderSpecializedFindingSection(issue, in); extra != "" {
		b.WriteString("\n")
		b.WriteString(extra)
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

	b.WriteString("## Verification\n\n")
	b.WriteString(verificationSteps(issue, in) + "\n\n")

	b.WriteString("## Issue filing policy\n\n")
	b.WriteString(renderIssueFilingPolicy(in) + "\n")

	b.WriteString("\n## False-positive guidance\n\n")
	b.WriteString(falsePositiveGuidance(in) + "\n")

	b.WriteString("\n## Repository Detective metadata\n\n")
	b.WriteString(renderProductMetadata(in, issue) + "\n")

	b.WriteString("\n## Regression risk\n\n")
	b.WriteString(fmt.Sprintf("%s — %s\n\n", capitalize(issue.RegressionRisk), regressionReason(issue)))

	b.WriteString("## Suggested tests\n\n")
	b.WriteString(strings.TrimSpace(issue.RequiredTests) + "\n\n")

	b.WriteString("## Reproduction\n\n")
	b.WriteString(reproductionSteps(issue, in) + "\n\n")

	b.WriteString("## Report flow\n\n")
	b.WriteString(reportFlowTable(issue, in) + "\n\n")

	b.WriteString("## Acceptance criteria\n\n")
	b.WriteString(acceptanceCriteria(issue) + "\n\n")

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

func renderFindingSection(issue *ai.CodeIssue, in IssueRenderInput) string {
	category := NormalizeCategory(issue.Category, issue.Source)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- Fingerprint: `%s`\n", defaultString(issue.Fingerprint, "unknown")))
	if in.FindingID > 0 {
		b.WriteString(fmt.Sprintf("- Finding ID: `%d`\n", in.FindingID))
	}
	b.WriteString(fmt.Sprintf("- Rule: `%s`\n", defaultString(issue.RuleID, "unknown")))
	b.WriteString(fmt.Sprintf("- Scanner/source: %s\n", displaySource(issue)))
	b.WriteString(fmt.Sprintf("- Severity: %s\n", strings.ToLower(defaultString(issue.Severity, "unknown"))))
	b.WriteString(fmt.Sprintf("- Confidence: %.2f\n", issue.Confidence))
	b.WriteString(fmt.Sprintf("- Category: %s\n", category))
	b.WriteString(fmt.Sprintf("- Status: %s\n", lifecycleStatusLabel(issue)))
	if !in.Now.IsZero() {
		b.WriteString(fmt.Sprintf("- First seen: %s\n", in.Now.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("- Last seen: %s\n", in.Now.Format(time.RFC3339)))
	}
	if issue.SourceType != "" {
		b.WriteString(fmt.Sprintf("- Source type: %s\n", issue.SourceType))
	}
	if issue.ReportingAction != "" {
		b.WriteString(fmt.Sprintf("- Reporting action: %s\n", issue.ReportingAction))
	}
	if issue.FalsePositiveRisk != "" {
		b.WriteString(fmt.Sprintf("- False-positive risk: %s\n", issue.FalsePositiveRisk))
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderLocationSection(issue *ai.CodeIssue, in IssueRenderInput) string {
	provider := defaultString(in.Provider, inferProvider(in))
	ref := strings.TrimSpace(in.Ref)
	if ref == "" {
		ref = strings.TrimSpace(in.Commit)
	}
	var b strings.Builder
	if in.Repository != "" {
		b.WriteString(fmt.Sprintf("- Repository: `%s`\n", in.Repository))
	}
	b.WriteString(fmt.Sprintf("- Provider: %s\n", provider))
	if ref != "" {
		b.WriteString(fmt.Sprintf("- Branch/ref: `%s`\n", ref))
	}
	if in.Commit != "" {
		b.WriteString(fmt.Sprintf("- Commit: `%s`\n", in.Commit))
	} else if issue.CommitSHA != "" {
		b.WriteString(fmt.Sprintf("- Commit: `%s`\n", issue.CommitSHA))
	}
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
	if issue.PackageName != "" {
		b.WriteString(fmt.Sprintf("- Function/package/component: `%s`\n", issue.PackageName))
	}
	if isHistoricalFinding(issue) {
		b.WriteString("- Current tree or historical: historical (may not exist at HEAD)\n")
	} else if issue.File != "" {
		b.WriteString("- Current tree or historical: current tree\n")
	}
	if in.Context != "" {
		b.WriteString(fmt.Sprintf("- Trigger context: %s\n", in.Context))
	}
	if in.PullRequest > 0 {
		b.WriteString(fmt.Sprintf("- Pull request: #%d\n", in.PullRequest))
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderSpecializedFindingSection(issue *ai.CodeIssue, in IssueRenderInput) string {
	meta := parseIssueEvidence(issue)
	var sections []string

	if isHistoricalFinding(issue) || NormalizeCategory(issue.Category, issue.Source) == CategorySecret && issue.CommitSHA != "" {
		var b strings.Builder
		b.WriteString("## Secret / history context\n\n")
		commit := firstNonEmptyStr(issue.CommitSHA, evidenceField(meta, "commit", "commit_sha"))
		b.WriteString(fmt.Sprintf("- Commit hash: `%s`\n", defaultString(commit, "unknown")))
		present := evidenceField(meta, "current_tree_present", "present_in_tree")
		if present == "" {
			if isHistoricalFinding(issue) {
				present = "unknown — verify at HEAD"
			} else {
				present = "yes (current tree)"
			}
		}
		b.WriteString(fmt.Sprintf("- Current-tree present: %s\n", present))
		rotation := "yes — rotate if credential was ever active"
		if NormalizeCategory(issue.Category, issue.Source) != CategorySecret {
			rotation = "review if secret material was exposed"
		}
		b.WriteString(fmt.Sprintf("- Rotation required: %s\n", rotation))
		sections = append(sections, b.String())
	}

	if isContainerFinding(issue) {
		var b strings.Builder
		b.WriteString("## Container context\n\n")
		b.WriteString(fmt.Sprintf("- Image: `%s`\n", defaultString(evidenceField(meta, "image", "Image"), issue.File)))
		b.WriteString(fmt.Sprintf("- Digest: `%s`\n", defaultString(evidenceField(meta, "image_digest", "digest", "ImageID"), "unknown")))
		b.WriteString(fmt.Sprintf("- Package: `%s`\n", defaultString(firstNonEmptyStr(issue.PackageName, evidenceField(meta, "package", "PackageName", "pkg_name")), "unknown")))
		b.WriteString(fmt.Sprintf("- Installed version: `%s`\n", defaultString(evidenceField(meta, "version", "installed_version", "PackageVersion"), "unknown")))
		b.WriteString(fmt.Sprintf("- Fixed version: `%s`\n", defaultString(evidenceField(meta, "fixed_version", "FixedVersion"), "unknown")))
		b.WriteString(fmt.Sprintf("- CVE: `%s`\n", defaultString(evidenceField(meta, "cve", "cve_id", "VulnerabilityID", "vulnerability_id"), "unknown")))
		sections = append(sections, b.String())
	}

	if isSBOMFinding(issue) {
		var b strings.Builder
		b.WriteString("## SBOM context\n\n")
		component := firstNonEmptyStr(issue.PackageName, evidenceField(meta, "sbom_component", "component", "purl", "bom_ref"))
		b.WriteString(fmt.Sprintf("- Package/component: `%s`\n", defaultString(component, "unknown")))
		b.WriteString(fmt.Sprintf("- Ecosystem: `%s`\n", defaultString(evidenceField(meta, "ecosystem", "type", "package_type"), "unknown")))
		if lic := evidenceField(meta, "license", "License"); lic != "" {
			b.WriteString(fmt.Sprintf("- License: `%s`\n", lic))
		}
		sections = append(sections, b.String())
	}

	if isGraphFinding(issue) {
		var b strings.Builder
		b.WriteString("## Graph context\n\n")
		b.WriteString(fmt.Sprintf("- Node type: `%s`\n", defaultString(evidenceField(meta, "node_type", "NodeType"), "unknown")))
		reach := evidenceField(meta, "entrypoint_reachable", "EntrypointReachable", "path_classification")
		if reach != "" {
			b.WriteString(fmt.Sprintf("- Reachability/entrypoint context: %s\n", reach))
		} else {
			b.WriteString("- Reachability/entrypoint context: review Repository Map for inbound/outbound edges\n")
		}
		sections = append(sections, b.String())
	}

	if isPreinstallFinding(issue, in) {
		sections = append(sections, "## Pre-install audit context\n\n"+
			"- Report-only: yes — pre-install audits do not file upstream issues automatically\n"+
			"- Issue filed upstream: no (operator review required)\n")
	}

	return strings.Join(sections, "\n")
}

func renderIssueFilingPolicy(in IssueRenderInput) string {
	reportOnly := "no"
	if in.ReportOnly {
		reportOnly = "yes"
	}
	backlog := "disabled"
	if in.IssuePolicy != "" && strings.Contains(strings.ToLower(in.IssuePolicy), "backlog") {
		backlog = "active"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- Scan ID: `%s`\n", defaultString(in.ScanID, "unknown")))
	b.WriteString(fmt.Sprintf("- Scan type: %s\n", defaultString(in.ScanType, "repository scan")))
	b.WriteString(fmt.Sprintf("- Report-only: %s\n", reportOnly))
	b.WriteString(fmt.Sprintf("- Issue policy: %s\n", defaultString(in.IssuePolicy, "default")))
	b.WriteString(fmt.Sprintf("- Confidence gate: %s\n", defaultString(in.ConfidenceGate, "profile default")))
	b.WriteString(fmt.Sprintf("- Severity gate: %s\n", defaultString(in.SeverityGate, "profile default")))
	b.WriteString(fmt.Sprintf("- Backlog-control: %s\n", backlog))
	return strings.TrimRight(b.String(), "\n")
}

func falsePositiveGuidance(in IssueRenderInput) string {
	repo := defaultString(in.Repository, "owner/repo")
	return fmt.Sprintf(
		"If this is noise or acceptable risk:\n\n"+
			"1. Mark false positive in Repository Detective with a short reason (repo-scoped calibration).\n"+
			"2. Or file a **Scanner false positive** issue on the product repo using template `scanner_false_positive`.\n"+
			"3. Include fingerprint `%s`, scan ID `%s`, rule, and redacted evidence — **never paste raw secrets**.\n\n"+
			"Global suppressions require operator review; prefer repo-scoped calibration for beta feedback.",
		defaultString(in.Issue.Fingerprint, "unknown"),
		defaultString(in.ScanID, "unknown"),
	) + fmt.Sprintf("\n\nRepository under test: `%s`.", repo)
}

func renderProductMetadata(in IssueRenderInput, issue *ai.CodeIssue) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- Product version: %s\n", defaultString(in.ProductVersion, "see /api/v1/about")))
	if in.PublicBaseURL != "" && in.ScanID != "" {
		base := strings.TrimRight(in.PublicBaseURL, "/")
		b.WriteString(fmt.Sprintf("- Scan URL: %s/scans/%s\n", base, in.ScanID))
		if in.FindingID > 0 {
			b.WriteString(fmt.Sprintf("- Finding URL: %s/findings/%d\n", base, in.FindingID))
		}
		b.WriteString(fmt.Sprintf("- Report URL: %s/scans/%s\n", base, in.ScanID))
	} else if in.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan ID reference: `%s`\n", in.ScanID))
	}
	if issue != nil && issue.Fingerprint != "" {
		b.WriteString(fmt.Sprintf("- Fingerprint reference: `%s`\n", issue.Fingerprint))
	}
	return strings.TrimRight(b.String(), "\n")
}

func verificationSteps(issue *ai.CodeIssue, in IssueRenderInput) string {
	category := NormalizeCategory(issue.Category, issue.Source)
	switch category {
	case CategorySecret:
		return "1. Confirm secret is revoked/rotated.\n2. Remove from source and history if exposed.\n3. Re-scan; fingerprint should not reappear as active-present."
	case CategoryDependency:
		return "1. Upgrade dependency to fixed version.\n2. Run tests and dependency scan.\n3. Re-scan lockfile or image; CVE fingerprint should clear."
	default:
		steps := reproductionSteps(issue, in)
		return steps + "\n5. Re-scan or manually confirm; close when fingerprint no longer reproduces."
	}
}

func parseIssueEvidence(issue *ai.CodeIssue) map[string]string {
	out := map[string]string{}
	if issue == nil {
		return out
	}
	raw := strings.TrimSpace(issue.Evidence)
	if raw == "" {
		return out
	}
	var generic map[string]any
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return out
	}
	for k, v := range generic {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
			out[k] = s
		}
	}
	return out
}

func evidenceField(meta map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := meta[k]; strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func inferProvider(in IssueRenderInput) string {
	base := strings.ToLower(in.GiteaBaseURL)
	switch {
	case strings.Contains(base, "github.com"):
		return "github"
	case strings.Contains(base, "gitlab"):
		return "gitlab"
	case in.GiteaBaseURL != "":
		return "gitea"
	default:
		return "gitea"
	}
}

func isHistoricalFinding(issue *ai.CodeIssue) bool {
	if issue == nil {
		return false
	}
	if issue.CommitSHA != "" && issue.File == "" {
		return true
	}
	meta := parseIssueEvidence(issue)
	if v := evidenceField(meta, "historical", "is_historical"); strings.EqualFold(v, "true") || v == "1" {
		return true
	}
	return strings.Contains(strings.ToLower(issue.SourceType), "history")
}

func isContainerFinding(issue *ai.CodeIssue) bool {
	if issue == nil {
		return false
	}
	src := strings.ToLower(issue.Source)
	cat := strings.ToLower(issue.Category)
	if strings.Contains(src, "container") || strings.Contains(src, "trivy") || strings.Contains(src, "grype") {
		return true
	}
	if cat == "container" || cat == "vulnerability" && issue.PackageName != "" {
		meta := parseIssueEvidence(issue)
		return evidenceField(meta, "image", "image_digest", "digest") != ""
	}
	return false
}

func isSBOMFinding(issue *ai.CodeIssue) bool {
	if issue == nil {
		return false
	}
	src := strings.ToLower(issue.Source)
	if strings.Contains(src, "sbom") {
		return true
	}
	meta := parseIssueEvidence(issue)
	return evidenceField(meta, "sbom_component", "purl", "bom_ref") != ""
}

func isGraphFinding(issue *ai.CodeIssue) bool {
	if issue == nil {
		return false
	}
	return strings.EqualFold(issue.Source, "graph")
}

func isPreinstallFinding(issue *ai.CodeIssue, in IssueRenderInput) bool {
	if issue == nil {
		return false
	}
	src := strings.ToLower(issue.Source)
	st := strings.ToLower(issue.SourceType)
	if strings.Contains(src, "preinstall") || strings.Contains(st, "preinstall") {
		return true
	}
	return strings.EqualFold(in.ScanType, "pre-install") || strings.EqualFold(in.ScanType, "preinstall")
}

// AIGeneratedRiskWording returns cautious phrasing for future AI-code findings.
func AIGeneratedRiskWording() string {
	return "Possible AI-generated or low-context code risk"
}
