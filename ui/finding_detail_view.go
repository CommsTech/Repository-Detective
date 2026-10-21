package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// ActionableFindingView enriches a finding for the Finding Detail 2.0 operator brief
// (RD-PRODUCT-001). Top of page answers: what / why / what changes / can RD fix it.
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

	// Finding Detail 2.0 — operator brief
	OperatorHeadline   string
	DependencyKind     string
	FoundIn            string
	WhyCareSummary     string
	ScannerMatchLabel  string
	RuntimeExposure    string
	Exploitability     string
	FixComplexityLabel string
	RegressionRisk     string
	RecommendedAction  string
	CapabilityRows     []FindingCapability
	AfterMergeNote     string
	UniqueEvidence     []DedupedEvidence
	InstanceCount      int
	UniqueEvidenceN    int
	AdvisoryCount      int
	IsDependencyVuln   bool
}

// FindingCapability is one "Repository Detective can / cannot" row.
type FindingCapability struct {
	Label   string
	Allowed bool
	Note    string
}

// DedupedEvidence is one unique evidence body with occurrence count.
type DedupedEvidence struct {
	Text       string
	Count      int
	LatestScan string
}

func buildActionableFindingView(detail store.FindingDetail) ActionableFindingView {
	view := ActionableFindingView{
		Summary:            detail.Title,
		RelatedRuleID:      detail.RuleID,
		PackageName:        detail.PackageName,
		FalsePositiveGuide: "If this is noise or acceptable risk, mark false positive with a reason. Calibration drafts help suppress similar matches in future scans without deleting history.",
		InstanceCount:      len(detail.Instances),
		RuntimeExposure:    "Unknown",
		Exploitability:     "Unknown",
		FixComplexityLabel: "Unknown",
		RegressionRisk:     "Unknown",
		DependencyKind:     "Unknown",
		AfterMergeNote:     "After merge, Repository Detective will rescan and only close the issue if the finding fingerprint disappears.",
	}
	view.WhyItMatters = whyItMatters(detail)
	view.ConfidenceReason = confidenceReason(detail.Confidence)
	view.SeverityReason = severityReason(detail.Severity, detail.Category)
	view.EvidenceKind = evidenceKind(detail)
	view.IssueFilingStatus, view.IssueFilingDetail = issueFilingView(detail)
	view.RecommendedFix, view.VerificationSteps = fixAndVerify(detail)
	view.ScannerMatchLabel = scannerMatchLabel(detail.Confidence)
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

	view.UniqueEvidence = dedupeEvidence(detail.Instances)
	view.UniqueEvidenceN = len(view.UniqueEvidence)
	view.AdvisoryCount = countDistinctAdvisories(detail)
	view.IsDependencyVuln = isDependencyVuln(detail, view)
	view.FoundIn = formatFoundIn(detail)
	view.OperatorHeadline = operatorHeadline(detail, view)
	view.WhyCareSummary = whyCareSummary(detail, view)
	view.RecommendedAction = recommendedActionLine(detail, view)
	enrichAssessment(&view, detail)
	view.CapabilityRows = defaultCapabilities(detail, view)
	return view
}

func operatorHeadline(d store.FindingDetail, v ActionableFindingView) string {
	if v.IsDependencyVuln && v.PackageName != "" {
		name := humanPackageName(v.PackageName)
		return fmt.Sprintf("%s dependency vulnerable", name)
	}
	title := strings.TrimSpace(d.Title)
	if title == "" {
		return "Security finding"
	}
	// Prefer short operator language over raw scanner titles.
	lower := strings.ToLower(title)
	if strings.Contains(lower, "cve-") && v.PackageName != "" {
		return fmt.Sprintf("%s dependency vulnerable", humanPackageName(v.PackageName))
	}
	return title
}

