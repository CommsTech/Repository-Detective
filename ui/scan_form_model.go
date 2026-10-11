package ui

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// ScanFormView drives manual scan UI (modal and full-page form).
type ScanFormView struct {
	Repo              store.Repository
	Effective         store.EffectiveSettings
	ProfileMeta       store.EffectiveSettingsMeta
	Profiles          []store.ScanProfileOption
	ScanEnabled       bool
	DefaultReportOnly bool
	IssueFilingOn     bool
	DefaultRef        string
	FilingPolicy      store.ScanFilingPolicy
	ScanPolicyMode    string
	SeverityGate      string
	ConfidenceGate    float64
	MaxIssuesPerScan  int
	ScannerSummary    string
	EnableCodeGraph   bool
	NotificationsOn   bool
	RunnerDelegation  bool

	RemediationPREnabled  bool
	LLMSanityGateEnabled  bool
	BacklogControlEnabled bool
}

func (h *Handler) buildScanFormView(repo store.Repository, effective store.EffectiveSettings, meta store.EffectiveSettingsMeta) ScanFormView {
	filing := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:                  store.ScanKindManual,
		Effective:             effective,
		RequestDryRun:         false,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
		MaxIssuesPerScan:      h.platform.MaxIssuesPerScan,
	})
	ref := strings.TrimSpace(repo.DefaultBranch)
	if ref == "" {
		ref = "main"
	}
	return ScanFormView{
		Repo:                  repo,
		Effective:             effective,
		ProfileMeta:           meta,
		Profiles:              store.PrimaryScanProfileOptions,
		ScanEnabled:           h.ScanTriggerEnabled(),
		DefaultReportOnly:     filing.DryRunCheckboxDefault,
		IssueFilingOn:         filing.IssueFilingAllowed,
		DefaultRef:            ref,
		FilingPolicy:          filing,
		ScanPolicyMode:        h.platform.ScanPolicyMode,
		SeverityGate:          effective.SeverityGate,
		ConfidenceGate:        effective.ConfidenceGate,
		MaxIssuesPerScan:      h.platform.MaxIssuesPerScan,
		ScannerSummary:        store.ScannerSummaryLabel(effective),
		EnableCodeGraph:       effective.EnableCodeGraph,
		NotificationsOn:       h.platform.NotificationsEnabled,
		RunnerDelegation:      h.platform.RunnerDelegationEnabled,
		RemediationPREnabled:  h.remediationPREnabled,
		LLMSanityGateEnabled:  h.platform.LLMSanityGateEnabled,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
	}
}
