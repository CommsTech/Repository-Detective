package ui

import (
	"git.commsnet.org/commstech/bugbot/store"
)

// RepoControlPageView is template data for the fleet control page.
type RepoControlPageView struct {
	Rows              []RepoControlRowView
	ScanTriggerEnabled bool
	Profiles          []string
	RemediationPREnabled  bool
	LLMSanityGateEnabled  bool
	BacklogControlEnabled bool
	ScanPolicyMode        string
}

// RepoControlRowView is one repository row on /ui/repos.
type RepoControlRowView struct {
	store.RepositoryControlRow
	IssueFilingLabel string
	ReportOnlyLabel  string
	CountsDiffer     bool
}

func (h *Handler) buildRepoControlPage(rows []store.RepositoryControlRow) RepoControlPageView {
	out := RepoControlPageView{
		ScanTriggerEnabled:    h.ScanTriggerEnabled(),
		Profiles:              store.AllowedScanProfiles,
		RemediationPREnabled:  h.remediationPREnabled,
		LLMSanityGateEnabled:  h.platform.LLMSanityGateEnabled,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
		ScanPolicyMode:        h.platform.ScanPolicyMode,
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
		out.Rows = append(out.Rows, view)
	}
	return out
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
