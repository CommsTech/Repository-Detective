package ui

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/operator"
)

// CapabilityStatus describes one platform capability for the health page.
type CapabilityStatus struct {
	Name          string
	State         string // enabled | disabled | degraded | unavailable
	Reason        string
	ConfigKeys    []string
	SafetyNote    string
	SettingsURL   string
	SettingsLabel string
}

// PlatformContext holds non-secret wiring used for setup detection and capability status.
type PlatformContext struct {
	GiteaURLConfigured           bool
	GiteaTokenConfigured         bool
	APIKeyConfigured             bool
	WebhookSecretConfigured      bool
	RunnerSharedSecretSet        bool
	RunnerCallbackBaseURL        string
	PublicURL                    string
	RemediationPRRequireApproval bool
	RemediationPRMaxFiles        int
	RemediationPRMaxDiffLines    int
	RemediationPRBranchPrefix    string
	LLMSanityGateEnabled         bool
	BacklogControlEnabled        bool
	MaxIssuesPerScan             int
	ScanPolicyMode               string
	NotificationsEnabled         bool
	SchedulerEnabled             bool
	RunnerDelegationEnabled      bool
	RunnerRequireHMAC            bool
	RunnerMode                   string
	RemediationPRRequireTests    bool
	RemediationPRUseRunnerVerification bool
	GiteaActionsTestBackendEnabled bool
	OpenClawAIReviewEnabled        bool
	OpenClawEndpointConfigured     bool
	ContainerScanningEnabled       bool
	ContainerScanRequireRunner     bool
	ContainerScanAllowCoreSocket   bool
	ContainerScanCreateIssues      bool
}

func buildCapabilityStatuses(
	readiness operator.Readiness,
	notifyCfg notify.Config,
	platform PlatformContext,
	basePath string,
) []CapabilityStatus {
	return []CapabilityStatus{
		remediationPRStatus(readiness, platform, basePath),
		runnerDelegationStatus(readiness, platform, basePath),
		notificationsStatus(readiness, notifyCfg, basePath),
		preinstallAuditStatus(readiness, platform, basePath),
	}
}

func remediationPRStatus(r operator.Readiness, p PlatformContext, basePath string) CapabilityStatus {
	url := configureSectionURL(basePath, "remediation-pr")
	if r.Features.RemediationPREnabled {
		return CapabilityStatus{
			Name: "Remediation PR", State: "enabled",
			Reason:      "Low-risk approved remediation PR flow is available when plans pass validation.",
			ConfigKeys:  []string{"remediation_pr_enabled", "remediation_pr_require_approval", "gitea_token"},
			SafetyNote:  "Broad auto-PR remains off by default; approval gates apply.",
			SettingsURL: url, SettingsLabel: "Configure",
		}
	}
	reason := "Intentionally disabled by default for safety."
	if !p.GiteaTokenConfigured {
		reason = "Requires Gitea token with repository write permission."
	} else if !r.Features.RemediationPlannerEnabled {
		reason = "Remediation planner is disabled — enable remediation_planner_enabled first."
	}
	safety := "Enable only after reviewing remediation planner output and PR size limits."
	if p.RemediationPRRequireTests {
		safety += " Tests must pass before PR creation."
	}
	return CapabilityStatus{
		Name: "Remediation PR", State: "disabled", Reason: reason,
		ConfigKeys:  []string{"remediation_pr_enabled", "remediation_pr_require_approval", "remediation_pr_require_tests", "gitea_token"},
		SafetyNote:  safety,
		SettingsURL: url, SettingsLabel: "Configure",
	}
}

