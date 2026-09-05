package ui

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// RepoControlPageView is template data for the fleet control page.
type RepoControlPageView struct {
	Rows                  []RepoControlRowView
	FleetHealth           store.FleetHealthSummary
	ScanTriggerEnabled    bool
	Profiles              []store.ScanProfileOption
	RemediationPREnabled  bool
	LLMSanityGateEnabled  bool
	BacklogControlEnabled bool
	ScanPolicyMode        string
	WebhookSetup          WebhookSetupStatus
}

// WebhookSetupStatus summarizes platform prerequisites for push webhooks.
type WebhookSetupStatus struct {
	Ready      bool
	Issues     []string
	OnboardURL string
	DocsURL    string
}

// RepoControlRowView is one repository row on /ui/repos.
type RepoControlRowView struct {
	store.RepositoryControlRow
	IssueFilingLabel string
	ReportOnlyLabel  string
	CountsDiffer     bool
	ScanStale        bool
	LastWebhookAt    string
	WebhookHint      string
}

func (h *Handler) buildRepoControlPage(rows []store.RepositoryControlRow, fleet store.FleetHealthSummary) RepoControlPageView {
	staleByID := make(map[int64]bool, len(fleet.Rows))
	webhookByID := make(map[int64]string, len(fleet.Rows))
	for _, fr := range fleet.Rows {
		staleByID[fr.RepositoryID] = fr.StaleScan
		if fr.LastWebhookAt != nil {
			webhookByID[fr.RepositoryID] = fr.LastWebhookAt.UTC().Format("2006-01-02 15:04")
		}
	}
	webhookSetup := h.webhookSetupStatus()
	out := RepoControlPageView{
		FleetHealth:           fleet,
		ScanTriggerEnabled:    h.ScanTriggerEnabled(),
		Profiles:              store.PrimaryScanProfileOptions,
		RemediationPREnabled:  h.remediationPREnabled,
		LLMSanityGateEnabled:  h.platform.LLMSanityGateEnabled,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
		ScanPolicyMode:        h.platform.ScanPolicyMode,
		WebhookSetup:          webhookSetup,
	}
	for _, row := range rows {
		effective, meta := store.ResolveEffectiveSettingsFull(h.global, row.RawSettings)
		filing := store.ResolveScanFilingPolicy(store.ScanFilingInput{
			Kind:                  store.ScanKindManual,
			Effective:             effective,
			BacklogControlEnabled: h.platform.BacklogControlEnabled,
			MaxIssuesPerScan:      h.platform.MaxIssuesPerScan,
		})
		view := RepoControlRowView{
			RepositoryControlRow: row,
			IssueFilingLabel:     issueFilingLabel(filing.IssueFilingAllowed, effective.IssuePolicy),
			ReportOnlyLabel:      reportOnlyLabel(filing),
		}
		view.ScanEnabled = effective.Enabled
		view.ScheduleEnabled = effective.ScheduleEnabled
		view.IssueFilingOn = filing.IssueFilingAllowed
		view.ScanProfile = meta.ScanProfile
		view.DefaultReportOnly = filing.DryRunCheckboxDefault
		if !filing.IssueFilingAllowed || row.DryRunReportOnly {
			view.SkippedReportOnly = row.ReportOnlyFindings
		}
		view.CountsDiffer = !filing.IssueFilingAllowed || row.DryRunReportOnly || row.ReportOnlyFindings > 0 ||
			row.ScanFindingsTotal != row.ForgeOpenIssues
		view.ScanStale = staleByID[row.ID]
		view.LastWebhookAt = webhookByID[row.ID]
		view.WebhookHint = webhookRowHint(view.ScanEnabled, view.LastWebhookAt, webhookSetup)
		out.Rows = append(out.Rows, view)
	}
	return out
}

func (h *Handler) webhookSetupStatus() WebhookSetupStatus {
	status := WebhookSetupStatus{
		Ready:      true,
		OnboardURL: strings.TrimSuffix(h.basePath, "/ui") + "/onboard",
		DocsURL:    "https://git.commsnet.org/commstech/Repository-Detective/src/branch/main/docs/SETUP.md",
	}
	if strings.TrimSpace(h.platform.PublicURL) == "" {
		status.Ready = false
		status.Issues = append(status.Issues, "PUBLIC_URL is not set — Gitea cannot reach Repository Detective for push webhooks.")
	}
	if !h.platform.WebhookSecretConfigured {
		status.Ready = false
		status.Issues = append(status.Issues, "Webhook secret is missing — set REPOSITORY_DETECTIVE_WEBHOOK_SECRET and use the same value in Gitea.")
	}
	if !h.platform.GiteaURLConfigured || !h.platform.GiteaTokenConfigured {
		status.Ready = false
		status.Issues = append(status.Issues, "Gitea URL/token is incomplete — forge connection is required to register and verify webhooks.")
	}
	return status
}

func webhookRowHint(scanEnabled bool, lastWebhookAt string, setup WebhookSetupStatus) string {
	if !scanEnabled {
		return ""
	}
	if !setup.Ready {
		return "Scanning is on for manual/API scans, but push webhooks are not fully configured yet."
	}
	if strings.TrimSpace(lastWebhookAt) == "" {
		return "No push webhook received yet — register the webhook in Onboard or Gitea repo settings."
	}
	return ""
}

func (h *Handler) buildFleetScanFormPlaceholder() ScanFormView {
	effective, meta := store.ResolveEffectiveSettingsFull(h.global, store.RepoSettings{})
	return h.buildScanFormView(store.Repository{}, effective, meta)
}

func issueFilingLabel(on bool, policy string) string {
	if on {
		return "on (" + policy + ")"
	}
	return "off (policy)"
}

func reportOnlyLabel(filing store.ScanFilingPolicy) string {
	if !filing.IssueFilingAllowed {
		return "enforced"
	}
	if filing.DryRunCheckboxDefault {
		return "optional (dry run default)"
	}
	return "off unless checked"
}
