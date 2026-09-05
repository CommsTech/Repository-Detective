package ui

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// PlatformSettingsApplier applies saved platform settings to the running process.
type PlatformSettingsApplier func(settings store.PlatformSettings) error

// SetPlatformSettingsApplier wires live apply of Configure form saves.
func (h *Handler) SetPlatformSettingsApplier(fn PlatformSettingsApplier) {
	if h != nil {
		h.applyPlatformSettings = fn
	}
}

// SetGlobal replaces the in-memory global settings snapshot used by the UI.
func (h *Handler) SetGlobal(global store.GlobalSettingsSnapshot) {
	if h != nil {
		h.global = global
	}
}

// ConfigureFormValues is the editable form state for the Configure page.
type ConfigureFormValues struct {
	ScanProfile string

	SchedulerEnabled          bool
	NotificationsEnabled      bool
	PreinstallAuditEnabled    bool
	RemediationPlannerEnabled bool
	RemediationPREnabled      bool
	EvidenceClosureEnabled    bool

	AutoCreateIssues  bool
	IssuePolicy       string
	RemediationPolicy string
	SeverityGate      string
	ConfidenceGate    float64
	AnalysisDepth     int
	ScheduleCron      string
	ScheduleEnabled   bool

	EnableTrivy             bool
	EnableGrype             bool
	EnableGitleaks          bool
	EnableSemgrep           bool
	EnableGovulncheck       bool
	EnableGosec             bool
	EnableStaticcheck       bool
	EnableHadolint          bool
	EnableCheckov           bool
	EnableLinters           bool
	EnablePerformanceChecks bool
	EnableCodeGraph         bool

	AIRecommendationsEnabled            bool
	AIRecommendationsMaxTokensPerScan   int
	AIRecommendationsTokenBudgetPerScan int
	AIRecommendationsMaxFindingsPerScan int
}

// SaveConfigure persists editable platform settings from the Configure form.
func (h *Handler) SaveConfigure(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}

	settings := platformSettingsFromForm(c)
	settings.ScanProfile = store.NormalizeScanProfile(settings.ScanProfile)
	settings.UpdatedBy = "ui"
	if err := store.ValidatePlatformSettings(settings); err != nil {
		h.renderConfigurePage(c, err.Error(), settings)
		return
	}
	if err := h.store.SavePlatformSettings(c.Request.Context(), settings); err != nil {
		h.renderConfigurePage(c, "failed to save settings: "+err.Error(), settings)
		return
	}
	if h.applyPlatformSettings != nil {
		if err := h.applyPlatformSettings(settings); err != nil {
			h.renderConfigurePage(c, "saved but failed to apply live: "+err.Error(), settings)
			return
		}
	}
	h.global = store.ApplyPlatformSettingsToGlobal(h.global, settings)

	q := url.Values{}
	q.Set("saved", "1")
	if key := c.Query("api_key"); key != "" {
		q.Set("api_key", key)
	}
	c.Redirect(http.StatusSeeOther, h.basePath+"/configure?"+q.Encode())
}

func (h *Handler) renderConfigurePage(c *gin.Context, errMsg string, draft store.PlatformSettings) {
	saved, _ := h.store.GetPlatformSettings(c.Request.Context())
	// Never call readinessFn here — it probes every scanner binary and makes Configure
	// time out in the browser so Save never reaches the server.
	readiness := h.configureReadiness(saved)
	form := buildConfigureForm(h.global, h.platform, readiness, saved, draft)
	// Keep status sections aligned with the editable form (live DB overrides).
	readiness.Features.SchedulerEnabled = form.SchedulerEnabled
	readiness.Features.NotificationsEnabled = form.NotificationsEnabled
	readiness.Features.PreinstallAuditEnabled = form.PreinstallAuditEnabled
	readiness.Features.RemediationPlannerEnabled = form.RemediationPlannerEnabled
	readiness.Features.RemediationPREnabled = form.RemediationPREnabled
	readiness.Features.EvidenceClosureEnabled = form.EvidenceClosureEnabled
	readiness.Features.ScanProfile = form.ScanProfile
	platform := h.platform
	platform.NotificationsEnabled = form.NotificationsEnabled
	platform.OpenClawAIReviewEnabled = form.AIRecommendationsEnabled

	caps := buildCapabilityStatuses(readiness, h.notifyGlobal, platform, h.basePath)
	sections := buildConfigureSections(readiness, platform, h.notifyGlobal, h.global, h.basePath)

	notice := "Change options below, then click Save settings at the top or bottom. Changes apply immediately to new scans. Secrets stay in .env only."
	savedOK := c.Query("saved") == "1" && errMsg == ""
	if savedOK {
		notice = "Settings saved and applied live. Checkboxes and section badges below now match what you saved. Features that need secrets (notification channels, forge token) stay degraded until those are set in .env. AI stays optional unless LLM auditors are enabled."
	}

	h.renderNav(c, "configure.html", "Configure", "settings", map[string]any{
		"Readiness":           readiness,
		"Capabilities":        caps,
		"Platform":            platform,
		"Sections":            sections,
		"SetupComplete":       h.isSetupComplete(c.Request.Context()),
		"Form":                form,
		"Profiles":            store.PrimaryScanProfileOptions,
		"ProfileDescriptions": store.ProfileDescriptions,
		"Error":               errMsg,
		"SavedAt":             saved.UpdatedAt,
		"Editable":            true,
		"NoticeText":          notice,
		"SavedOK":             savedOK,
	})
}

