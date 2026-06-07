package ui

import (
	"strconv"
	"strings"

	"git.commsnet.org/commstech/bugbot/notify"
	"git.commsnet.org/commstech/bugbot/operator"
	"git.commsnet.org/commstech/bugbot/store"
)

// ConfigureSetting is one config key row on the Configure page.
type ConfigureSetting struct {
	Key          string
	DisplayValue string
	Source       string
	Secret       bool
	Present      bool
	Hint         string
}

// ConfigureSection is an anchor-linked feature block on the Configure page.
type ConfigureSection struct {
	ID              string
	Title           string
	Status          string
	StatusClass     string
	Summary         string
	SafetyNote      string
	BetaDefault     string
	RestartRequired bool
	DocPath         string
	Settings        []ConfigureSetting
	WorkflowURL     string
	WorkflowLabel   string
}

func buildConfigureSections(
	readiness operator.Readiness,
	platform PlatformContext,
	notifyCfg notify.Config,
	global store.GlobalSettingsSnapshot,
	basePath string,
) []ConfigureSection {
	f := readiness.Features
	return []ConfigureSection{
		{
			ID: "database", Title: "Database",
			Status: statusLabel(f.DatabaseEnabled), StatusClass: statusClass(f.DatabaseEnabled),
			Summary: "SQLite persistence for scans, findings, and operator UI.", RestartRequired: true,
			DocPath: "docs/CONFIGURATION.md",
			Settings: []ConfigureSetting{
				boolSetting("database_enabled", f.DatabaseEnabled),
				{Key: "database.path", DisplayValue: "data/bugbot.db (default)", Source: "config", Hint: "Set database_path in config or DATABASE_PATH env"},
			},
		},
		{
			ID: "scheduler", Title: "Scheduler",
			Status: statusLabel(f.SchedulerEnabled), StatusClass: statusClass(f.SchedulerEnabled),
			Summary: "Cron-driven scheduled repository scans.", RestartRequired: true,
			DocPath: "docs/CONFIGURATION.md",
			Settings: []ConfigureSetting{
				boolSetting("scheduler_enabled", f.SchedulerEnabled),
				boolSetting("schedule_enabled", global.ScheduleEnabled),
				{Key: "schedule_cron", DisplayValue: emptyOrValue(global.ScheduleCron), Source: "config"},
			},
		},
		{
			ID: "runner-delegation", Title: "Runner delegation",
			Status: runnerDelegationConfigureStatus(f.RunnerDelegationEnabled, platform),
			StatusClass: runnerDelegationConfigureClass(f.RunnerDelegationEnabled, platform),
			Summary: "Delegate heavy scans to authenticated external runners.",
			SafetyNote: "Disabled by default until runner_shared_secret and callback URL are configured.",
			BetaDefault: "disabled", RestartRequired: true, DocPath: "docs/RUNNERS.md",
			Settings: []ConfigureSetting{
				boolSetting("runner_delegation_enabled", f.RunnerDelegationEnabled),
				secretSetting("runner_shared_secret", platform.RunnerSharedSecretSet),
				{Key: "runner_callback_base_url", DisplayValue: emptyOrValue(platform.RunnerCallbackBaseURL), Source: "config"},
				{Key: "public_url", DisplayValue: emptyOrValue(platform.PublicURL), Source: "config"},
			},
		},
		{
			ID: "notifications", Title: "Notifications",
			Status: notificationsConfigureStatus(f.NotificationsEnabled, notifyCfg),
			StatusClass: notificationsConfigureClass(f.NotificationsEnabled, notifyCfg),
			Summary: "Webhook, Slack, Discord, or Telegram alerts on scan events.",
			BetaDefault: "disabled until channel configured", RestartRequired: true,
			DocPath: "docs/CONFIGURATION.md",
			Settings: []ConfigureSetting{
				boolSetting("notifications_enabled", f.NotificationsEnabled),
				secretSetting("notification_webhook_url", strings.TrimSpace(notifyCfg.WebhookURL) != ""),
				secretSetting("notification_slack_webhook_url", strings.TrimSpace(notifyCfg.SlackWebhookURL) != ""),
				secretSetting("notification_discord_webhook_url", strings.TrimSpace(notifyCfg.DiscordWebhookURL) != ""),
				secretSetting("notification_telegram_bot_token", strings.TrimSpace(notifyCfg.TelegramBotToken) != ""),
			},
		},
		{
			ID: "preinstall-audit", Title: "Pre-install audit",
			Status: preinstallConfigureStatus(f.PreinstallAuditEnabled, platform),
			StatusClass: preinstallConfigureClass(f.PreinstallAuditEnabled, platform),
			Summary: "Audit third-party repositories before install. Report-only — does not file issues.",
			SafetyNote: "HTTPS public repos only; private IPs blocked unless preinstall_allow_private_networks=true.",
			BetaDefault: "disabled", RestartRequired: true, DocPath: "docs/PREINSTALL_AUDIT.md",
			WorkflowURL: basePath + "/preinstall", WorkflowLabel: "Open pre-install audit workflow",
			Settings: []ConfigureSetting{
				boolSetting("preinstall_audit_enabled", f.PreinstallAuditEnabled),
				{Key: "public_url", DisplayValue: emptyOrValue(platform.PublicURL), Source: "config", Hint: "Required for shareable report links"},
				{Key: "preinstall_allow_private_networks", DisplayValue: "false (recommended)", Source: "default", Hint: "Set true only for trusted internal registries"},
			},
		},
		{
			ID: "remediation-planner", Title: "Remediation planner",
			Status: statusLabel(f.RemediationPlannerEnabled), StatusClass: statusClass(f.RemediationPlannerEnabled),
			Summary: "Generate remediation plans from findings (no automatic PR).", RestartRequired: true,
			DocPath: "docs/REMEDIATION.md",
			Settings: []ConfigureSetting{
				boolSetting("remediation_planner_enabled", f.RemediationPlannerEnabled),
				{Key: "remediation_policy", DisplayValue: global.RemediationPolicy, Source: "config"},
			},
		},
		{
			ID: "remediation-pr", Title: "Remediation PR",
			Status: remediationPRConfigureStatus(f.RemediationPREnabled, f.RemediationPlannerEnabled, platform),
			StatusClass: remediationPRConfigureClass(f.RemediationPREnabled, f.RemediationPlannerEnabled, platform),
			Summary: "Create gated pull requests from approved remediation plans.",
			SafetyNote: "Beta recommendation: keep disabled until planner output is reviewed. Approval gate applies when enabled.",
			BetaDefault: "disabled", RestartRequired: true, DocPath: "docs/REMEDIATION.md",
			Settings: []ConfigureSetting{
				boolSetting("remediation_pr_enabled", f.RemediationPREnabled),
				boolSetting("remediation_pr_require_approval", platform.RemediationPRRequireApproval),
				secretSetting("gitea_token", platform.GiteaTokenConfigured),
				{Key: "remediation_pr_max_files_changed", DisplayValue: strconv.Itoa(platform.RemediationPRMaxFiles), Source: "config"},
				{Key: "remediation_pr_max_diff_lines", DisplayValue: strconv.Itoa(platform.RemediationPRMaxDiffLines), Source: "config"},
				{Key: "remediation_pr_branch_prefix", DisplayValue: emptyOrValue(platform.RemediationPRBranchPrefix), Source: "config"},
			},
		},
		{
			ID: "evidence-closure", Title: "Evidence closure",
			Status: statusLabel(f.EvidenceClosureEnabled), StatusClass: statusClass(f.EvidenceClosureEnabled),
			Summary: "Verify fixes via rescan evidence before closing findings.", RestartRequired: true,
			DocPath: "docs/CLOSURE.md",
			Settings: []ConfigureSetting{
				boolSetting("evidence_closure_enabled", f.EvidenceClosureEnabled),
			},
		},
		{
			ID: "operator-ui", Title: "Operator UI",
			Status: statusLabel(f.UIEnabled), StatusClass: statusClass(f.UIEnabled),
			Summary: "Web control plane for operators.", RestartRequired: true,
			Settings: []ConfigureSetting{
				boolSetting("operator_ui_enabled", f.UIEnabled),
				secretSetting("repository_detective_api_key", platform.APIKeyConfigured),
			},
		},
		{
			ID: "scan-profile", Title: "Scan profile",
			Status: f.ScanProfile, StatusClass: "medium",
			Summary: "Deterministic scanner bundle and calibration defaults.", RestartRequired: true,
			DocPath: "docs/SCAN_PROFILES.md",
			Settings: []ConfigureSetting{
				{Key: "scan_profile", DisplayValue: f.ScanProfile, Source: "config"},
				{Key: "analysis_depth", DisplayValue: strconv.Itoa(global.AnalysisDepth), Source: "config"},
				{Key: "severity_gate", DisplayValue: global.SeverityGate, Source: "config"},
			},
		},
		{
			ID: "sbom", Title: "SBOM generation & checking",
			Status: "enabled when scanners run", StatusClass: "completed",
			Summary: "Generate CycloneDX SBOM during scans; check with grype when available.",
			DocPath: "docs/SBOM.md",
			Settings: []ConfigureSetting{
				boolSetting("enable_grype", global.EnableGrype),
				{Key: "syft / cyclonedx-gomod", DisplayValue: "see System Health scanners", Source: "runtime", Hint: "Installed in container when INSTALL_EXTERNAL_TOOLS=true"},
			},
		},
		{
			ID: "report-only-dry-run", Title: "Report-only dry run",
			Status: "available", StatusClass: "completed",
			Summary: "API flag report_only_dry_run skips issue filing — safe for non-product repos.",
			SafetyNote: "Required for calibration dry-runs; do not enable bulk issue filing without approval.",
			DocPath: "docs/dogfood-reports/non-product-dry-run-next-gate.md",
			Settings: []ConfigureSetting{
				{Key: "report_only_dry_run", DisplayValue: "API request field", Source: "api", Hint: "POST /api/v1/analyze with report_only_dry_run: true"},
			},
		},
	}
}