func humanPackageName(pkg string) string {
	pkg = strings.TrimSpace(pkg)
	if i := strings.LastIndex(pkg, "/"); i >= 0 && i+1 < len(pkg) {
		pkg = pkg[i+1:]
	}
	if i := strings.LastIndex(pkg, ":"); i >= 0 && i+1 < len(pkg) {
		// purl-ish pkg:pypi/cryptography
		rest := pkg[i+1:]
		if j := strings.LastIndex(rest, "/"); j >= 0 && j+1 < len(rest) {
			return rest[j+1:]
		}
		return rest
	}
	return pkg
}

func formatFoundIn(d store.FindingDetail) string {
	if d.FilePath == "" {
		if strings.Contains(strings.ToLower(d.Source), "trivy") || strings.Contains(strings.ToLower(d.Source), "grype") {
			return "Container / SBOM inventory (no single source file)"
		}
		return "Repository-level / no single file path"
	}
	path := displayPath(d.FilePath)
	if d.Line > 0 {
		return fmt.Sprintf("%s:%d", path, d.Line)
	}
	return path
}

func whyCareSummary(d store.FindingDetail, v ActionableFindingView) string {
	if v.IsDependencyVuln {
		n := v.AdvisoryCount
		if n <= 0 {
			n = 1
		}
		if n == 1 {
			return "One scanner advisory maps to this dependency problem."
		}
		return fmt.Sprintf("%d scanner advisories map to this one dependency problem.", n)
	}
	if v.UniqueEvidenceN > 1 {
		return fmt.Sprintf("Seen across %d scans with %d unique evidence shapes (duplicates collapsed).", v.InstanceCount, v.UniqueEvidenceN)
	}
	if v.InstanceCount > 1 {
		return fmt.Sprintf("Recurring finding — observed in %d scans; evidence deduplicated for display.", v.InstanceCount)
	}
	return v.WhyItMatters
}

func recommendedActionLine(d store.FindingDetail, v ActionableFindingView) string {
	if v.IsDependencyVuln && v.PackageName != "" {
		pkg := humanPackageName(v.PackageName)
		cur := strings.TrimSpace(v.PackageVersion)
		fix := strings.TrimSpace(v.FixedVersion)
		if cur != "" && fix != "" {
			return fmt.Sprintf("%s==%s → %s>=%s", pkg, cur, pkg, fix)
		}
		if fix != "" {
			return fmt.Sprintf("Upgrade %s to %s (or newer fixed release)", pkg, fix)
		}
		if cur != "" {
			return fmt.Sprintf("Upgrade %s (currently %s) to a patched release", pkg, cur)
		}
		return fmt.Sprintf("Upgrade vulnerable dependency %s", pkg)
	}
	return v.RecommendedFix
}

func enrichAssessment(view *ActionableFindingView, d store.FindingDetail) {
	if view.IsDependencyVuln {
		view.RuntimeExposure = "Unknown — reachability not proven"
		view.Exploitability = "Moderate / Unknown"
		if view.FixedVersion != "" {
			view.FixComplexityLabel = "Low"
			view.RegressionRisk = "Low/Medium"
		} else {
			view.FixComplexityLabel = "Medium"
			view.RegressionRisk = "Medium"
		}
		return
	}
	switch strings.ToLower(d.Category) {
	case "secret":
		view.RuntimeExposure = "Possible if credential is live"
		view.Exploitability = "High if credential is valid"
		view.FixComplexityLabel = "Medium"
		view.RegressionRisk = "Low"
	case "misconfiguration", "iac":
		view.RuntimeExposure = "Depends on deployment"
		view.Exploitability = "Context-dependent"
		view.FixComplexityLabel = "Low/Medium"
		view.RegressionRisk = "Low/Medium"
	default:
		view.RuntimeExposure = "Unknown"
		view.Exploitability = "Unknown"
		view.FixComplexityLabel = "Unknown"
		view.RegressionRisk = "Unknown"
	}
}

