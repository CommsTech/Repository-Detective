package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler serves control-plane JSON API routes.
type Handler struct {
	store        store.QueryStore
	global       store.GlobalSettingsSnapshot
	notifyGlobal notify.Config
	logger       *logrus.Logger
	toolsProbe   func() []operator.ToolStatus
}

// NewHandler creates an API handler. store may be nil when database is disabled.
func NewHandler(s store.QueryStore, global store.GlobalSettingsSnapshot, logger *logrus.Logger) *Handler {
	return &Handler{store: s, global: global, logger: logger}
}

// SetToolsProbe attaches a live scanner PATH probe used to align dashboard metrics
// with System Health (current missing binaries, not historical rows).
func (h *Handler) SetToolsProbe(fn func() []operator.ToolStatus) {
	if h != nil {
		h.toolsProbe = fn
	}
}

// SetGlobal replaces the in-memory global settings snapshot used by API handlers.
func (h *Handler) SetGlobal(global store.GlobalSettingsSnapshot) {
	if h != nil {
		h.global = global
	}
}

// SetNotificationGlobal attaches redacted global notification config for settings responses.
func (h *Handler) SetNotificationGlobal(cfg notify.Config) {
	if h != nil {
		h.notifyGlobal = cfg
	}
}

// RegisterRoutes mounts control-plane routes on the given group (already authenticated).
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/dashboard/summary", h.DashboardSummary)

	g.GET("/repos", h.ListRepositories)
	g.GET("/repos/:id", h.GetRepository)
	g.GET("/repos/:id/settings", h.GetRepoSettings)
	g.PUT("/repos/:id/settings", h.UpdateRepoSettings)
	h.registerRepoScanControlRoutes(g)
	g.GET("/repos/:id/scans", h.ListRepoScans)
	g.GET("/repos/:id/findings", h.ListRepoFindings)
	g.GET("/repos/:id/graph", h.GetRepoGraph)
	g.GET("/repos/:id/graph/export", h.ExportRepoGraph)
	g.GET("/repos/:id/reconciliation", h.GetRepoReconciliation)

	g.GET("/scans/:scan_id", h.GetScan)
	g.GET("/scans/:scan_id/scanner-results", h.GetScanScannerResults)
	g.GET("/scans/:scan_id/graph", h.GetScanGraph)
	g.GET("/scans/:scan_id/graph/export", h.ExportScanGraph)

	g.GET("/findings", h.ListFindings)
	g.GET("/findings/export", h.ExportFindings)
	g.GET("/findings/:id", h.GetFinding)
	g.GET("/findings/:id/lifecycle", h.GetFindingLifecycle)
}

func (h *Handler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database disabled",
			"message": "Enable database_enabled to use the control plane API",
		})
		return false
	}
	return true
}

func parseRepoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return 0, false
	}
	return id, true
}

func parseFindingID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid finding id"})
		return 0, false
	}
	return id, true
}

func listOptions(c *gin.Context) store.ListOptions {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	return store.NormalizeListOptions(store.ListOptions{Limit: limit, Offset: offset})
}

func (h *Handler) DashboardSummary(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	summary, err := h.store.DashboardSummary(c.Request.Context(), limit)
	if err != nil {
		h.logger.Errorf("dashboard summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard"})
		return
	}
	if h.toolsProbe != nil {
		store.ApplyPlatformReadiness(&summary, h.toolsProbe())
	}
	c.JSON(http.StatusOK, toDashboardSummaryResponse(summary))
}

func (h *Handler) ListRepositories(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	repos, err := h.store.ListRepositoriesWithSummary(c.Request.Context(), listOptions(c))
	if err != nil {
		h.logger.Errorf("list repos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list repositories"})
		return
	}
	out := make([]repositorySummaryResponse, 0, len(repos))
	for _, repo := range repos {
		out = append(out, toRepositorySummaryResponse(repo))
	}
	c.JSON(http.StatusOK, gin.H{"repositories": out})
}

