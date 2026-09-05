package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// ActionableFindingView enriches a finding for engineer-actionable UI sections.
type ActionableFindingView struct {
	Summary            string
	WhyItMatters       string
	EvidenceKind       string
	CurrentInTree      string
	CommitSHA          string
	ImageDigest        string
	PackageName        string
	PackageVersion     string
	FixedVersion       string
	CVEID              string
	SBOMRelation       string
	ScannerCommand     string
	ConfidenceReason   string
	SeverityReason     string
	WhyFlagged         string
	RecommendedFix     string
	VerificationSteps  []string
	IssueFilingStatus  string
	IssueFilingDetail  string
	FalsePositiveGuide string
	RelatedRuleID      string
	RawMetadataPretty  string
	HasSecretEvidence  bool
}

func buildActionableFindingView(detail store.FindingDetail) ActionableFindingView {
	view := ActionableFindingView{
		Summary:            detail.Title,
		RelatedRuleID:      detail.RuleID,
		PackageName:        detail.PackageName,
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
	rule := strings.ToUpper(strings.TrimSpace(d.RuleID))
	switch strings.ToLower(d.Category) {
	case "secret":
		fix = "Remove the secret from source, rotate the credential, and purge from git history if exposed."
		steps = []string{"Confirm secret is real and active", "Rotate/revoke credential", "Remove from code and history", "Re-scan to verify fingerprint cleared"}
	case "vulnerability", "vuln", "dependency":
		fix = "Upgrade the affected package/image to a fixed version, or apply the vendor mitigation and re-scan."
		steps = []string{"Confirm the package is reachable at runtime", "Identify fixed version from advisory metadata", "Upgrade lockfile/image and run tests", "Re-scan to clear the CVE fingerprint"}
	case "reliability":
		fix = "Handle the failed operation explicitly (return/log/retry) instead of ignoring errors or omitting timeouts."
		steps = []string{"Reproduce the failure path", "Propagate or log the error with context", "Add/adjust timeouts for network calls", "Cover with a unit or integration test", "Re-scan"}
	case "security":
		fix = "Apply the scanner-recommended hardening (pin action SHAs, sanitize inputs, or remove unsafe APIs)."
		steps = []string{"Confirm the match is not a false positive in tests/docs", "Apply the minimal secure change", "Add a regression test or lint allowlist if intentional", "Re-scan"}
	case "public_release":
		fix = "Replace internal hostnames/IPs with documented placeholders, or keep them only in private ops docs that are excluded from public release scans."
		steps = []string{"Confirm whether the reference is intentional for private deploy docs", "Redact or templatize for public surfaces", "Re-scan"}
	case "optimization", "performance":
		fix = "Reduce hot-path cost only when profiling shows impact; otherwise calibrate as accepted noise for non-hot paths."
		steps = []string{"Confirm the path is performance-sensitive", "Refactor if measurable", "Otherwise mark intentional/false positive with reason", "Re-scan"}
	case "maintainability", "code_quality", "tech_debt", "architecture", "test_gap":
		fix = "Treat as backlog hygiene unless it blocks a release gate — prefer small refactors with tests over drive-by rewrites."
		steps = []string{"Confirm the finding is still present on main", "Decide fix vs calibrate based on release risk", "If fixing, keep the change scoped to the reported location", "Re-scan or suppress with an auditable reason"}
	default:
		fix = "Apply a minimal fix at the reported location; follow the remediation plan when available."
		steps = []string{"Reproduce at listed path", "Apply targeted fix", "Run project tests/CI", "Re-scan or mark false positive with reason"}
	}
	if strings.HasPrefix(rule, "GITLEAKS") || rule == "SEC-HARDCODED-SECRET" {
		fix = "Treat as a credential incident until proven otherwise: rotate, remove from tree/history, and re-scan."
		steps = []string{"Validate the match is not a test fixture or placeholder", "Rotate/revoke if real", "Remove from source and history", "Add allowlist only for intentional fixtures", "Re-scan"}
	}
	if strings.HasPrefix(rule, "TRIVY-") || strings.HasPrefix(rule, "GRYPE-") {
		fix = "Upgrade the vulnerable dependency or base image to a patched version listed in the advisory."
		steps = []string{"Map package to lockfile/Dockerfile", "Upgrade to fixed version", "Rebuild and test", "Re-scan SBOM/image"}
	}
	if strings.Contains(rule, "MUTABLE-ACTION") || strings.Contains(strings.ToLower(d.Title), "mutable-action") {
		fix = "Pin GitHub/Gitea Actions to a full commit SHA with a version comment."
		steps = []string{"Resolve the tag to a commit SHA", "Update the workflow uses: line", "Re-scan workflows"}
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