func defaultCapabilities(d store.FindingDetail, v ActionableFindingView) []FindingCapability {
	canPR := v.IsDependencyVuln && v.FixedVersion != "" && d.Category != "secret"
	return []FindingCapability{
		{Label: "Create branch", Allowed: canPR, Note: ternary(canPR, "Via approved remediation plan", "Not available for this finding type yet")},
		{Label: "Update dependency / apply fix", Allowed: canPR, Note: ternary(canPR, "Lockfile / manifest patch when plan allows", "Manual fix required")},
		{Label: "Run validation", Allowed: canPR, Note: ternary(canPR, "Allowlisted checks when remediation PR enabled", "Operator-run tests")},
		{Label: "Open PR", Allowed: canPR, Note: ternary(canPR, "Never auto-merges", "Planner / PR path not applicable")},
		{Label: "Merge automatically", Allowed: false, Note: "Repository Detective never merges"},
	}
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func isDependencyVuln(d store.FindingDetail, v ActionableFindingView) bool {
	cat := strings.ToLower(d.Category)
	src := strings.ToLower(d.Source)
	if cat == "vulnerability" || cat == "vuln" || cat == "dependency" {
		return true
	}
	if strings.Contains(src, "trivy") || strings.Contains(src, "grype") {
		return true
	}
	rule := strings.ToUpper(d.RuleID)
	if strings.HasPrefix(rule, "TRIVY-") || strings.HasPrefix(rule, "GRYPE-") || strings.HasPrefix(rule, "CVE-") {
		return true
	}
	return v.PackageName != "" && (v.CVEID != "" || v.FixedVersion != "")
}

func scannerMatchLabel(c float64) string {
	switch {
	case c >= 0.85:
		return "High confidence"
	case c >= 0.6:
		return "Medium confidence"
	default:
		return "Low confidence"
	}
}

func countDistinctAdvisories(d store.FindingDetail) int {
	seen := map[string]struct{}{}
	for _, inst := range d.Instances {
		key := strings.TrimSpace(inst.EvidenceRedacted)
		if key == "" && len(inst.RawMetadataJSON) > 0 {
			key = string(inst.RawMetadataJSON)
		}
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	if len(seen) == 0 {
		if d.RuleID != "" {
			return 1
		}
		return 0
	}
	return len(seen)
}

func dedupeEvidence(instances []store.FindingInstance) []DedupedEvidence {
	order := make([]string, 0)
	byText := map[string]*DedupedEvidence{}
	for _, inst := range instances {
		text := strings.TrimSpace(inst.EvidenceRedacted)
		if text == "" {
			continue
		}
		if existing, ok := byText[text]; ok {
			existing.Count++
			if inst.ScanID != "" {
				existing.LatestScan = inst.ScanID
			}
			continue
		}
		order = append(order, text)
		byText[text] = &DedupedEvidence{Text: text, Count: 1, LatestScan: inst.ScanID}
	}
	out := make([]DedupedEvidence, 0, len(order))
	for _, text := range order {
		out = append(out, *byText[text])
	}
	return out
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
	view.PackageName = firstNonEmpty(view.PackageName, metaString(meta, "package", "PackageName", "pkg_name", "PkgName", "name"))
	view.PackageVersion = metaString(meta, "version", "installed_version", "PackageVersion", "InstalledVersion")
	view.FixedVersion = metaString(meta, "fixed_version", "FixedVersion", "FixedVersionStr")
	view.CVEID = metaString(meta, "cve", "cve_id", "VulnerabilityID", "vulnerability_id", "VulnerabilityID")
	view.ScannerCommand = metaString(meta, "scanner", "tool", "command")
	view.SBOMRelation = metaString(meta, "sbom_component", "purl", "bom_ref")
	dep := strings.ToLower(metaString(meta, "dependency_type", "DependencyType", "relationship", "Relation"))
	switch {
	case strings.Contains(dep, "direct"):
		view.DependencyKind = "Direct"
	case strings.Contains(dep, "trans"):
		view.DependencyKind = "Transitive"
	}
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