func (h *Handler) GetRepository(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}
	c.JSON(http.StatusOK, toRepositoryResponse(repo))
}

func (h *Handler) GetRepoSettings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	if _, err := h.store.GetRepository(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}
	settings, err := h.store.GetRepoSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, toSettingsResponse(id, settings, h.global, h.notifyGlobal))
}

func (h *Handler) UpdateRepoSettings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	if _, err := h.store.GetRepository(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	var update store.SettingsUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if err := store.ValidateSettingsUpdate(update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.store.GetRepoSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	existing.RepositoryID = id
	existing.UpdatedAt = time.Now().UTC()
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, update)
	if err := store.ValidateRepoSettings(merged); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SaveRepoSettings(c.Request.Context(), merged); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
		return
	}
	c.JSON(http.StatusOK, toSettingsResponse(id, merged, h.global, h.notifyGlobal))
}

func (h *Handler) ListRepoScans(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	scans, err := h.store.ListScansByRepository(c.Request.Context(), id, listOptions(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scans"})
		return
	}
	out := make([]scanResponse, 0, len(scans))
	for _, scan := range scans {
		out = append(out, toScanResponse(scan))
	}
	c.JSON(http.StatusOK, gin.H{"scans": out})
}

func (h *Handler) ListRepoFindings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	filter := findingFilterFromQuery(c)
	filter.RepositoryID = id
	h.respondFindings(c, filter)
}

func (h *Handler) ListFindings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	h.respondFindings(c, findingFilterFromQuery(c))
}

func (h *Handler) ExportFindings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	filter := findingFilterFromQuery(c)
	if filter.Limit <= 0 || filter.Limit > 5000 {
		filter.Limit = 5000
	}
	findings, err := h.store.ListFindings(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list findings"})
		return
	}
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	stamp := time.Now().UTC().Format("20060102T150405Z")
	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="findings-`+stamp+`.csv"`)
		var b strings.Builder
		b.WriteString("id,repository_id,severity,category,status,source,rule_id,fingerprint,title,file_path,line\n")
		for _, f := range findings {
			b.WriteString(strconv.FormatInt(f.ID, 10))
			b.WriteByte(',')
			b.WriteString(strconv.FormatInt(f.RepositoryID, 10))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Severity))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Category))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Status))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Source))
			b.WriteByte(',')
			b.WriteString(csvCell(f.RuleID))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Fingerprint))
			b.WriteByte(',')
			b.WriteString(csvCell(f.Title))
			b.WriteByte(',')
			b.WriteString(csvCell(f.FilePath))
			b.WriteByte(',')
			b.WriteString(strconv.Itoa(f.Line))
			b.WriteByte('\n')
		}
		c.String(http.StatusOK, b.String())
		return
	}
	out := make([]findingListResponse, 0, len(findings))
	for _, f := range findings {
		out = append(out, toFindingListResponse(f))
	}
	c.Header("Content-Disposition", `attachment; filename="findings-`+stamp+`.json"`)
	c.JSON(http.StatusOK, gin.H{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"count":        len(out),
		"findings":     out,
	})
}

func csvCell(v string) string {
	v = strings.ReplaceAll(v, `"`, `""`)
	if strings.ContainsAny(v, ",\"\n\r") {
		return `"` + v + `"`
	}
	return v
}

func findingFilterFromQuery(c *gin.Context) store.FindingFilter {
	opts := listOptions(c)
	repoID, _ := strconv.ParseInt(c.Query("repo_id"), 10, 64)
	return store.FindingFilter{
		RepositoryID:      repoID,
		Severity:          c.Query("severity"),
		Category:          c.Query("category"),
		Status:            c.Query("status"),
		Source:            c.Query("source"),
		IncludeSuppressed: c.Query("show_suppressed") == "1" || c.Query("include_suppressed") == "true",
		OnlySuppressed:    c.Query("only_suppressed") == "1",
		Limit:             opts.Limit,
		Offset:            opts.Offset,
	}
}