// configureReadiness builds feature flags from live handler state without tool probes.
func (h *Handler) configureReadiness(saved store.PlatformSettings) operator.Readiness {
	features := operator.FeatureFlags{
		DatabaseEnabled:           true,
		DatabaseHealthy:           h.store != nil,
		SchedulerEnabled:          false,
		NotificationsEnabled:      h.platform.NotificationsEnabled,
		PreinstallAuditEnabled:    h.preinstallEnabled,
		RemediationPlannerEnabled: h.remediationEnabled,
		RemediationPREnabled:      h.remediationPREnabled,
		EvidenceClosureEnabled:    h.closureEnabled,
		RunnerDelegationEnabled:   h.platform.RunnerDelegationEnabled,
		UIEnabled:                 true,
		ScanProfile:               h.global.ScanProfile,
		PublicURLConfigured:       strings.TrimSpace(h.platform.PublicURL) != "",
	}
	if saved.SchedulerEnabled != nil {
		features.SchedulerEnabled = *saved.SchedulerEnabled
	}
	if saved.NotificationsEnabled != nil {
		features.NotificationsEnabled = *saved.NotificationsEnabled
	}
	if saved.PreinstallAuditEnabled != nil {
		features.PreinstallAuditEnabled = *saved.PreinstallAuditEnabled
	}
	if saved.RemediationPlannerEnabled != nil {
		features.RemediationPlannerEnabled = *saved.RemediationPlannerEnabled
	}
	if saved.RemediationPREnabled != nil {
		features.RemediationPREnabled = *saved.RemediationPREnabled
	}
	if saved.EvidenceClosureEnabled != nil {
		features.EvidenceClosureEnabled = *saved.EvidenceClosureEnabled
	}
	if p := strings.TrimSpace(saved.ScanProfile); p != "" {
		features.ScanProfile = p
	}
	return operator.Readiness{
		ProductName: "Repository Detective",
		Status:      "healthy",
		Features:    features,
	}
}

