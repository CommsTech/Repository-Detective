package ui

import (
	"strings"

	"git.commsnet.org/commstech/bugbot/store"
)

// ScanFormView drives manual scan UI (modal and full-page form).
type ScanFormView struct {
	Repo              store.Repository
	Effective         store.EffectiveSettings
	ProfileMeta       store.EffectiveSettingsMeta
	Profiles          []string
	ScanEnabled       bool
	DefaultReportOnly bool
	IssueFilingOn     bool
	DefaultRef        string

	RemediationPREnabled  bool
	LLMSanityGateEnabled  bool
	BacklogControlEnabled bool
}

func (h *Handler) buildScanFormView(repo store.Repository, effective store.EffectiveSettings, meta store.EffectiveSettingsMeta) ScanFormView {
	issueFiling := store.ShouldCreateForgeIssues(effective)
	ref := strings.TrimSpace(repo.DefaultBranch)
	if ref == "" {
		ref = "main"
	}
	return ScanFormView{
		Repo:                  repo,
		Effective:             effective,
		ProfileMeta:           meta,
		Profiles:              store.AllowedScanProfiles,
		ScanEnabled:           h.ScanTriggerEnabled(),
		DefaultReportOnly:     !issueFiling,
		IssueFilingOn:         issueFiling,
		DefaultRef:            ref,
		RemediationPREnabled:  h.remediationPREnabled,
		LLMSanityGateEnabled:  h.platform.LLMSanityGateEnabled,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
	}
}