func (h *Handler) respondFindings(c *gin.Context, filter store.FindingFilter) {
	findings, err := h.store.ListFindings(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list findings"})
		return
	}
	out := make([]findingListResponse, 0, len(findings))
	for _, f := range findings {
		out = append(out, toFindingListResponse(f))
	}
	c.JSON(http.StatusOK, gin.H{"findings": out})
}

func (h *Handler) GetScan(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	scan, err := h.store.GetScan(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scan not found"})
		return
	}
	c.JSON(http.StatusOK, toScanResponse(scan))
}

func (h *Handler) GetScanScannerResults(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	results, err := h.store.ListScannerResultsByScan(c.Request.Context(), c.Param("scan_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scanner results"})
		return
	}
	out := make([]scannerResultResponse, 0, len(results))
	for _, r := range results {
		out = append(out, toScannerResultResponse(r))
	}
	c.JSON(http.StatusOK, gin.H{"scanner_results": out})
}

func (h *Handler) GetFinding(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	detail, err := h.store.GetFindingDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "finding not found"})
		return
	}
	c.JSON(http.StatusOK, toFindingDetailResponse(detail))
}

func (h *Handler) GetFindingLifecycle(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	events, err := h.store.ListLifecycleEventsByFinding(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list lifecycle events"})
		return
	}
	out := make([]lifecycleEventResponse, 0, len(events))
	for _, ev := range events {
		out = append(out, toLifecycleEventResponse(ev))
	}
	c.JSON(http.StatusOK, gin.H{"lifecycle_events": out})
}

func (h *Handler) GetRepoReconciliation(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, _ := store.ResolveEffectiveSettingsFull(h.global, settings)
	issueFiling := store.ShouldCreateForgeIssues(effective)
	scanID := strings.TrimSpace(c.Query("scan_id"))
	var (
		sum store.ReconciliationSummary
		err error
	)
	if scanID != "" {
		sum, err = h.store.ReconciliationSummaryForScan(c.Request.Context(), id, scanID, issueFiling)
	} else {
		sum, err = h.store.ReconciliationSummaryForRepository(c.Request.Context(), id, issueFiling)
	}
	if err != nil {
		h.logger.Errorf("reconciliation summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load reconciliation summary"})
		return
	}
	c.JSON(http.StatusOK, sum)
}

// GlobalSnapshotFromConfig builds the global settings snapshot from main config values.
// Profile defaults are applied per-repo during ResolveEffectiveSettings when scan_profile is set globally.
func GlobalSnapshotFromConfig(cfg GlobalConfigInput) store.GlobalSettingsSnapshot {
	snap := legacyGlobalSnapshot(cfg)
	if cfg.ScanProfile != "" {
		snap.ScanProfile = store.NormalizeScanProfile(cfg.ScanProfile)
	} else {
		snap.ScanProfile = store.ScanProfileCustom
	}
	return snap
}

