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
	}
	for _, row := range rows {
		effective, meta := store.ResolveEffectiveSettingsFull(h.global, row.RawSettings)
		issueFiling := store.ShouldCreateForgeIssues(effective)
		view := RepoControlRowView{
			RepositoryControlRow: row,
			IssueFilingLabel:     issueFilingLabel(issueFiling, effective.IssuePolicy),
			ReportOnlyLabel:      reportOnlyLabel(issueFiling),
		}
		view.ScanEnabled = effective.Enabled
		view.ScheduleEnabled = effective.ScheduleEnabled
		view.IssueFilingOn = issueFiling
		view.ScanProfile = meta.ScanProfile
		view.DefaultReportOnly = !issueFiling
		if !issueFiling || row.DryRunReportOnly {
			view.SkippedReportOnly = row.ReportOnlyFindings
		}
		view.CountsDiffer = !issueFiling || row.DryRunReportOnly || row.ReportOnlyFindings > 0 ||
			row.ScanFindingsTotal != row.ForgeOpenIssues
		out.Rows = append(out.Rows, view)
	}
	return out
}

func issueFilingLabel(on bool, policy string) string {
	if on {
		return "on (" + policy + ")"
	}
	return "off (beta safe)"
}

func reportOnlyLabel(issueFilingOn bool) string {
	if issueFilingOn {
		return "default ON for manual scans"
	}
	return "enforced"
}