func buildConfigureForm(
	global store.GlobalSettingsSnapshot,
	platform PlatformContext,
	readiness operator.Readiness,
	saved store.PlatformSettings,
	draft store.PlatformSettings,
) ConfigureFormValues {
	// Start from live effective values.
	f := ConfigureFormValues{
		ScanProfile:               store.NormalizeScanProfile(firstNonEmpty(saved.ScanProfile, global.ScanProfile, readiness.Features.ScanProfile)),
		SchedulerEnabled:          readiness.Features.SchedulerEnabled,
		NotificationsEnabled:      readiness.Features.NotificationsEnabled,
		PreinstallAuditEnabled:    readiness.Features.PreinstallAuditEnabled,
		RemediationPlannerEnabled: readiness.Features.RemediationPlannerEnabled,
		RemediationPREnabled:      readiness.Features.RemediationPREnabled,
		EvidenceClosureEnabled:    readiness.Features.EvidenceClosureEnabled,
		AutoCreateIssues:          store.ShouldCreateForgeIssues(store.EffectiveFromGlobalSnapshot(global)),
		IssuePolicy:               firstNonEmpty(global.IssuePolicy, store.IssuePolicyOff),
		RemediationPolicy:         firstNonEmpty(global.RemediationPolicy, "suggest"),
		SeverityGate:              firstNonEmpty(global.SeverityGate, "high"),
		ConfidenceGate:            global.ConfidenceGate,
		AnalysisDepth:             global.AnalysisDepth,
		ScheduleCron:              global.ScheduleCron,
		ScheduleEnabled:           global.ScheduleEnabled,
		EnableTrivy:               global.EnableTrivy,
		EnableGrype:               global.EnableGrype,
		EnableGitleaks:            global.EnableGitleaks,
		EnableSemgrep:             global.EnableSemgrep,
		EnableGovulncheck:         global.EnableGovulncheck,
		EnableGosec:               global.EnableGosec,
		EnableStaticcheck:         global.EnableStaticcheck,
		EnableHadolint:            global.EnableHadolint,
		EnableCheckov:             global.EnableCheckov,
		EnableLinters:             global.EnableLinters,
		EnablePerformanceChecks:   global.EnablePerformanceChecks,
		EnableCodeGraph:           global.EnableCodeGraph,
		AIRecommendationsEnabled:  platform.OpenClawAIReviewEnabled,
	}
	if f.ConfidenceGate == 0 {
		f.ConfidenceGate = 0.75
	}
	if f.AnalysisDepth == 0 {
		f.AnalysisDepth = 2
	}
	applySavedBools(&f, saved)
	applyDraft(&f, draft)
	_ = platform
	return f
}

func applySavedBools(f *ConfigureFormValues, s store.PlatformSettings) {
	if s.SchedulerEnabled != nil {
		f.SchedulerEnabled = *s.SchedulerEnabled
	}
	if s.NotificationsEnabled != nil {
		f.NotificationsEnabled = *s.NotificationsEnabled
	}
	if s.PreinstallAuditEnabled != nil {
		f.PreinstallAuditEnabled = *s.PreinstallAuditEnabled
	}
	if s.RemediationPlannerEnabled != nil {
		f.RemediationPlannerEnabled = *s.RemediationPlannerEnabled
	}
	if s.RemediationPREnabled != nil {
		f.RemediationPREnabled = *s.RemediationPREnabled
	}
	if s.EvidenceClosureEnabled != nil {
		f.EvidenceClosureEnabled = *s.EvidenceClosureEnabled
	}
	if s.AutoCreateIssues != nil {
		f.AutoCreateIssues = *s.AutoCreateIssues
	}
	if s.IssuePolicy != "" {
		f.IssuePolicy = s.IssuePolicy
	}
	if s.RemediationPolicy != "" {
		f.RemediationPolicy = s.RemediationPolicy
	}
	if s.SeverityGate != "" {
		f.SeverityGate = s.SeverityGate
	}
	if s.ConfidenceGate != nil {
		f.ConfidenceGate = *s.ConfidenceGate
	}
	if s.AnalysisDepth != nil {
		f.AnalysisDepth = *s.AnalysisDepth
	}
	if s.ScheduleCron != "" {
		f.ScheduleCron = s.ScheduleCron
	}
	if s.ScheduleEnabled != nil {
		f.ScheduleEnabled = *s.ScheduleEnabled
	}
	setBool(&f.EnableTrivy, s.EnableTrivy)
	setBool(&f.EnableGrype, s.EnableGrype)
	setBool(&f.EnableGitleaks, s.EnableGitleaks)
	setBool(&f.EnableSemgrep, s.EnableSemgrep)
	setBool(&f.EnableGovulncheck, s.EnableGovulncheck)
	setBool(&f.EnableGosec, s.EnableGosec)
	setBool(&f.EnableStaticcheck, s.EnableStaticcheck)
	setBool(&f.EnableHadolint, s.EnableHadolint)
	setBool(&f.EnableCheckov, s.EnableCheckov)
	setBool(&f.EnableLinters, s.EnableLinters)
	setBool(&f.EnablePerformanceChecks, s.EnablePerformanceChecks)
	setBool(&f.EnableCodeGraph, s.EnableCodeGraph)
	if s.AIRecommendationsEnabled != nil {
		f.AIRecommendationsEnabled = *s.AIRecommendationsEnabled
	}
	if s.AIRecommendationsMaxTokensPerScan != nil {
		f.AIRecommendationsMaxTokensPerScan = *s.AIRecommendationsMaxTokensPerScan
	}
	if s.AIRecommendationsTokenBudgetPerScan != nil {
		f.AIRecommendationsTokenBudgetPerScan = *s.AIRecommendationsTokenBudgetPerScan
	}
	if s.AIRecommendationsMaxFindingsPerScan != nil {
		f.AIRecommendationsMaxFindingsPerScan = *s.AIRecommendationsMaxFindingsPerScan
	}
	if s.ScanProfile != "" {
		f.ScanProfile = s.ScanProfile
	}
}