func legacyGlobalSnapshot(cfg GlobalConfigInput) store.GlobalSettingsSnapshot {
	return store.GlobalSettingsSnapshot{
		ScanProfile:                 cfg.ScanProfile,
		Enabled:                     true,
		PolicyLevel:                 "issue_only",
		WorkspaceMode:               cfg.WorkspaceMode,
		AnalysisDepth:               cfg.AnalysisDepth,
		EnableLLMAuditors:           cfg.EnableLLMAuditors,
		EnableTrivy:                 cfg.EnableTrivy,
		EnableGrype:                 cfg.EnableGrype,
		EnableGitleaks:              cfg.EnableGitleaks,
		EnableSemgrep:               cfg.EnableSemgrep,
		EnableGovulncheck:           cfg.EnableGovulncheck,
		EnableGosec:                 cfg.EnableGosec,
		EnableStaticcheck:           cfg.EnableStaticcheck,
		EnableHadolint:              cfg.EnableHadolint,
		EnableCheckov:               cfg.EnableCheckov,
		EnableLinters:               cfg.EnableLinters,
		SeverityGate:                cfg.GiteaStatusFailOn,
		ConfidenceGate:              cfg.MinIssueConfidence,
		IssuePolicy:                 issuePolicyFromAutoCreate(cfg.AutoCreateIssues),
		RemediationPolicy:           "off",
		RunnerPolicy:                "core",
		ScheduleEnabled:             false,
		AIPolicy:                    aiPolicyFromLLM(cfg.EnableLLMAuditors),
		EnableHealthChecks:          cfg.EnableHealthChecks,
		EnableTechDebtChecks:        cfg.EnableTechDebtChecks,
		EnableReliabilityChecks:     cfg.EnableReliabilityChecks,
		EnableMaintainabilityChecks: cfg.EnableMaintainabilityChecks,
		EnableTestGapChecks:         cfg.EnableTestGapChecks,
		EnablePerformanceChecks:     cfg.EnablePerformanceChecks,
		EnableAIRiskChecks:          cfg.EnableAIRiskChecks,
		HealthMaxFindings:           cfg.HealthMaxFindings,
		HealthLargeFileLines:        cfg.HealthLargeFileLines,
		HealthLargeFunctionLines:    cfg.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       cfg.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     cfg.HealthMaxFunctionParams,
		EnableCodeGraph:             cfg.EnableCodeGraph,
		GraphMaxNodes:               cfg.GraphMaxNodes,
		GraphMaxEdges:               cfg.GraphMaxEdges,
		GraphTimeoutSeconds:         cfg.GraphTimeoutSeconds,
		GraphIncludeFunctions:       cfg.GraphIncludeFunctions,
		GraphIncludeFindings:        cfg.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   cfg.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         cfg.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   cfg.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        cfg.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      cfg.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       cfg.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       cfg.IACScannerMaxFindings,
	}
}

// GlobalConfigInput avoids importing main package.
type GlobalConfigInput struct {
	ScanProfile                 string
	WorkspaceMode               string
	AnalysisDepth               int
	EnableLLMAuditors           bool
	EnableTrivy                 bool
	EnableGrype                 bool
	EnableGitleaks              bool
	EnableSemgrep               bool
	EnableGovulncheck           bool
	EnableGosec                 bool
	EnableStaticcheck           bool
	EnableHadolint              bool
	EnableCheckov               bool
	EnableLinters               bool
	GiteaStatusFailOn           string
	MinIssueConfidence          float64
	AutoCreateIssues            bool
	EnableHealthChecks          bool
	EnableTechDebtChecks        bool
	EnableReliabilityChecks     bool
	EnableMaintainabilityChecks bool
	EnableTestGapChecks         bool
	EnablePerformanceChecks     bool
	EnableAIRiskChecks          bool
	HealthMaxFindings           int
	HealthLargeFileLines        int
	HealthLargeFunctionLines    int
	HealthMaxNestingDepth       int
	HealthMaxFunctionParams     int
	EnableCodeGraph             bool
	GraphMaxNodes               int
	GraphMaxEdges               int
	GraphTimeoutSeconds         int
	GraphIncludeFunctions       bool
	GraphIncludeFindings        bool
	GovulncheckTimeoutSeconds   int
	GosecTimeoutSeconds         int
	StaticcheckTimeoutSeconds   int
	GoScannerMaxFindings        int
	HadolintTimeoutSeconds      int
	CheckovTimeoutSeconds       int
	IACScannerMaxFindings       int
}

func issuePolicyFromAutoCreate(auto bool) string {
	if auto {
		return "all"
	}
	return "off"
}

func aiPolicyFromLLM(enabled bool) string {
	if enabled {
		return "allowed"
	}
	return "disabled"
}
