package ui

import (
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
)

const systemHealthIssueTemplate = "system_health.md"

// HealthReportLink is a prefilled Gitea new-issue URL plus copyable body text.
type HealthReportLink struct {
	Label string
	URL   string
	Title string
	Body  string
	Kind  string // tool | scanner_failure | failed_scan | capability | metric
}

// BuildScannerFailureReport builds a product-repo issue prefill for one scanner run failure.
func BuildScannerFailureReport(ev store.ScannerFailureEvent, version, publicUIBase string) HealthReportLink {
	title := fmt.Sprintf("[Health] Scanner run failure: %s (%s)", ev.ScannerName, ev.Status)
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("Scanner `%s` reported status `%s` during a repository scan.\n\n", ev.ScannerName, ev.Status))
	b.WriteString("## Details\n\n")
	writeHealthCommonMeta(&b, version, publicUIBase)
	b.WriteString(fmt.Sprintf("- Scanner: `%s`\n", ev.ScannerName))
	b.WriteString(fmt.Sprintf("- Status: `%s`\n", ev.Status))
	b.WriteString(fmt.Sprintf("- Repository: `%s`\n", ev.RepoFullName))
	b.WriteString(fmt.Sprintf("- Scan ID: `%s`\n", ev.ScanID))
	if !ev.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("- Scan started (UTC): `%s`\n", ev.StartedAt.UTC().Format(time.RFC3339)))
	}
	if ev.DurationMS > 0 {
		b.WriteString(fmt.Sprintf("- Duration: `%dms`\n", ev.DurationMS))
	}
	if errText := redactHealthText(ev.Error); errText != "" {
		b.WriteString(fmt.Sprintf("- Error: `%s`\n", errText))
	}
	if detail := redactHealthText(ev.Detail); detail != "" {
		b.WriteString(fmt.Sprintf("- Detail: `%s`\n", detail))
	}
	if publicUIBase != "" && ev.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan URL: %s/scans/%s\n", strings.TrimRight(publicUIBase, "/"), ev.ScanID))
	}
	b.WriteString("\n## Expected behavior\n\n")
	b.WriteString("Configured scanners complete successfully (or degrade with a clear, actionable status).\n\n")
	b.WriteString("## Actual behavior\n\n")
	b.WriteString(fmt.Sprintf("`%s` ended with `%s`.\n\n", ev.ScannerName, ev.Status))
	b.WriteString("## Extra notes (operator)\n\n")
	b.WriteString("<!-- Add anything else that helps triage. Do not paste secrets. -->\n")
	body := b.String()
	return HealthReportLink{
		Label: "Report issue",
		Title: title,
		Body:  body,
		URL:   giteaNewIssueURL(defaultProductIssueBase, systemHealthIssueTemplate, title, body),
		Kind:  "scanner_failure",
	}
}

// BuildFailedScanReport builds a product-repo issue prefill for a failed repository scan.
func BuildFailedScanReport(brief store.FailedScanBrief, version, publicUIBase string) HealthReportLink {
	title := fmt.Sprintf("[Health] Failed repository scan: %s", brief.RepoFullName)
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("Repository scan for `%s` failed.\n\n", brief.RepoFullName))
	b.WriteString("## Details\n\n")
	writeHealthCommonMeta(&b, version, publicUIBase)
	b.WriteString(fmt.Sprintf("- Repository: `%s`\n", brief.RepoFullName))
	b.WriteString(fmt.Sprintf("- Scan ID: `%s`\n", brief.ScanID))
	if brief.Bucket != "" {
		b.WriteString(fmt.Sprintf("- Failure bucket: `%s`\n", brief.Bucket))
	}
	if !brief.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("- Scan started (UTC): `%s`\n", brief.StartedAt.UTC().Format(time.RFC3339)))
	}
	if errText := redactHealthText(brief.Error); errText != "" {
		b.WriteString(fmt.Sprintf("- Error: `%s`\n", errText))
	}
	if publicUIBase != "" && brief.ScanID != "" {
		b.WriteString(fmt.Sprintf("- Scan URL: %s/scans/%s\n", strings.TrimRight(publicUIBase, "/"), brief.ScanID))
	}
	b.WriteString("\n## Expected behavior\n\nScan completes and persists findings.\n\n")
	b.WriteString("## Actual behavior\n\nScan status is `failed`.\n\n")
	b.WriteString("## Extra notes (operator)\n\n<!-- Add context. Do not paste secrets. -->\n")
	body := b.String()
	return HealthReportLink{
		Label: "Report issue",
		Title: title,
		Body:  body,
		URL:   giteaNewIssueURL(defaultProductIssueBase, systemHealthIssueTemplate, title, body),
		Kind:  "failed_scan",
	}
}

