package store

import (
	"fmt"
	"strings"
)

// Scan kind distinguishes how filing policy is resolved.
type ScanKind string

const (
	ScanKindConnectedRepo ScanKind = "connected_repo"
	ScanKindManual        ScanKind = "manual"
	ScanKindScheduled     ScanKind = "scheduled"
	ScanKindWebhook       ScanKind = "webhook"
	ScanKindPreinstall    ScanKind = "preinstall"
)

// Deployment scan modes (global posture).
const (
	ScanModePrivateBetaSafe      = "private_beta_safe"
	ScanModeProductionSelfHosted = "production_self_hosted"
	ScanModeReportOnlyDryRun     = "report_only_dry_run"
	ScanModePreinstallAudit      = "preinstall_audit"
)

// ScanFilingInput captures request + settings for filing resolution.
type ScanFilingInput struct {
	Kind                  ScanKind
	Effective             EffectiveSettings
	RequestDryRun         bool
	BacklogControlEnabled bool
	MaxIssuesPerScan      int
}

// ScanFilingPolicy is the resolved filing behavior for one scan.
type ScanFilingPolicy struct {
	Mode                  string
	IssueFilingAllowed    bool
	ReportOnlyDryRun      bool
	WillFileIssues        bool
	WillCreatePRs         bool
	IssuesPreflightLine   string
	ReportOnlyPreflight   string
	DryRunCheckboxDefault bool
	DryRunCheckboxLocked  bool
	Reason                string
}

// DeploymentScanMode derives global posture from settings snapshot.
func DeploymentScanMode(global GlobalSettingsSnapshot) string {
	effective := EffectiveFromGlobalSnapshot(global)
	if !ShouldCreateForgeIssues(effective) {
		return ScanModePrivateBetaSafe
	}
	return ScanModeProductionSelfHosted
}

// ResolveScanFilingPolicy computes effective issue filing for a scan.
func ResolveScanFilingPolicy(in ScanFilingInput) ScanFilingPolicy {
	if in.Kind == ScanKindPreinstall {
		return ScanFilingPolicy{
			Mode:                  ScanModePreinstallAudit,
			IssueFilingAllowed:    false,
			ReportOnlyDryRun:      true,
			WillFileIssues:        false,
			WillCreatePRs:         false,
			IssuesPreflightLine:   "Pre-install audits are always report-only — no forge issues will be created.",
			ReportOnlyPreflight:   "Report-only enforced (pre-install audit).",
			DryRunCheckboxDefault: true,
			DryRunCheckboxLocked:  true,
			Reason:                "preinstall_audit",
		}
	}

	filingAllowed := ShouldCreateForgeIssues(in.Effective)
	reportOnly := in.RequestDryRun || !filingAllowed

	mode := ScanModeProductionSelfHosted
	reason := "repo_policy"
	if !filingAllowed {
		mode = ScanModePrivateBetaSafe
		reason = "issue_policy_disabled"
	} else if in.RequestDryRun {
		mode = ScanModeReportOnlyDryRun
		reason = "operator_dry_run"
	}

	willFile := filingAllowed && !reportOnly
	issuesLine := issuesPreflightLine(willFile, filingAllowed, in.BacklogControlEnabled, in.MaxIssuesPerScan)
	reportLine := reportOnlyPreflightLine(reportOnly, filingAllowed)

	return ScanFilingPolicy{
		Mode:                  mode,
		IssueFilingAllowed:    filingAllowed,
		ReportOnlyDryRun:      reportOnly,
		WillFileIssues:        willFile,
		WillCreatePRs:         false,
		IssuesPreflightLine:   issuesLine,
		ReportOnlyPreflight:   reportLine,
		DryRunCheckboxDefault: reportOnly,
		DryRunCheckboxLocked:  !filingAllowed,
		Reason:                reason,
	}
}

func issuesPreflightLine(willFile, filingAllowed, backlog bool, maxIssues int) string {
	if !filingAllowed {
		return "Issue filing is disabled by repo/global policy — this scan will not create or update forge issues."
	}
	if !willFile {
		return "This scan is report-only and will not file or update Gitea issues."
	}
	line := "This scan will file or update Gitea issues for eligible findings (duplicates prevented by fingerprint)."
	if backlog {
		line += " Backlog-control may block new low/medium issues."
	}
	if maxIssues > 0 {
		line += fmt.Sprintf(" Max issues per scan: %d.", maxIssues)
	}
	return line
}

func reportOnlyPreflightLine(reportOnly, filingAllowed bool) string {
	if !filingAllowed {
		return "Report-only enforced — issue filing is off in policy."
	}
	if reportOnly {
		return "Dry run selected — findings persist, reports generate, issue_sync_status will be skipped."
	}
	return "Dry run not selected — issue filing follows repo policy when eligible."
}

// ScannerSummaryLabel returns a compact enabled-scanner summary for preflight UI.
func ScannerSummaryLabel(e EffectiveSettings) string {
	scanners := EnabledScannersList(e)
	if len(scanners) == 0 {
		return "none"
	}
	return strings.Join(scanners, ", ")
}
