package ui

import (
	"fmt"

	"git.commsnet.org/commstech/repository-detective/store"
)

// RepoSettingField is one policy row with human-readable help.
type RepoSettingField struct {
	Key             string
	Label           string
	EffectiveValue  string
	Source          string
	Explanation     string
	BetaNote        string
	Badge           string
	BadgeClass      string
	CanCreateIssues bool
	CanCreatePRs    bool
	RestartRequired bool
	Dangerous       bool
}

// RepoSettingsSection groups related repo policy fields.
type RepoSettingsSection struct {
	ID      string
	Title   string
	Summary string
	Fields  []RepoSettingField
}

func buildRepoSettingsSections(effective store.EffectiveSettings, meta store.EffectiveSettingsMeta, _ store.GlobalSettingsSnapshot) []RepoSettingsSection {
	issueFiling := store.ShouldCreateForgeIssues(effective)
	return []RepoSettingsSection{
		{
			ID: "overview", Title: "Overview",
			Summary: "High-level scan posture for this repository.",
			Fields: []RepoSettingField{
				field("enabled", "Repository enabled", fmt.Sprintf("%v", effective.Enabled), sourceForOverride(meta.ProfileModified), "When false, scans are skipped.", "Keep enabled for active repos.", "Safe beta default", "monitor", false, false, false, false),
				field("scan_profile", "Scan profile", store.ScanProfileLabel(meta.ScanProfile), meta.ProfileSource, store.ScanProfileDescription(meta.ScanProfile), "Standard recommended for day-to-day scans with issue filing.", "Safe default", "monitor", false, false, true, false),
				field("policy_level", "Policy level", effective.PolicyLevel, sourceForOverride(meta.ProfileModified), "monitor_only = report/findings only; issue_only+ may file forge issues.", "Use monitor_only for report-only beta.", badgeForIssueFiling(issueFiling), badgeClassForIssueFiling(issueFiling), issueFiling, false, true, issueFiling),
			},
		},
		{
			ID: "issue-filing", Title: "Issue filing",
			Summary: "Controls whether findings create Gitea/GitHub issues.",
			Fields: []RepoSettingField{
				field("issue_policy", "Issue policy", effective.IssuePolicy, sourceForOverride(meta.ProfileModified), "off = never file; fingerprint = dedupe; all = file broadly.", "Keep off until operator approves.", badgeForIssueFiling(issueFiling), badgeClassForIssueFiling(issueFiling), issueFiling, false, true, issueFiling),
				field("severity_gate", "Severity gate", effective.SeverityGate, sourceForOverride(meta.ProfileModified), "Minimum severity required before filing.", "high recommended for beta.", "Safe beta default", "monitor", false, false, false, false),
				field("confidence_gate", "Confidence gate", fmt.Sprintf("%.2f", effective.ConfidenceGate), sourceForOverride(meta.ProfileModified), "Minimum confidence before filing.", "Raise to reduce noise.", "Safe beta default", "monitor", false, false, false, false),
			},
		},
		{
			ID: "report-only", Title: "Report-only / safety",
			Summary: "Report-only dry-runs persist findings without forge side effects.",
			Fields: []RepoSettingField{
				field("report_only_dry_run", "Report-only dry-run", "API flag per scan", "api", "Set report_only_dry_run:true on manual/API scans.", "Explicit operator choice — unchecked when issue filing is enabled.", "Per-scan override", "monitor", false, false, false, false),
				field("remediation_policy", "Remediation policy", effective.RemediationPolicy, sourceForOverride(meta.ProfileModified), "off/suggest — planning only; does not open PRs by itself.", "Keep off/suggest in beta.", "Safe beta default", "monitor", false, false, false, false),
			},
		},
		{
			ID: "scanners", Title: "Scanners",
			Summary: "Deterministic scanner toggles (effective values after profile merge).",
			Fields: []RepoSettingField{
				boolField("enable_trivy", "Trivy", effective.EnableTrivy, "Dependency/container vulnerabilities."),
				boolField("enable_gitleaks", "Gitleaks", effective.EnableGitleaks, "Secret detection."),
				boolField("enable_staticcheck", "Staticcheck", effective.EnableStaticcheck, "Go static analysis."),
				boolField("enable_code_graph", "Code graph", effective.EnableCodeGraph, "Repository map / risk overlay."),
			},
		},
		{
			ID: "scheduling", Title: "Scheduling",
			Summary: "Cron schedule for automatic scans (manual Scan now always available).",
			Fields: []RepoSettingField{
				field("schedule_enabled", "Schedule enabled", fmt.Sprintf("%v", effective.ScheduleEnabled), sourceForOverride(meta.ProfileModified), "When true, cron triggers periodic scans.", "", "Advanced", "skipped", false, false, true, false),
				field("schedule_cron", "Schedule cron", settingsEmptyOrValue(effective.ScheduleCron), sourceForOverride(meta.ProfileModified), "Standard 5-field cron expression.", "", "Advanced", "skipped", false, false, true, false),
			},
		},
	}
}

func field(key, label, effective, source, explanation, beta, badge, badgeClass string, canIssue, canPR, restart, dangerous bool) RepoSettingField {
	return RepoSettingField{
		Key: key, Label: label, EffectiveValue: effective, Source: source,
		Explanation: explanation, BetaNote: beta, Badge: badge, BadgeClass: badgeClass,
		CanCreateIssues: canIssue, CanCreatePRs: canPR, RestartRequired: restart, Dangerous: dangerous,
	}
}

func boolField(key, label string, on bool, explanation string) RepoSettingField {
	return field(key, label, fmt.Sprintf("%v", on), "effective", explanation, "", "Scanner", "monitor", false, false, false, false)
}

func sourceForOverride(modified bool) string {
	if modified {
		return "repo override"
	}
	return "profile/default"
}

func badgeForIssueFiling(on bool) string {
	if on {
		return "Can create issues"
	}
	return "Safe beta default"
}

func badgeClassForIssueFiling(on bool) string {
	if on {
		return "enforce"
	}
	return "monitor"
}

func settingsEmptyOrValue(v string) string {
	if v == "" {
		return "—"
	}
	return v
}