// BuildToolHealthReport builds a product-repo issue prefill for a scanner availability problem.
func BuildToolHealthReport(tool operator.ToolStatus, version, publicUIBase string) HealthReportLink {
	title := fmt.Sprintf("[Health] Scanner availability: %s (%s)", tool.Name, tool.StatusState)
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("System Health reports scanner `%s` as `%s`.\n\n", tool.Name, tool.StatusState))
	b.WriteString("## Details\n\n")
	writeHealthCommonMeta(&b, version, publicUIBase)
	b.WriteString(fmt.Sprintf("- Scanner: `%s`\n", tool.Name))
	b.WriteString(fmt.Sprintf("- Enabled in config: `%v`\n", tool.EnabledInConfig || tool.Configured))
	b.WriteString(fmt.Sprintf("- Binary installed: `%v`\n", tool.BinaryInstalled || tool.Available))
	b.WriteString(fmt.Sprintf("- Status: `%s`\n", tool.StatusState))
	if v := strings.TrimSpace(tool.VersionDisplay()); v != "" && v != "—" {
		b.WriteString(fmt.Sprintf("- Version display: `%s`\n", v))
	}
	if hint := strings.TrimSpace(tool.RemediationHint()); hint != "" {
		b.WriteString(fmt.Sprintf("- Hint: %s\n", hint))
	}
	if publicUIBase != "" {
		b.WriteString(fmt.Sprintf("- Health URL: %s/health\n", strings.TrimRight(publicUIBase, "/")))
	}
	b.WriteString("\n## Expected behavior\n\nEnabled scanners are installed and report a real version string.\n\n")
	b.WriteString("## Actual behavior\n\n")
	b.WriteString(fmt.Sprintf("`%s` is `%s`.\n\n", tool.Name, tool.StatusState))
	b.WriteString("## Extra notes (operator)\n\n<!-- Add image/tag, PATH notes, etc. Do not paste secrets. -->\n")
	body := b.String()
	return HealthReportLink{
		Label: "Report issue",
		Title: title,
		Body:  body,
		URL:   giteaNewIssueURL(defaultProductIssueBase, systemHealthIssueTemplate, title, body),
		Kind:  "tool",
	}
}

// BuildCapabilityHealthReport builds a product-repo issue prefill for a degraded capability.
func BuildCapabilityHealthReport(cap CapabilityStatus, version, publicUIBase string) HealthReportLink {
	title := fmt.Sprintf("[Health] Capability %s: %s", cap.Name, cap.State)
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("Platform capability `%s` is `%s`.\n\n", cap.Name, cap.State))
	b.WriteString("## Details\n\n")
	writeHealthCommonMeta(&b, version, publicUIBase)
	b.WriteString(fmt.Sprintf("- Capability: `%s`\n", cap.Name))
	b.WriteString(fmt.Sprintf("- State: `%s`\n", cap.State))
	if cap.Reason != "" {
		b.WriteString(fmt.Sprintf("- Reason: %s\n", cap.Reason))
	}
	if cap.SafetyNote != "" {
		b.WriteString(fmt.Sprintf("- Safety note: %s\n", cap.SafetyNote))
	}
	if len(cap.ConfigKeys) > 0 {
		b.WriteString(fmt.Sprintf("- Config keys: `%s`\n", strings.Join(cap.ConfigKeys, "`, `")))
	}
	if publicUIBase != "" {
		b.WriteString(fmt.Sprintf("- Health URL: %s/health\n", strings.TrimRight(publicUIBase, "/")))
	}
	b.WriteString("\n## Expected behavior\n\nCapability is enabled and healthy for this deployment.\n\n")
	b.WriteString("## Actual behavior\n\n")
	b.WriteString(fmt.Sprintf("`%s` is `%s`.\n\n", cap.Name, cap.State))
	b.WriteString("## Extra notes (operator)\n\n<!-- Add config intent / recent changes. Do not paste secrets. -->\n")
	body := b.String()
	return HealthReportLink{
		Label: "Report issue",
		Title: title,
		Body:  body,
		URL:   giteaNewIssueURL(defaultProductIssueBase, systemHealthIssueTemplate, title, body),
		Kind:  "capability",
	}
}

func writeHealthCommonMeta(b *strings.Builder, version, publicUIBase string) {
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	b.WriteString(fmt.Sprintf("- Repository Detective version: `%s`\n", version))
	b.WriteString(fmt.Sprintf("- Product issue repo: `%s`\n", defaultProductIssueBase))
	if publicUIBase != "" {
		b.WriteString(fmt.Sprintf("- UI base: %s\n", strings.TrimRight(publicUIBase, "/")))
	}
}

func redactHealthText(value string) string {
	value = issues.SanitizeSecretEvidence(strings.TrimSpace(value))
	value = scrubLegacyBrand(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 400 {
		return value[:400] + "…"
	}
	return value
}

func healthPublicUIBase(platformPublicURL, basePath string) string {
	base := strings.TrimRight(strings.TrimSpace(platformPublicURL), "/")
	if base == "" {
		return strings.TrimRight(strings.TrimSpace(basePath), "/")
	}
	path := strings.TrimSpace(basePath)
	if path == "" || path == "/" {
		return base + "/ui"
	}
	if strings.HasSuffix(base, path) {
		return base
	}
	return base + path
}
