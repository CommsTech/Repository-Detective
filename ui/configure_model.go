package ui

import (
	"strconv"
	"strings"

	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
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
				{Key: "database.path", DisplayValue: "data/repository-detective.db", Source: "config", Hint: "Set database_path in config or REPOSITORY_DETECTIVE_DATABASE_PATH"},
			},
		},
		{
			ID: "scheduler", Title: "Scheduler",
			Status: statusLabel(f.SchedulerEnabled), StatusClass: statusClass(f.SchedulerEnabled),
			Summary: "Cron-driven scheduled repository scans.", RestartRequired: false,
			DocPath: "docs/CONFIGURATION.md",
			Settings: []ConfigureSetting{
				boolSetting("scheduler_enabled", f.SchedulerEnabled),
				boolSetting("schedule_enabled", global.ScheduleEnabled),
				{Key: "schedule_cron", DisplayValue: emptyOrValue(global.ScheduleCron), Source: "live"},
			},
		},
		{
			ID: "runner-delegation", Title: "Runner delegation",
			Status: runnerDelegationConfigureStatus(f.RunnerDelegationEnabled, platform),
			StatusClass: runnerDelegationConfigureClass(f.RunnerDelegationEnabled, platform),
			Summary: "Delegate heavy scans to authenticated Repository Detective native runners (optional Gitea Actions for repo-native tests).",
			SafetyNote: "Disabled by default. Native runners use HMAC auth; Gitea act_runner registration tokens belong in secrets only — never in git.",
			BetaDefault: "disabled", RestartRequired: true, DocPath: "docs/RUNNER_DELEGATION.md",
			Settings: []ConfigureSetting{
				boolSetting("runner_delegation_enabled", f.RunnerDelegationEnabled),
				{Key: "runner_mode", DisplayValue: emptyOrValue(platform.RunnerMode), Source: "config", Hint: "native (scans) or gitea_actions (repo tests) — see docs/RUNNER_DELEGATION.md"},
				secretSetting("runner_shared_secret", platform.RunnerSharedSecretSet),
				boolSetting("runner_require_hmac", platform.RunnerRequireHMAC),
				{Key: "runner_callback_base_url", DisplayValue: emptyOrValue(platform.RunnerCallbackBaseURL), Source: "config"},
				{Key: "public_url", DisplayValue: emptyOrValue(platform.PublicURL), Source: "config"},
				boolSetting("gitea_actions_test_backend_enabled", platform.GiteaActionsTestBackendEnabled),
			},
		},
		{
			ID: "notifications", Title: "Notifications",
			Status: notificationsConfigureStatus(f.NotificationsEnabled, notifyCfg),
			StatusClass: notificationsConfigureClass(f.NotificationsEnabled, notifyCfg),
			Summary: "Webhook, Slack, Discord, or Telegram alerts on scan events.",
			BetaDefault: "disabled until channel configured", RestartRequired: false,
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
			Summary: "Audit third-party repositories before install — marketing/on-ramp flow. Report-only; never files issues or PRs.",
			SafetyNote: "HTTPS public repos only; private IPs blocked unless preinstall_allow_private_networks=true. Disclosure drafts require operator approval before external submission.",
			BetaDefault: "enabled (report-only)", RestartRequired: false, DocPath: "docs/PREINSTALL_AUDIT.md",
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
			Summary: "Generate remediation plans from findings (no automatic PR).", RestartRequired: false,
			DocPath: "docs/REMEDIATION.md",
			Settings: []ConfigureSetting{
				boolSetting("remediation_planner_enabled", f.RemediationPlannerEnabled),
				{Key: "remediation_policy", DisplayValue: global.RemediationPolicy, Source: "live"},
			},
		},
		{
			ID: "remediation-pr", Title: "Remediation PR",
			Status: remediationPRConfigureStatus(f.RemediationPREnabled, f.RemediationPlannerEnabled, platform),
			StatusClass: remediationPRConfigureClass(f.RemediationPREnabled, f.RemediationPlannerEnabled, platform),
			Summary: "Create gated pull requests from approved remediation plans.",
			SafetyNote: "Beta recommendation: keep disabled until planner output is reviewed. Requires operator approval and passing tests when enabled.",
			BetaDefault: "disabled", RestartRequired: false, DocPath: "docs/REMEDIATION_PR.md",
			Settings: []ConfigureSetting{
				boolSetting("remediation_pr_enabled", f.RemediationPREnabled),
				boolSetting("remediation_pr_require_approval", platform.RemediationPRRequireApproval),
				boolSetting("remediation_pr_require_tests", platform.RemediationPRRequireTests),
				boolSetting("remediation_pr_use_runner_verification", platform.RemediationPRUseRunnerVerification),
				secretSetting("gitea_token", platform.GiteaTokenConfigured),
				{Key: "remediation_pr_max_files_changed", DisplayValue: strconv.Itoa(platform.RemediationPRMaxFiles), Source: "config"},
				{Key: "remediation_pr_max_diff_lines", DisplayValue: strconv.Itoa(platform.RemediationPRMaxDiffLines), Source: "config"},
				{Key: "remediation_pr_branch_prefix", DisplayValue: emptyOrValue(platform.RemediationPRBranchPrefix), Source: "config"},
			},
		},
		{
			ID: "evidence-closure", Title: "Evidence closure",
			Status: statusLabel(f.EvidenceClosureEnabled), StatusClass: statusClass(f.EvidenceClosureEnabled),
			Summary: "Verify fixes via rescan evidence before closing findings.", RestartRequired: false,
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
			ID: "ai-recommendations", Title: "AI recommendations",
			Status: openclawConfigureStatus(platform),
			StatusClass: openclawConfigureClass(platform),
			Summary: "Optional second-opinion advisory recommendations on redacted finding summaries — provider-neutral (OpenClaw, OpenAI-compatible, Ollama, custom HTTP JSON).",
			SafetyNote: "Disabled by default. CAH harness selects uncertain findings only. No raw secrets, full source, or PHI/PII. Deterministic scanners remain source of truth.",
			BetaDefault: "disabled", RestartRequired: false, DocPath: "docs/AI_RECOMMENDATIONS.md",
			Settings: []ConfigureSetting{
				boolSetting("ai_recommendations_enabled", platform.OpenClawAIReviewEnabled),
				{Key: "ai_recommendations_provider", DisplayValue: "openclaw", Source: "default", Hint: "openclaw | openai-compatible | ollama | custom-http-json"},
				{Key: "ai_recommendations_endpoint", DisplayValue: openclawEndpointDisplay(platform), Source: "config", Hint: "Falls back to ai_base_url; legacy openclaw_ai_endpoint still honored"},
				{Key: "ai_recommendations_max_tokens_per_scan", DisplayValue: "0 (no calls until set)", Source: "default", Hint: "Set >0 to allow recommendation calls"},
				boolSetting("ai_recommendations_send_source_snippets", false),
				boolSetting("ai_recommendations_send_full_files", false),
				boolSetting("ai_recommendations_redact_secrets", true),
				boolSetting("ai_recommendations_advisory_only", true),
				boolSetting("ai_recommendations_require_operator_approval", true),
				boolSetting("ai_recommendations_use_cah_harness", true),
			},
		},
		{
			ID: "scan-profile", Title: "Scan profile",
			Status: store.ScanProfileLabel(f.ScanProfile), StatusClass: "medium",
			Summary: store.ScanProfileDescription(f.ScanProfile), RestartRequired: false,
			DocPath: "docs/SCAN_PROFILES.md",
			Settings: []ConfigureSetting{
				{Key: "scan_profile", DisplayValue: store.ScanProfileLabel(f.ScanProfile), Source: "live"},
				{Key: "analysis_depth", DisplayValue: strconv.Itoa(global.AnalysisDepth), Source: "live"},
				{Key: "severity_gate", DisplayValue: global.SeverityGate, Source: "live"},
			},
		},
		{
			ID: "sbom", Title: "SBOM generation & checking",
			Status: "enabled when scanners run", StatusClass: "completed",
			Summary: "Generate CycloneDX SBOM during scans; check with grype when available.",
			DocPath: "docs/SBOM.md",
			Settings: []ConfigureSetting{
				boolSetting("enable_grype", global.EnableGrype),
				{Key: "syft / cyclonedx-gomod", DisplayValue: "see System Health scanners", Source: "runtime", Hint: "Bundled in all-in-one/runner images (INSTALL_EXTERNAL_TOOLS=true); required for non-Go SBOMs"},
			},
		},
		{
			ID: "secret-scanning", Title: "Secret scanning",
			Status: statusLabel(global.EnableGitleaks), StatusClass: statusClass(global.EnableGitleaks),
			Summary: "Gitleaks current-tree scan plus optional Git-history scan (labeled gitleaks-history; slower).",
			SafetyNote: "History scanning clones the repository and may take several minutes on large repos. Raw secrets are never stored.",
			BetaDefault: "tree + history on deep scans", RestartRequired: true,
			DocPath: "docs/guides/SECRET_SCANNING_AND_GIT_HISTORY.md",
			Settings: []ConfigureSetting{
				boolSetting("enable_gitleaks", global.EnableGitleaks),
				{Key: "secret_scan_git_history_enabled", DisplayValue: "true (default)", Source: "config", Hint: "Full history on onboarding/scheduled deep scans"},
				{Key: "secret_scan_history_max_commits", DisplayValue: "0 = full history", Source: "config"},
				{Key: "secret_scan_history_timeout_seconds", DisplayValue: "600", Source: "default"},
				{Key: "secret_scan_redact", DisplayValue: "true", Source: "default"},
			},
		},
		{
			ID: "issue-filing", Title: "Issue filing policy",
			Status: issueFilingConfigureStatus(global),
			StatusClass: issueFilingConfigureClass(global),
			Summary: "Connected repo scans file or update Gitea issues when policy allows. Dry run is an explicit per-scan choice.",
			SafetyNote: "Private beta package defaults to report-only (auto_create_issues: false). Production/homelab uses auto_create_issues: true.",
			BetaDefault: "report-only until auto_create_issues enabled",
			DocPath: "docs/SCAN_POLICY.md",
			Settings: []ConfigureSetting{
				{Key: "auto_create_issues", DisplayValue: issuePolicyDisplay(global), Source: "config", Hint: "Maps to global issue_policy all/off"},
				{Key: "scan_policy_mode", DisplayValue: store.DeploymentScanMode(global), Source: "derived"},
				{Key: "reporting.max_issues_per_scan", DisplayValue: strconv.Itoa(platform.MaxIssuesPerScan), Source: "config"},
				boolSetting("dogfood_backlog_control_enabled", platform.BacklogControlEnabled),
			},
		},
		{
			ID: "report-only-dry-run", Title: "Report-only dry run",
			Status: "available", StatusClass: "completed",
			Summary: "Explicit report_only_dry_run skips issue filing for one scan — findings still persist.",
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

func issuePolicyDisplay(global store.GlobalSettingsSnapshot) string {
	if store.ShouldCreateForgeIssues(store.EffectiveFromGlobalSnapshot(global)) {
		return "true (issues enabled)"
	}
	return "false (report-only default)"
}

func issueFilingConfigureStatus(global store.GlobalSettingsSnapshot) string {
	if store.ShouldCreateForgeIssues(store.EffectiveFromGlobalSnapshot(global)) {
		return "enabled"
	}
	return "report-only default"
}

func issueFilingConfigureClass(global store.GlobalSettingsSnapshot) string {
	if store.ShouldCreateForgeIssues(store.EffectiveFromGlobalSnapshot(global)) {
		return "completed"
	}
	return "pending"
}

func openclawConfigureStatus(p PlatformContext) string {
	if p.OpenClawAIReviewEnabled {
		if !p.OpenClawEndpointConfigured {
			return "degraded"
		}
		return "enabled"
	}
	if p.OpenClawEndpointConfigured {
		return "disabled (endpoint ready)"
	}
	return "disabled"
}

func openclawConfigureClass(p PlatformContext) string {
	if p.OpenClawAIReviewEnabled && !p.OpenClawEndpointConfigured {
		return "medium"
	}
	return statusClass(p.OpenClawAIReviewEnabled)
}

func openclawEndpointDisplay(p PlatformContext) string {
	if p.OpenClawEndpointConfigured {
		return "configured (redacted)"
	}
	return "(not set)"
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