func boolSetting(key string, on bool) ConfigureSetting {
	val := "false"
	if on {
		val = "true"
	}
	return ConfigureSetting{Key: key, DisplayValue: val, Source: "config"}
}

func secretSetting(key string, present bool) ConfigureSetting {
	display := "missing"
	if present {
		display = "present (redacted)"
	}
	return ConfigureSetting{Key: key, DisplayValue: display, Source: "config/env", Secret: true, Present: present}
}

func statusLabel(on bool) string {
	if on {
		return "enabled"
	}
	return "disabled"
}

func statusClass(on bool) string {
	if on {
		return "completed"
	}
	return "skipped"
}

func emptyOrValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(not set)"
	}
	return v
}

func runnerDelegationConfigureStatus(on bool, p PlatformContext) string {
	if on && !p.RunnerSharedSecretSet {
		return "degraded"
	}
	return statusLabel(on)
}

func runnerDelegationConfigureClass(on bool, p PlatformContext) string {
	if on && !p.RunnerSharedSecretSet {
		return "medium"
	}
	return statusClass(on)
}

func notificationsConfigureStatus(on bool, cfg notify.Config) string {
	if on && len(notifyConfiguredChannels(cfg)) == 0 {
		return "degraded"
	}
	return statusLabel(on)
}

func notificationsConfigureClass(on bool, cfg notify.Config) string {
	if on && len(notifyConfiguredChannels(cfg)) == 0 {
		return "medium"
	}
	return statusClass(on)
}

func preinstallConfigureStatus(on bool, p PlatformContext) string {
	if on && strings.TrimSpace(p.PublicURL) == "" {
		return "degraded"
	}
	return statusLabel(on)
}

func preinstallConfigureClass(on bool, p PlatformContext) string {
	if on && strings.TrimSpace(p.PublicURL) == "" {
		return "medium"
	}
	return statusClass(on)
}

func remediationPRConfigureStatus(prOn, plannerOn bool, p PlatformContext) string {
	if prOn {
		return "enabled"
	}
	if !plannerOn {
		return "disabled (planner off)"
	}
	if !p.GiteaTokenConfigured {
		return "disabled (token missing)"
	}
	return "disabled"
}

func remediationPRConfigureClass(prOn, plannerOn bool, p PlatformContext) string {
	if prOn {
		return "completed"
	}
	if !plannerOn || !p.GiteaTokenConfigured {
		return "medium"
	}
	return "skipped"
}
