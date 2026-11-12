package issues

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func renderSpecializedFindingSection(issue *ai.CodeIssue, in IssueRenderInput) string {
	meta := parseIssueEvidence(issue)
	var sections []string

	if isHistoricalFinding(issue) || (NormalizeCategory(issue.Category, issue.Source) == CategorySecret && issue.CommitSHA != "") {
		var b strings.Builder
		b.WriteString("**Secret / history**\n\n")
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
		b.WriteString("**Container**\n\n")
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
		b.WriteString("**SBOM**\n\n")
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
		b.WriteString("**Graph**\n\n")
		b.WriteString(fmt.Sprintf("- Node type: `%s`\n", defaultString(evidenceField(meta, "node_type", "NodeType"), "unknown")))
		reach := evidenceField(meta, "entrypoint_reachable", "EntrypointReachable", "path_classification")
		if reach != "" {
			b.WriteString(fmt.Sprintf("- Reachability: %s\n", reach))
		}
		sections = append(sections, b.String())
	}

	if isPreinstallFinding(issue, in) {
		sections = append(sections, "**Pre-install audit**\n\n- Report-only: yes\n")
	}

	return strings.Join(sections, "\n")
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
