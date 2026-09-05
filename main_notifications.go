package main

import (
	"context"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

var notifyManager *notify.Manager

func applyPrivacyToNotificationChannels() {
	mode := privacy.NormalizeMode(config.PrivacyMode)
	if mode != privacy.ModeLocalOnly {
		return
	}
	gate := func(name, rawURL string, enabled *bool) {
		if rawURL == "" || enabled == nil || !*enabled {
			return
		}
		d := privacy.EvaluateURLEgress(mode, rawURL)
		if !d.Allowed {
			logger.Warnf("privacy_mode=local_only disabled %s notifications: %s", name, d.Reason)
			*enabled = false
		}
	}
	gate("slack", config.SlackWebhookURL, &config.SlackEnabled)
	gate("discord", config.DiscordWebhookURL, &config.DiscordEnabled)
	gate("webhook", config.WebhookNotificationURL, &config.WebhookNotificationsEnabled)
	// Telegram Bot API is always an external cloud destination.
	if config.TelegramEnabled {
		logger.Warn("privacy_mode=local_only disabled telegram notifications (Telegram Bot API is EXTERNAL)")
		config.TelegramEnabled = false
	}
}

func initNotifyManager() {
	applyPrivacyToNotificationChannels()
	cfg := notify.GlobalNotificationConfigFromMain(
		config.NotificationsEnabled,
		config.NotificationMinSeverity,
		config.NotificationCooldownSeconds,
		config.PublicURL,
		config.TelegramEnabled, config.TelegramBotToken, config.TelegramChatID,
		config.SlackEnabled, config.SlackWebhookURL,
		config.DiscordEnabled, config.DiscordWebhookURL,
		config.WebhookNotificationsEnabled, config.WebhookNotificationURL, config.WebhookNotificationSecret,
	)
	resolve := func(repositoryID int64) notify.EffectiveSettings {
		repoSettings := store.RepoSettings{}
		if rdStore != nil && repositoryID > 0 {
			if s, err := rdStore.GetRepoSettings(context.Background(), repositoryID); err == nil {
				repoSettings = s
			}
		}
		return notify.ResolveEffective(cfg, repoSettings)
	}
	notifyManager = notify.NewManager(cfg, resolve, logger, nil)
}

type preinstallNotifyBridge struct{}

func (preinstallNotifyBridge) OnAuditComplete(req store.AuditRequest, findingCount int) {
	if notifyManager == nil {
		return
	}
	host := notify.SafeAuditHost(req.RepoURL)
	label := host
	if req.RepoOwner != "" && req.RepoName != "" {
		label = req.RepoOwner + "/" + req.RepoName
	} else if label == "" {
		label = "external-repo"
	}
	evType := notify.EventPreinstallCaution
	summary := "Pre-install audit completed with caution recommendation"
	if req.Recommendation == store.AuditRecommendationDoNotInstall {
		evType = notify.EventPreinstallDoNotInstall
		summary = "Pre-install audit recommends do not install"
	} else if req.Recommendation == store.AuditRecommendationSafe {
		return
	}
	notifyManager.Emit(context.Background(), 0, notify.Event{
		Type:       evType,
		Severity:   auditSeverity(req.Recommendation),
		Repository: label,
		ScanID:     req.AuditID,
		Title:      summary,
		Summary:    summary + " (score=" + itoa(req.RiskScore) + ", findings=" + itoa(findingCount) + ")",
		Counts:     map[string]int{"findings": findingCount},
	})
}

func (preinstallNotifyBridge) OnDisclosureReport(auditID, reportType string) {
	if notifyManager == nil {
		return
	}
	notifyManager.Emit(context.Background(), 0, notify.Event{
		Type:       notify.EventDisclosureReportGenerated,
		Severity:   "info",
		Repository: "preinstall-audit",
		ScanID:     auditID,
		Title:      "Disclosure report generated",
		Summary:    "Report type: " + notify.SanitizeText(reportType),
	})
}

func auditSeverity(recommendation string) string {
	switch recommendation {
	case store.AuditRecommendationDoNotInstall:
		return "critical"
	case store.AuditRecommendationCaution:
		return "high"
	default:
		return "info"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func notifyScanFinish(ctx context.Context, scanCtx *store.ScanContext, repositoryID int64, result *analyzers.AnalysisResult, analysisErr error) {
	if notifyManager == nil || scanCtx == nil {
		return
	}
	repoFull := scanCtx.Owner + "/" + scanCtx.Repo
	if analysisErr != nil {
		evType := notify.EventScanFailed
		if scanCtx.TriggerType == store.TriggerScheduled {
			evType = notify.EventScheduledScanFailed
		}
		notifyManager.Emit(ctx, repositoryID, notify.Event{
			Type:       evType,
			Severity:   "high",
			Repository: repoFull,
			ScanID:     scanCtx.ScanID,
			Title:      "Scan failed",
			Summary:    notify.SanitizeSummary(analysisErr.Error()),
		})
		return
	}
	if result == nil {
		return
	}
	counts, maxSev := severityCounts(filterIssuesWithSuppression(repositoryID, result.Issues))
	if maxSev == "critical" {
		notifyManager.Emit(ctx, repositoryID, notify.Event{
			Type:       notify.EventCriticalFinding,
			Severity:   "critical",
			Repository: repoFull,
			ScanID:     scanCtx.ScanID,
			Title:      notify.FindingSeverityWording("critical"),
			Summary:    "Critical findings detected during scan",
			Counts:     counts,
		})
	}
	if maxSev == "high" || counts["high"] > 0 || counts["critical"] > 0 {
		sev := "high"
		if counts["critical"] > 0 {
			sev = "critical"
		}
		notifyManager.Emit(ctx, repositoryID, notify.Event{
			Type:       notify.EventHighFinding,
			Severity:   sev,
			Repository: repoFull,
			ScanID:     scanCtx.ScanID,
			Title:      notify.FindingSeverityWording("high"),
			Summary:    "High severity findings detected during scan",
			Counts:     counts,
		})
	}
	if len(result.Issues) > 0 && (counts["high"] > 0 || counts["critical"] > 0 || counts["medium"] > 0) {
		notifyManager.Emit(ctx, repositoryID, notify.Event{
			Type:       notify.EventScanCompletedWithFindings,
			Severity:   maxSev,
			Repository: repoFull,
			ScanID:     scanCtx.ScanID,
			Title:      "Scan completed with findings",
			Summary:    "Scan finished with gate-relevant findings",
			Counts:     counts,
		})
	}
}

func severityCounts(issues []ai.CodeIssue) (map[string]int, string) {
	counts := map[string]int{}
	maxRank := 0
	maxSev := ""
	for _, issue := range issues {
		sev := strings.ToLower(strings.TrimSpace(issue.Severity))
		if sev == "" {
			sev = "info"
		}
		counts[sev]++
		r := notify.SeverityRank(sev)
		if r > maxRank {
			maxRank = r
			maxSev = sev
		}
	}
	return counts, maxSev
}

func notifyPRGateFailed(ctx context.Context, repositoryID int64, owner, repo string, scanID string, eval gitea.CommitStatusEvaluation) {
	if notifyManager == nil {
		return
	}
	if eval.State != gitea.CommitStateFailure && eval.State != gitea.CommitStateError {
		return
	}
	notifyManager.Emit(ctx, repositoryID, notify.Event{
		Type:       notify.EventPRGateFailed,
		Severity:   "high",
		Repository: owner + "/" + repo,
		ScanID:     scanID,
		Title:      "PR security gate failed",
		Summary:    notify.SanitizeSummary(eval.Description),
	})
}

func notifyRunnerJobFailed(ctx context.Context, repositoryID int64, repoFull, scanID, detail string) {
	if notifyManager == nil {
		return
	}
	notifyManager.Emit(ctx, repositoryID, notify.Event{
		Type:       notify.EventRunnerJobFailed,
		Severity:   "high",
		Repository: repoFull,
		ScanID:     scanID,
		Title:      "Runner job failed",
		Summary:    notify.SanitizeSummary(detail),
	})
}

func notifyRunnerJobsExpired(ctx context.Context, count int64) {
	if notifyManager == nil || count <= 0 {
		return
	}
	notifyManager.Emit(ctx, 0, notify.Event{
		Type:       notify.EventRunnerJobExpired,
		Severity:   "medium",
		Repository: "runner",
		Title:      "Runner jobs expired",
		Summary:    "Expired runner jobs: " + itoa(int(count)),
		Counts:     map[string]int{"expired": int(count)},
	})
}

var _ preinstall.AuditNotifier = preinstallNotifyBridge{}