func applyDraft(f *ConfigureFormValues, d store.PlatformSettings) {
	// Draft only used after validation errors — reuse same merge.
	if d.ScanProfile == "" && d.IssuePolicy == "" && d.SeverityGate == "" &&
		d.SchedulerEnabled == nil && d.EnableTrivy == nil && d.AIRecommendationsEnabled == nil {
		return
	}
	applySavedBools(f, d)
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

func platformSettingsFromForm(c *gin.Context) store.PlatformSettings {
	confGate, _ := strconv.ParseFloat(strings.TrimSpace(c.PostForm("confidence_gate")), 64)
	depth, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("analysis_depth")))
	maxTok, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("ai_recommendations_max_tokens_per_scan")))
	tokBudget, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("ai_recommendations_token_budget_per_scan")))
	maxFindings, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("ai_recommendations_max_findings_per_scan")))

	return store.PlatformSettings{
		ScanProfile:                         strings.TrimSpace(c.PostForm("scan_profile")),
		SchedulerEnabled:                    formBoolPtr(c, "scheduler_enabled"),
		NotificationsEnabled:                formBoolPtr(c, "notifications_enabled"),
		PreinstallAuditEnabled:              formBoolPtr(c, "preinstall_audit_enabled"),
		RemediationPlannerEnabled:           formBoolPtr(c, "remediation_planner_enabled"),
		RemediationPREnabled:                formBoolPtr(c, "remediation_pr_enabled"),
		EvidenceClosureEnabled:              formBoolPtr(c, "evidence_closure_enabled"),
		AutoCreateIssues:                    formBoolPtr(c, "auto_create_issues"),
		IssuePolicy:                         strings.TrimSpace(c.PostForm("issue_policy")),
		RemediationPolicy:                   strings.TrimSpace(c.PostForm("remediation_policy")),
		SeverityGate:                        strings.TrimSpace(c.PostForm("severity_gate")),
		ConfidenceGate:                      &confGate,
		AnalysisDepth:                       &depth,
		ScheduleCron:                        strings.TrimSpace(c.PostForm("schedule_cron")),
		ScheduleEnabled:                     formBoolPtr(c, "schedule_enabled"),
		EnableTrivy:                         formBoolPtr(c, "enable_trivy"),
		EnableGrype:                         formBoolPtr(c, "enable_grype"),
		EnableGitleaks:                      formBoolPtr(c, "enable_gitleaks"),
		EnableSemgrep:                       formBoolPtr(c, "enable_semgrep"),
		EnableGovulncheck:                   formBoolPtr(c, "enable_govulncheck"),
		EnableGosec:                         formBoolPtr(c, "enable_gosec"),
		EnableStaticcheck:                   formBoolPtr(c, "enable_staticcheck"),
		EnableHadolint:                      formBoolPtr(c, "enable_hadolint"),
		EnableCheckov:                       formBoolPtr(c, "enable_checkov"),
		EnableLinters:                       formBoolPtr(c, "enable_linters"),
		EnablePerformanceChecks:             formBoolPtr(c, "enable_performance_checks"),
		EnableCodeGraph:                     formBoolPtr(c, "enable_code_graph"),
		AIRecommendationsEnabled:            formBoolPtr(c, "ai_recommendations_enabled"),
		AIRecommendationsMaxTokensPerScan:   &maxTok,
		AIRecommendationsTokenBudgetPerScan: &tokBudget,
		AIRecommendationsMaxFindingsPerScan: &maxFindings,
	}
}

func formBoolPtr(c *gin.Context, name string) *bool {
	v := strings.TrimSpace(c.PostForm(name))
	b := v == "true" || v == "1" || v == "on"
	return &b
}