func runnerDelegationStatus(r operator.Readiness, p PlatformContext, basePath string) CapabilityStatus {
	url := configureSectionURL(basePath, "runner-delegation")
	if r.Features.RunnerDelegationEnabled {
		state := "enabled"
		reason := "Runner jobs can be delegated when runners authenticate with the shared secret."
		if !p.RunnerSharedSecretSet {
			state = "degraded"
			reason = "Delegation flag is on but runner_shared_secret is empty — jobs may fail HMAC validation."
		}
		return CapabilityStatus{
			Name: "Runner delegation", State: state, Reason: reason,
			ConfigKeys:  []string{"runner_delegation_enabled", "runner_shared_secret", "runner_callback_base_url", "public_url"},
			SettingsURL: url, SettingsLabel: "Configure",
		}
	}
	reason := "Disabled by default until runner_shared_secret and callback URL are configured."
	if p.RunnerSharedSecretSet && p.PublicURL != "" {
		reason = "Shared secret present — set runner_delegation_enabled=true to activate native runner workers."
	}
	return CapabilityStatus{
		Name: "Runner delegation", State: "disabled", Reason: reason,
		ConfigKeys:  []string{"runner_delegation_enabled", "runner_shared_secret", "runner_mode", "runner_callback_base_url"},
		SafetyNote:  "Native Repository Detective runners handle scans; Gitea act_runner is optional for repo-native test verification only.",
		SettingsURL: url, SettingsLabel: "Configure",
	}
}

func notificationsStatus(r operator.Readiness, cfg notify.Config, basePath string) CapabilityStatus {
	url := configureSectionURL(basePath, "notifications")
	if r.Features.NotificationsEnabled {
		channels := notifyConfiguredChannels(cfg)
		if len(channels) == 0 {
			return CapabilityStatus{
				Name: "Notifications", State: "degraded",
				Reason:      "Global notifications_enabled is true but no channel (webhook/Slack/Discord/Telegram) is configured.",
				ConfigKeys:  []string{"notifications_enabled", "notification_webhook_url", "notification_slack_webhook_url"},
				SettingsURL: url, SettingsLabel: "Configure",
			}
		}
		return CapabilityStatus{
			Name: "Notifications", State: "enabled",
			Reason:      fmt.Sprintf("Active channels: %s", strings.Join(channels, ", ")),
			ConfigKeys:  []string{"notifications_enabled", "notification_min_severity"},
			SettingsURL: url, SettingsLabel: "Configure",
		}
	}
	return CapabilityStatus{
		Name: "Notifications", State: "disabled",
		Reason:      "Set notifications_enabled=true and configure at least one delivery channel.",
		ConfigKeys:  []string{"notifications_enabled", "notification_webhook_url", "notification_slack_webhook_url", "notification_discord_webhook_url"},
		SettingsURL: url, SettingsLabel: "Configure",
	}
}

func preinstallAuditStatus(r operator.Readiness, p PlatformContext, basePath string) CapabilityStatus {
	url := configureSectionURL(basePath, "preinstall-audit")
	if r.Features.PreinstallAuditEnabled {
		reason := "Pre-install audit routes are available."
		state := "enabled"
		if p.PublicURL == "" {
			state = "degraded"
			reason = "Enabled but public_url is unset — report links may be incomplete."
		}
		return CapabilityStatus{
			Name: "Pre-install audit", State: state, Reason: reason,
			ConfigKeys:  []string{"preinstall_audit_enabled", "public_url", "preinstall_allow_private_networks"},
			SafetyNote:  "Private network targets remain blocked unless explicitly allowed.",
			SettingsURL: url, SettingsLabel: "Configure",
		}
	}
	return CapabilityStatus{
		Name: "Pre-install audit", State: "disabled",
		Reason:      "Set preinstall_audit_enabled=true to expose /preinstall audit workflow.",
		ConfigKeys:  []string{"preinstall_audit_enabled", "public_url"},
		SettingsURL: url, SettingsLabel: "Configure",
	}
}

func notifyConfiguredChannels(cfg notify.Config) []string {
	var out []string
	if strings.TrimSpace(cfg.WebhookURL) != "" && cfg.WebhookEnabled {
		out = append(out, "webhook")
	}
	if strings.TrimSpace(cfg.SlackWebhookURL) != "" && cfg.SlackEnabled {
		out = append(out, "Slack")
	}
	if strings.TrimSpace(cfg.DiscordWebhookURL) != "" && cfg.DiscordEnabled {
		out = append(out, "Discord")
	}
	if strings.TrimSpace(cfg.TelegramBotToken) != "" && strings.TrimSpace(cfg.TelegramChatID) != "" && cfg.TelegramEnabled {
		out = append(out, "Telegram")
	}
	return out
}
