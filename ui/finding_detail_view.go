package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/bugbot/store"
)

// ActionableFindingView enriches a finding for engineer-actionable UI sections.
type ActionableFindingView struct {
	Summary              string
	WhyItMatters         string
	EvidenceKind         string
	CurrentInTree        string
	CommitSHA            string
	ImageDigest          string
	PackageName          string
	PackageVersion       string
	FixedVersion         string
	CVEID                string
	SBOMRelation         string
	ScannerCommand       string
	ConfidenceReason     string
	SeverityReason       string
	WhyFlagged           string
	RecommendedFix       string
	VerificationSteps    []string
	IssueFilingStatus    string
	IssueFilingDetail    string
	FalsePositiveGuide   string
	RelatedRuleID        string
	RawMetadataPretty    string
	HasSecretEvidence    bool
}

func buildActionableFindingView(detail store.FindingDetail) ActionableFindingView {
	view := ActionableFindingView{
		Summary:           detail.Title,
		RelatedRuleID:     detail.RuleID,
		PackageName:       detail.PackageName,
		FalsePositiveGuide: "If this is noise or acceptable risk, mark false positive with a reason. Calibration drafts help suppress similar matches in future scans without deleting history.",
	}
	view.WhyItMatters = whyItMatters(detail)
	view.ConfidenceReason = confidenceReason(detail.Confidence)
	view.SeverityReason = severityReason(detail.Severity, detail.Category)
	view.EvidenceKind = evidenceKind(detail)
	view.IssueFilingStatus, view.IssueFilingDetail = issueFilingView(detail)
	view.RecommendedFix, view.VerificationSteps = fixAndVerify(detail)
	if len(detail.Instances) > 0 {
		view.WhyFlagged = strings.TrimSpace(detail.Instances[0].EvidenceRedacted)
		view.RawMetadataPretty = jsonPretty(detail.Instances[0].RawMetadataJSON)
		parseInstanceMetadata(&view, detail.Instances[0])
	}
	if detail.Category == "secret" || detail.Source == "gitleaks" {
		view.HasSecretEvidence = true
	}
	if detail.RuleID == "SEC-HARDCODED-SECRET" {
		view.FalsePositiveGuide = "Static heuristic only — verify the literal is a live credential. Placeholders (e.g. status messages like \"Decryption failed\"), examples, and env-backed config are often false positives. Use the Gitea false-positive template with scan ID."
	}
	if detail.FilePath != "" {
		view.CurrentInTree = "Inspect path in latest scan workspace; re-scan to confirm still present."
	}
	return view
}

func whyItMatters(d store.FindingDetail) string {
	switch strings.ToLower(d.Category) {
	case "secret":
		return "Exposed credentials can allow unauthorized access, data theft, or supply-chain compromise. Rotate and remove from history when confirmed."
	case "vulnerability", "vuln":
		return "Known CVEs in dependencies may be exploitable depending on reachability and deployment context."
	case "misconfiguration", "iac":
		return "Misconfigurations can weaken security controls or leak resources in production."
	case "quality":
		return "Quality issues may indicate maintainability or reliability risk."
	default:
		if d.Severity == "critical" || d.Severity == "high" {
			return "High-severity findings warrant prompt triage before release or deployment."
		}
		return "Review to determine exploitability and operational impact in your environment."
	}
}

func confidenceReason(c float64) string {
	switch {
	case c >= 0.85:
		return "High confidence — scanner match is strong; still verify context in your repo."
	case c >= 0.6:
		return "Medium confidence — worth manual review; may be context-dependent."
	default:
		return "Low confidence — higher false-positive risk; prioritize evidence review."
	}
}

func severityReason(sev, cat string) string {
	return fmt.Sprintf("%s severity for category %s — triage against your risk appetite and exposure.", strings.ToUpper(sev), cat)
}

func evidenceKind(d store.FindingDetail) string {
	src := strings.ToLower(d.Source)
	switch {
	case strings.Contains(src, "gitleaks") || d.Category == "secret":
		return "Secret pattern match (redacted)"
	case strings.Contains(src, "trivy"), strings.Contains(src, "grype"):
		return "Container/package vulnerability"
	case src == "graph":
		return "Repository graph heuristic"
	case strings.Contains(src, "sbom"):
		return "SBOM inventory correlation"
	case strings.Contains(src, "semgrep"), strings.Contains(src, "gosec"):
		return "Static analysis rule match"
	default:
		return "Scanner finding"
	}
}

func issueFilingView(d store.FindingDetail) (status, detail string) {
	if d.Suppressed {
		return "Suppressed", "Finding is calibrated/suppressed — new issues are not filed for this fingerprint."
	}
	if len(d.ExternalIssues) == 0 {
		return "No forge issue linked", "Either filing was disabled, below threshold, dry-run, or not yet reconciled for this fingerprint."
	}
	var parts []string
	for _, ex := range d.ExternalIssues {
		parts = append(parts, fmt.Sprintf("%s #%d (%s)", ex.ForgeType, ex.IssueNumber, ex.IssueURL))
	}
	return "Issue linked", strings.Join(parts, "; ")
}

func fixAndVerify(d store.FindingDetail) (fix string, steps []string) {
	switch strings.ToLower(d.Category) {
	case "secret":
		fix = "Remove the secret from source, rotate the credential, and purge from git history if exposed."
		steps = []string{"Confirm secret is real and active", "Rotate/revoke credential", "Remove from code and history", "Re-scan to verify fingerprint cleared"}
	case "vulnerability", "vuln":
		fix = "Upgrade affected package to a fixed version or apply vendor mitigation."
		steps = []string{"Confirm package is used at runtime", "Check fixed version availability", "Upgrade and run tests", "Re-scan image or lockfile"}
	default:
		fix = "Apply minimal fix at reported location; follow remediation plan when available."
		steps = []string{"Reproduce at listed path", "Apply targeted fix", "Run project tests/CI", "Re-scan or mark false positive with reason"}
	}
	return fix, steps
}

func parseInstanceMetadata(view *ActionableFindingView, inst store.FindingInstance) {
	if len(inst.RawMetadataJSON) == 0 {
		return
	}
	var meta map[string]any
	if err := json.Unmarshal(inst.RawMetadataJSON, &meta); err != nil {
		return
	}
	view.CommitSHA = metaString(meta, "commit", "commit_sha", "CommitSHA")
	view.ImageDigest = metaString(meta, "image_digest", "digest", "ImageID")
	view.PackageName = firstNonEmpty(view.PackageName, metaString(meta, "package", "PackageName", "pkg_name"))
	view.PackageVersion = metaString(meta, "version", "installed_version", "PackageVersion")
	view.FixedVersion = metaString(meta, "fixed_version", "FixedVersion")
	view.CVEID = metaString(meta, "cve", "cve_id", "VulnerabilityID", "vulnerability_id")
	view.ScannerCommand = metaString(meta, "scanner", "tool", "command")
	view.SBOMRelation = metaString(meta, "sbom_component", "purl", "bom_ref")
	if view.ScannerCommand == "" {
		view.ScannerCommand = metaString(meta, "source")
	}
}

func metaString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := fmt.Sprint(v); strings.TrimSpace(s) != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
