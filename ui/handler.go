package ui

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/closure"
	"git.commsnet.org/commstech/bugbot/internal/security"
	"git.commsnet.org/commstech/bugbot/notify"
	"git.commsnet.org/commstech/bugbot/operator"
	"git.commsnet.org/commstech/bugbot/patcher"
	"git.commsnet.org/commstech/bugbot/preinstall"
	"git.commsnet.org/commstech/bugbot/remediation"
	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const settingsNotice = "Per-repo settings are enforced on scans (Phase 8). Runner policy is enforced for scheduled and manual full scans when runner delegation is enabled (Phase 12)."

// Handler serves server-rendered operator UI pages.
type Handler struct {
	store                store.QueryStore
	global               store.GlobalSettingsSnapshot
	notifyGlobal         notify.Config
	basePath             string
	logger               *logrus.Logger
	tmpl                 *template.Template
	preinstallRunner     *preinstall.Runner
	preinstallEnabled    bool
	apiKeySecret         string
	auth                 AuthConfig
	remediationEnabled   bool
	remediation          RemediationBackend
	remediationPREnabled bool
	remediationPR        RemediationPRBackend
	closureEnabled       bool
	closure              ClosureBackend
	suppressionEnabled   bool
	suppression          SuppressionBackend
	reconcileEnabled     bool
	reconciler           IssueReconciler
	scanTrigger          ScanTrigger
	readinessFn          func() operator.Readiness
	platform             PlatformContext
}

// IssueReconciler previews and applies existing issue reconciliation.
type IssueReconciler interface {
	Preview(ctx context.Context, repositoryID int64) (any, error)
	Apply(ctx context.Context, repositoryID int64) (any, error)
}

// SuppressionBackend applies calibration actions from the UI.
type SuppressionBackend interface {
	SuppressFinding(ctx context.Context, findingID int64, reason, createdBy string) error
	SuppressRuleForRepo(ctx context.Context, findingID int64, reason, createdBy string) error
	MarkIntentionalStandalone(ctx context.Context, findingID int64, reason, createdBy string) error
	MarkFalsePositive(ctx context.Context, findingID int64, reason, createdBy string) error
}

// RemediationBackend generates and updates remediation plans from the UI.
type RemediationBackend interface {
	GeneratePlan(ctx context.Context, findingID int64) (remediation.Plan, error)
	ApprovePlan(ctx context.Context, planID string) error
	RejectPlan(ctx context.Context, planID string) error
}

// RemediationPRBackend creates safe remediation pull requests from the UI.
type RemediationPRBackend interface {
	CheckPREligibility(ctx context.Context, planID string) (patcher.EligibilityResult, error)
	AttemptPR(ctx context.Context, planID string) (patcher.PatchAttempt, error)
	ListPatchAttempts(ctx context.Context, planID string) ([]patcher.PatchAttempt, error)
}

// ClosureBackend tracks evidence-based closure from the UI.
type ClosureBackend interface {
	GetClosureEvidence(ctx context.Context, findingID int64) (closure.Evidence, error)
	VerifyClosure(ctx context.Context, findingID int64) (closure.Evidence, error)
	CheckPatchAttemptMerge(ctx context.Context, attemptID string) (closure.Evidence, error)
}

// NewHandler creates a UI handler.
func NewHandler(s store.QueryStore, global store.GlobalSettingsSnapshot, basePath string, logger *logrus.Logger, preinstallRunner *preinstall.Runner, preinstallEnabled bool, apiKeySecret string) (*Handler, error) {
	basePath = normalizeBasePath(basePath)
	funcs := templateFuncs()
	tmpl, err := template.New("layout").Funcs(funcs).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Handler{
		store: s, global: global, basePath: basePath, logger: logger, tmpl: tmpl,
		preinstallRunner: preinstallRunner, preinstallEnabled: preinstallEnabled, apiKeySecret: apiKeySecret,
	}, nil
}

// SetAuthConfig wires local session auth for the operator UI.
func (h *Handler) SetAuthConfig(cfg AuthConfig) {
	if h != nil {
		h.auth = cfg
	}
}

// SetNotificationGlobal attaches redacted global notification config for settings pages.
func (h *Handler) SetNotificationGlobal(cfg notify.Config) {
	if h != nil {
		h.notifyGlobal = cfg
	}
}

// SetRemediationBackend wires remediation planning actions for the UI.
func (h *Handler) SetRemediationBackend(enabled bool, backend RemediationBackend) {
	if h != nil {
		h.remediationEnabled = enabled
		h.remediation = backend
	}
}

// SetRemediationPRBackend wires safe remediation PR actions for the UI.
func (h *Handler) SetRemediationPRBackend(enabled bool, backend RemediationPRBackend) {
	if h != nil {
		h.remediationPREnabled = enabled
		h.remediationPR = backend
	}
}

// SetSuppressionBackend wires false-positive and suppression actions for the UI.
func (h *Handler) SetSuppressionBackend(enabled bool, backend SuppressionBackend) {
	if h != nil {
		h.suppressionEnabled = enabled
		h.suppression = backend
	}
}

// SetIssueReconciler wires issue reconciliation for the UI.
func (h *Handler) SetIssueReconciler(enabled bool, r IssueReconciler) {
	if h != nil {
		h.reconcileEnabled = enabled
		h.reconciler = r
	}
}

// SetClosureBackend wires evidence-based closure for the UI.
func (h *Handler) SetClosureBackend(enabled bool, backend ClosureBackend) {
	if h != nil {
		h.closureEnabled = enabled
		h.closure = backend
	}
}

// SetReadinessFn supplies operator readiness (tools + feature flags) for dashboard display.
func (h *Handler) SetReadinessFn(fn func() operator.Readiness) {
	if h != nil {
		h.readinessFn = fn
	}
}

// SetPlatformContext wires non-secret platform state for setup detection and health capability cards.
func (h *Handler) SetPlatformContext(ctx PlatformContext) {
	if h != nil {
		h.platform = ctx
	}
}

func normalizeBasePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/ui"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimSuffix(path, "/")
}

// RegisterPublicRoutes mounts static assets without API-key auth (CSS/JS must load in the browser).
func (h *Handler) RegisterPublicRoutes(g *gin.RouterGroup) {
	if sub, err := fs.Sub(staticFS, "static"); err == nil {
		g.StaticFS("/static", http.FS(sub))
	}
	h.RegisterUnlockRoutes(g)
}

// RegisterRoutes mounts protected UI routes (caller applies auth middleware).
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("", h.Dashboard)
	g.GET("/", h.Dashboard)
	g.GET("/repos", h.Repositories)
	g.GET("/repos/:id", h.RepoDetail)
	g.GET("/repos/:id/containers", h.ContainerImages)
	g.GET("/repos/:id/settings", h.RepoSettings)
	g.POST("/repos/:id/settings", h.SaveRepoSettings)
	g.GET("/repos/:id/graph", h.RepoGraph)
	g.GET("/repos/:id/report", h.RepoReport)
	g.GET("/repos/:id/reconcile", h.RepoReconcilePreview)
	g.POST("/repos/:id/reconcile", h.RepoReconcileApply)
	g.GET("/scans", h.Scans)
	g.GET("/reports", h.Reports)
	g.GET("/health", h.SystemHealth)
	g.GET("/scans/:scan_id", h.ScanDetail)
	g.GET("/scans/:scan_id/graph", h.ScanGraph)
	h.registerGraphRoutes(g)
	g.GET("/findings", h.Findings)
	g.GET("/findings/:id", h.FindingDetail)
	g.POST("/findings/:id/remediation/generate", h.GenerateFindingRemediation)
	g.POST("/findings/:id/remediation/approve", h.ApproveFindingRemediation)
	g.POST("/findings/:id/remediation/reject", h.RejectFindingRemediation)
	g.POST("/findings/:id/remediation/attempt-pr", h.AttemptFindingRemediationPR)
	g.POST("/findings/:id/closure/verify", h.VerifyFindingClosure)
	g.POST("/findings/:id/closure/check-merge", h.CheckFindingClosureMerge)
	g.POST("/findings/:id/suppress", h.SuppressFinding)
	g.POST("/findings/:id/suppress-rule", h.SuppressGraphRule)
	g.POST("/findings/:id/mark-intentional", h.MarkIntentionalStandalone)
	g.POST("/findings/:id/mark-false-positive", h.MarkFindingFalsePositive)
	g.GET("/configure", h.Configure)
	g.GET("/preinstall", h.Preinstall)
	g.POST("/preinstall", h.StartPreinstallAudit)
	g.GET("/preinstall/audits/:audit_id", h.PreinstallAuditDetail)
	g.POST("/preinstall/reports/:report_id/reviewed", h.MarkPreinstallReportReviewed)
	g.GET("/projects", h.ProjectGroups)
	g.POST("/projects", h.CreateProjectGroup)
	g.GET("/learning", h.Learning)
	h.registerScanRoutes(g)
	h.registerRepoControlRoutes(g)
}

// BasePath returns the configured UI mount path.
func (h *Handler) BasePath() string {
	return h.basePath
}

func (h *Handler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.Status(http.StatusServiceUnavailable)
		h.render(c, "error.html", "Database disabled", map[string]any{
			"Message": "Enable database_enabled to use the operator UI.",
		})
		return false
	}
	return true
}

type pageData struct {
	Title         string
	BasePath      string
	APIKey        string
	CSRFToken     string
	Notice        string
	NavSection    string
	AuthLocal     bool
	SetupComplete bool
	CurrentUser   *store.User
	Data          map[string]any
}

func clientAPIKeyFromRequest(c *gin.Context) string {
	if key := c.GetHeader("X-Repository-Detective-API-Key"); key != "" {
		return key
	}
	if key := c.GetHeader("X-Bugbot-API-Key"); key != "" {
		return key
	}
	if key := apiKeyFromCookie(c); key != "" {
		return key
	}
	return c.Query("api_key")
}

// displayAPIKey returns the key for template embedding. Cookie-based sessions omit the key from HTML.
func displayAPIKey(c *gin.Context) string {
	key := clientAPIKeyFromRequest(c)
	cookieKey := apiKeyFromCookie(c)
	if cookieKey != "" && key == cookieKey {
		return ""
	}
	return key
}

func csrfClientKey(c *gin.Context) string {
	key := clientAPIKeyFromRequest(c)
	if key != "" {
		return key
	}
	if cookieKey := apiKeyFromCookie(c); cookieKey != "" {
		return "cookie"
	}
	return ""
}

func (h *Handler) page(c *gin.Context, title string, data map[string]any) pageData {
	if data == nil {
		data = map[string]any{}
	}
	apiKey := displayAPIKey(c)
	var csrf string
	var currentUser *store.User
	if h.auth.IsLocal() {
		apiKey = ""
		sessionID := c.GetString(ctxAuthSessionID)
		userID := c.GetInt64(ctxAuthUserID)
		if h.auth.CSRFEnabled {
			csrf = security.SessionCSRFToken(h.auth.SessionSecret, sessionID, userID)
		}
		if u, ok := c.Get(ctxAuthUser); ok {
			if user, ok := u.(store.User); ok {
				currentUser = &user
			}
		}
	} else if h.auth.CSRFEnabled || h.apiKeySecret != "" {
		csrf = security.CSRFToken(h.apiKeySecret, csrfClientKey(c))
	}
	return pageData{
		Title:         title,
		BasePath:      h.basePath,
		APIKey:        apiKey,
		CSRFToken:     csrf,
		Notice:        settingsNotice,
		AuthLocal:     h.auth.IsLocal(),
		SetupComplete: h.isSetupComplete(c.Request.Context()),
		CurrentUser:   currentUser,
		Data:          data,
	}
}

func (h *Handler) isSetupComplete(ctx context.Context) bool {
	if h.auth.IsLocal() {
		return h.store != nil
	}
	if !h.platform.APIKeyConfigured || !h.platform.GiteaURLConfigured || !h.platform.GiteaTokenConfigured {
		return false
	}
	if h.store == nil {
		return false
	}
	repos, err := h.store.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 1})
	return err == nil && len(repos) > 0
}

func (h *Handler) requireCSRF(c *gin.Context) bool {
	if h.auth.IsLocal() {
		if !h.auth.CSRFEnabled {
			return true
		}
		return h.requireAuthCSRF(c, c.GetInt64(ctxAuthUserID), c.GetString(ctxAuthSessionID))
	}
	if !h.auth.CSRFEnabled && h.apiKeySecret == "" {
		return true
	}
	token := c.PostForm("csrf_token")
	if !security.ValidCSRFToken(h.apiKeySecret, csrfClientKey(c), token) {
		c.String(http.StatusForbidden, "invalid or missing CSRF token")
		return false
	}
	return true
}

func (h *Handler) render(c *gin.Context, name string, title string, data map[string]any) {
	h.renderNav(c, name, title, "", data)
}

func (h *Handler) renderNav(c *gin.Context, name, title, navSection string, data map[string]any) {
	if c.Writer.Status() == 0 {
		c.Status(http.StatusOK)
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pd := h.page(c, title, data)
	pd.NavSection = navSection
	if err := h.tmpl.ExecuteTemplate(c.Writer, name, pd); err != nil {
		h.logger.Errorf("render %s: %v", name, err)
		c.String(http.StatusInternalServerError, "template error")
	}
}

func (h *Handler) Dashboard(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	summary, err := h.store.DashboardSummary(c.Request.Context(), 10)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	var readiness operator.Readiness
	if h.readinessFn != nil {
		readiness = h.readinessFn()
		store.ApplyPlatformReadiness(&summary, readiness.Tools)
	}
	summary.RemediationInsight = store.BuildRemediationInsight(
		summary.OpenFindingsCount,
		summary.Remediation.Candidates,
		readiness.Features.RemediationPlannerEnabled,
		readiness.Features.RemediationPREnabled,
		h.global.RemediationPolicy,
	)

	activeScans, _ := h.store.CountActiveScans(c.Request.Context())
	summary.ScanHealth.ActiveScans = activeScans

	var missingScanners []store.ScannerPlatformRollup
	for _, r := range summary.Platform.Rollups {
		if r.Configured && !r.Available {
			missingScanners = append(missingScanners, r)
		}
	}
	actions := store.BuildDashboardActions(
		summary.Backlog.CriticalOpen,
		summary.Backlog.HighOpen,
		summary.FailedScansCount,
		summary.ScanHealth.RecentFailedScans,
		missingScanners,
		summary.ScanHealth.ReposNeedingAttention,
	)

	repos, _ := h.store.ListRepositoriesWithSummary(c.Request.Context(), store.ListOptions{Limit: 200})
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].OpenFindingsCount > repos[j].OpenFindingsCount
	})
	topRisk := repos
	if len(topRisk) > 5 {
		topRisk = topRisk[:5]
	}

	critical, _ := h.store.ListFindings(c.Request.Context(), store.FindingFilter{Severity: "critical", Status: "open", Limit: 10})
	high, _ := h.store.ListFindings(c.Request.Context(), store.FindingFilter{Severity: "high", Status: "open", Limit: 10})
	severe := append(critical, high...)

	calibration, _ := h.store.CalibrationSummary(c.Request.Context())
	learningHealth, _ := h.store.LearningHealthSummary(c.Request.Context())

	data := map[string]any{
		"Summary":              summary,
		"Readiness":            readiness,
		"ActiveScans":          activeScans,
		"TopRiskyRepos":        topRisk,
		"AllRepos":             repos,
		"RecentSevereFindings": severe,
		"Actions":              actions,
		"Calibration":          calibration,
		"LearningHealth":       learningHealth,
		"ChartJSON":            buildDashboardChartJSONWithStore(c.Request.Context(), h.store, summary, repos),
	}
	h.renderNav(c, "dashboard.html", "Dashboard", "dashboard", data)
}

func (h *Handler) Scans(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scans, err := h.store.ListRecentScans(c.Request.Context(), store.ListOptions{Limit: 100})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list scans")
		return
	}
	statusFilter := strings.TrimSpace(c.Query("status"))
	if statusFilter != "" {
		filtered := scans[:0]
		for _, s := range scans {
			if strings.EqualFold(s.Status, statusFilter) {
				filtered = append(filtered, s)
			}
		}
		scans = filtered
	}
	active, _ := h.store.CountActiveScans(c.Request.Context())
	h.renderNav(c, "scans.html", "Scans", "scans", map[string]any{
		"Scans": scans, "StatusFilter": statusFilter, "ActiveScans": active,
	})
}

func (h *Handler) Reports(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	summary, err := h.store.DashboardSummary(c.Request.Context(), 20)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load report data")
		return
	}
	repos, _ := h.store.ListRepositoriesWithSummary(c.Request.Context(), store.ListOptions{Limit: 200})
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].OpenFindingsCount > repos[j].OpenFindingsCount
	})
	h.renderNav(c, "reports.html", "Reports", "reports", map[string]any{
		"Summary": summary, "Repositories": repos,
		"Executive": buildFleetExecutiveSummary(summary),
	})
}

func (h *Handler) RepoReport(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	const findingsPageSize = 200
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}
	findingsFilter := store.FindingFilter{
		RepositoryID: id,
		Severity:     c.Query("severity"),
		Category:     c.Query("category"),
		Status:       c.Query("status"),
		Source:       c.Query("source"),
		Limit:        findingsPageSize,
		Offset:       offset,
	}
	scans, _ := h.store.ListScansByRepository(c.Request.Context(), id, store.ListOptions{Limit: 10})
	findings, _ := h.store.ListFindings(c.Request.Context(), findingsFilter)
	findingsTotal, _ := h.store.CountFindings(c.Request.Context(), findingsFilter)
	sortFindingsBySeverity(findings)
	external, _ := h.store.ListExternalIssuesByRepository(c.Request.Context(), id, store.ListOptions{Limit: 100})
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, meta := store.ResolveEffectiveSettingsFull(h.global, settings)
	severityCounts, _ := h.store.OpenFindingsBySeverityForRepository(c.Request.Context(), id)
	if severityCounts == nil {
		severityCounts = map[string]int{}
	}
	categoryCounts, _ := h.store.OpenFindingsByCategoryForRepository(c.Request.Context(), id)
	if categoryCounts == nil {
		categoryCounts = map[string]int{}
	}
	confidenceBands, _ := h.store.OpenFindingsConfidenceBandsForRepository(c.Request.Context(), id, effective.ConfidenceGate)
	if confidenceBands == nil {
		confidenceBands = map[string]int{}
	}
	topFindings, _ := h.store.ListFindings(c.Request.Context(), store.FindingFilter{
		RepositoryID: id, Status: "open", Limit: 20,
	})
	sortFindingsBySeverity(topFindings)
	executive := buildRepoExecutiveSummary(repo, severityCounts, categoryCounts, confidenceBands, topFindings, scans, effective, meta.ScanProfile)
	h.renderNav(c, "repo_report.html", "Report — "+repo.FullName, "reports", map[string]any{
		"Repo": repo, "Scans": scans, "Findings": findings, "ExternalIssues": external,
		"Effective": effective, "ProfileMeta": meta, "SeverityCounts": severityCounts,
		"CategoryCounts": categoryCounts, "ConfidenceBands": confidenceBands,
		"Executive": executive, "ChartJSON": buildRepoReportChartJSON(severityCounts, categoryCounts),
		"FindingsFilter": findingsFilter, "FindingsTotal": findingsTotal, "FindingsPageSize": findingsPageSize,
	})
}

func sortFindingsBySeverity(findings []store.FindingListItem) {
	order := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
		"info":     4,
	}
	sort.Slice(findings, func(i, j int) bool {
		si := order[strings.ToLower(findings[i].Severity)]
		sj := order[strings.ToLower(findings[j].Severity)]
		if si != sj {
			return si < sj
		}
		return findings[i].LastSeenAt.After(findings[j].LastSeenAt)
	})
}

func (h *Handler) SystemHealth(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	summary, err := h.store.DashboardSummary(c.Request.Context(), 5)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load health data")
		return
	}
	data := map[string]any{"Summary": summary}
	if h.readinessFn != nil {
		data["Readiness"] = h.readinessFn()
	}
	active, _ := h.store.CountActiveScans(c.Request.Context())
	data["ActiveScans"] = active
	runnerJobs := summary.RunnerJobsByStatus
	delegationEnabled := false
	if r, ok := data["Readiness"].(operator.Readiness); ok {
		delegationEnabled = r.Features.RunnerDelegationEnabled
	}
	var lastJob *time.Time
	var lastErr string
	if rs, ok := h.store.(interface {
		RunnerJobSummary(context.Context) (store.RunnerJobSummary, error)
	}); ok {
		if rsum, err := rs.RunnerJobSummary(c.Request.Context()); err == nil {
			lastJob = rsum.LastJobAt
			lastErr = rsum.LastError
		}
	}
	data["RunnerTelemetry"] = operator.BuildRunnerTelemetry(delegationEnabled, runnerJobs, lastJob, lastErr)
	if r, ok := data["Readiness"].(operator.Readiness); ok {
		data["Capabilities"] = buildCapabilityStatuses(r, h.notifyGlobal, h.platform, h.basePath)
	}
	h.renderNav(c, "health.html", "System Health", "health", data)
}

func (h *Handler) Repositories(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	rows, err := h.store.ListRepositoryControlRows(c.Request.Context(), store.ListOptions{Limit: 100})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list repositories")
		return
	}
	page := h.buildRepoControlPage(rows)
	fleetForm := h.buildFleetScanFormPlaceholder()
	data := map[string]any{"ControlPage": page, "FleetScanForm": fleetForm}
	if n := strings.TrimSpace(c.Query("notice")); n != "" {
		data["Notice"] = n
	}
	h.renderNav(c, "repos.html", "Repositories", "repos", data)
}

func (h *Handler) RepoDetail(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	scans, _ := h.store.ListScansByRepository(c.Request.Context(), id, store.ListOptions{Limit: 10})
	findings, _ := h.store.ListFindings(c.Request.Context(), store.FindingFilter{RepositoryID: id, Limit: 20})
	external, _ := h.store.ListExternalIssuesByRepository(c.Request.Context(), id, store.ListOptions{Limit: 20})
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, meta := store.ResolveEffectiveSettingsFull(h.global, settings)
	var cronInfo store.CronDescription
	if effective.ScheduleEnabled && effective.ScheduleCron != "" {
		last, _ := h.store.GetLastScheduledScanFinishedAt(c.Request.Context(), id)
		baseline := time.Now().UTC()
		if last != nil {
			baseline = last.UTC()
		}
		cronInfo = store.DescribeCron(effective.ScheduleCron, baseline)
	}
	scheduledScans, _ := h.store.ListRecentScheduledScans(c.Request.Context(), 5)
	recon, _ := h.loadReconciliation(c, id, "")
	scanForm := h.buildScanFormView(repo, effective, meta)
	data := map[string]any{
		"Repo": repo, "Scans": scans, "Findings": findings,
		"ExternalIssues": external, "Effective": effective, "ProfileMeta": meta,
		"CronInfo": cronInfo, "ScheduledScans": scheduledScans,
		"ReconcileEnabled":   h.reconcileEnabled,
		"Reconciliation":     recon,
		"ScanTriggerEnabled": h.ScanTriggerEnabled(),
		"ScanForm":           scanForm,
	}
	if started := strings.TrimSpace(c.Query("scan_started")); started != "" {
		data["ScanStartedID"] = started
		data["Notice"] = fmt.Sprintf("Manual scan queued — ID %s.", started)
	}
	h.renderNav(c, "repo_detail.html", repo.FullName, "repos", data)
}

func (h *Handler) ContainerImages(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	refs, _ := h.store.ListContainerImageReferences(c.Request.Context(), id)
	scans, _ := h.store.ListContainerImageScans(c.Request.Context(), id, 20)
	h.renderNav(c, "container_images.html", repo.FullName+" — Container Images", "repos", map[string]any{
		"RepoID": id, "Repo": repo, "References": refs, "Scans": scans,
		"Enabled": false, "RequireRunner": true, "AllowCoreSocket": false, "CreateIssues": false,
	})
}

func (h *Handler) RepoSettings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, meta := store.ResolveEffectiveSettingsFull(h.global, settings)
	var cronInfo store.CronDescription
	if effective.ScheduleCron != "" {
		last, _ := h.store.GetLastScheduledScanFinishedAt(c.Request.Context(), id)
		baseline := time.Now().UTC()
		if last != nil {
			baseline = last.UTC()
		}
		cronInfo = store.DescribeCron(effective.ScheduleCron, baseline)
	}
	selectedProfile := meta.ScanProfile
	if settings.ScanProfile != nil && *settings.ScanProfile != "" {
		selectedProfile = *settings.ScanProfile
	}
	notifyEff := notify.ResolveEffective(h.notifyGlobal, settings)
	suppressions, _ := h.store.ListFindingSuppressions(c.Request.Context(), store.SuppressionFilter{
		RepositoryID: id,
		ActiveOnly:   true,
		Limit:        100,
	})
	sections := buildRepoSettingsSections(effective, meta, h.global)
	h.renderNav(c, "repo_settings.html", "Settings — "+repo.FullName, "policies", map[string]any{
		"Repo": repo, "Settings": settings, "Effective": effective, "ProfileMeta": meta,
		"SelectedProfile": selectedProfile,
		"Profiles":        store.AllowedScanProfiles, "ProfileDescriptions": store.ProfileDescriptions,
		"Allowed": allowedSettingsDoc(), "CronInfo": cronInfo,
		"NotificationGlobal": h.notifyGlobal, "EffectiveNotifications": notifyEff,
		"NotificationEvents": store.AllowedNotificationEvents,
		"Suppressions":       suppressions,
		"SettingsSections":   sections,
		"IssueFilingEnabled": store.ShouldCreateForgeIssues(effective),
	})
}

func (h *Handler) SaveRepoSettings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := h.store.GetRepository(c.Request.Context(), id); err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}

	update := store.SettingsUpdate{
		ScanProfile:       strPtr(c.PostForm("scan_profile")),
		PolicyLevel:       strPtr(c.PostForm("policy_level")),
		WorkspaceMode:     strPtr(c.PostForm("workspace_mode")),
		SeverityGate:      strPtr(c.PostForm("severity_gate")),
		IssuePolicy:       strPtr(c.PostForm("issue_policy")),
		RemediationPolicy: strPtr(c.PostForm("remediation_policy")),
		RunnerPolicy:      strPtr(c.PostForm("runner_policy")),
		ScheduleCron:      strPtr(c.PostForm("schedule_cron")),
		AIPolicy:          strPtr(c.PostForm("ai_policy")),
	}
	if v := c.PostForm("analysis_depth"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			update.AnalysisDepth = &n
		}
	}
	if v := c.PostForm("confidence_gate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			update.ConfidenceGate = &f
		}
	}
	update.Enabled = boolPtr(c.PostForm("enabled"))
	update.EnableLLMAuditors = boolPtr(c.PostForm("enable_llm_auditors"))
	update.EnableTrivy = boolPtr(c.PostForm("enable_trivy"))
	update.EnableGrype = boolPtr(c.PostForm("enable_grype"))
	update.EnableGitleaks = boolPtr(c.PostForm("enable_gitleaks"))
	update.EnableSemgrep = boolPtr(c.PostForm("enable_semgrep"))
	update.EnableGovulncheck = boolPtr(c.PostForm("enable_govulncheck"))
	update.EnableGosec = boolPtr(c.PostForm("enable_gosec"))
	update.EnableStaticcheck = boolPtr(c.PostForm("enable_staticcheck"))
	update.EnableHadolint = boolPtr(c.PostForm("enable_hadolint"))
	update.EnableCheckov = boolPtr(c.PostForm("enable_checkov"))
	update.EnableLinters = boolPtr(c.PostForm("enable_linters"))
	update.ScheduleEnabled = boolPtr(c.PostForm("schedule_enabled"))
	update.EnableHealthChecks = boolPtr(c.PostForm("enable_health_checks"))
	update.EnableTechDebtChecks = boolPtr(c.PostForm("enable_tech_debt_checks"))
	update.EnableReliabilityChecks = boolPtr(c.PostForm("enable_reliability_checks"))
	update.EnableMaintainabilityChecks = boolPtr(c.PostForm("enable_maintainability_checks"))
	update.EnableTestGapChecks = boolPtr(c.PostForm("enable_test_gap_checks"))
	update.EnablePerformanceChecks = boolPtr(c.PostForm("enable_performance_checks"))
	update.EnableAIRiskChecks = boolPtr(c.PostForm("enable_ai_risk_checks"))
	update.HealthMaxFindings = intPtr(c.PostForm("health_max_findings"))
	update.HealthLargeFileLines = intPtr(c.PostForm("health_large_file_lines"))
	update.HealthLargeFunctionLines = intPtr(c.PostForm("health_large_function_lines"))
	update.HealthMaxNestingDepth = intPtr(c.PostForm("health_max_nesting_depth"))
	update.HealthMaxFunctionParams = intPtr(c.PostForm("health_max_function_params"))
	update.EnableCodeGraph = boolPtr(c.PostForm("enable_code_graph"))
	update.GraphMaxNodes = intPtr(c.PostForm("graph_max_nodes"))
	update.GraphMaxEdges = intPtr(c.PostForm("graph_max_edges"))
	update.GraphTimeoutSeconds = intPtr(c.PostForm("graph_timeout_seconds"))
	update.GraphIncludeFunctions = boolPtr(c.PostForm("graph_include_functions"))
	update.GraphIncludeFindings = boolPtr(c.PostForm("graph_include_findings"))
	update.GovulncheckTimeoutSeconds = intPtr(c.PostForm("govulncheck_timeout_seconds"))
	update.GosecTimeoutSeconds = intPtr(c.PostForm("gosec_timeout_seconds"))
	update.StaticcheckTimeoutSeconds = intPtr(c.PostForm("staticcheck_timeout_seconds"))
	update.GoScannerMaxFindings = intPtr(c.PostForm("go_scanner_max_findings"))
	update.HadolintTimeoutSeconds = intPtr(c.PostForm("hadolint_timeout_seconds"))
	update.CheckovTimeoutSeconds = intPtr(c.PostForm("checkov_timeout_seconds"))
	update.IACScannerMaxFindings = intPtr(c.PostForm("iac_scanner_max_findings"))
	update.NotificationsEnabled = boolPtr(c.PostForm("notifications_enabled"))
	update.NotificationMinSeverity = strPtr(c.PostForm("notification_min_severity"))
	update.NotificationEvents = strPtr(c.PostForm("notification_events"))
	update.NotificationCooldownSeconds = intPtr(c.PostForm("notification_cooldown_seconds"))

	if err := store.ValidateSettingsUpdate(update); err != nil {
		h.render(c, "repo_settings.html", "Settings error", map[string]any{
			"Error": err.Error(), "RepoID": id,
		})
		return
	}

	existing, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	existing.RepositoryID = id
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, update)
	if err := store.ValidateRepoSettings(merged); err != nil {
		repo, _ := h.store.GetRepository(c.Request.Context(), id)
		settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
		effective, meta := store.ResolveEffectiveSettingsFull(h.global, settings)
		h.render(c, "repo_settings.html", "Settings error", map[string]any{
			"Error": err.Error(), "Repo": repo, "Settings": settings, "Effective": effective,
			"ProfileMeta": meta, "Profiles": store.AllowedScanProfiles,
			"ProfileDescriptions": store.ProfileDescriptions,
			"Allowed":             allowedSettingsDoc(),
		})
		return
	}
	if err := h.store.SaveRepoSettings(c.Request.Context(), merged); err != nil {
		c.String(http.StatusInternalServerError, "failed to save settings")
		return
	}
	q := ""
	if key := c.Query("api_key"); key != "" {
		q = "?api_key=" + url.QueryEscape(key)
	}
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/repos/%d/settings%s", h.basePath, id, q))
}

func (h *Handler) ScanDetail(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	scan, err := h.store.GetScan(c.Request.Context(), scanID)
	if err != nil {
		c.String(http.StatusNotFound, "scan not found")
		return
	}
	results, _ := h.store.ListScannerResultsByScan(c.Request.Context(), scanID)
	repo, _ := h.store.GetRepository(c.Request.Context(), scan.RepositoryID)
	runnerJob, _ := h.store.GetRunnerJobByScanID(c.Request.Context(), scanID)
	summaryView := buildScanDetailView(scan.SummaryJSON)
	instanceCount, _ := h.store.CountFindingInstancesForScan(c.Request.Context(), scanID)
	if summaryView.PersistenceExpectedCount == 0 && summaryView.IssuesFound > 0 {
		summaryView.PersistenceExpectedCount = summaryView.IssuesFound
	}
	if summaryView.PersistencePersistedCount == 0 && instanceCount > 0 {
		summaryView.PersistencePersistedCount = instanceCount
	}
	if instanceCount > 0 && summaryView.PersistenceExpectedCount > 0 && instanceCount < summaryView.PersistenceExpectedCount {
		summaryView.PersistenceIncomplete = true
	}
	repoName := ""
	if repo.FullName != "" {
		repoName = repo.FullName
	}
	recon, _ := h.loadReconciliation(c, scan.RepositoryID, scanID)
	aiReview, _ := h.store.GetAIAdvisoryReviewByScanID(c.Request.Context(), scanID)
	var aiRecs []store.AIAdvisoryRecommendation
	if aiReview.ReviewID != "" {
		aiRecs, _ = h.store.ListAIAdvisoryRecommendations(c.Request.Context(), aiReview.ReviewID)
	}
	h.renderNav(c, "scan_detail.html", "Scan "+scanID[:8], "scans", map[string]any{
		"Scan":           scan,
		"ScannerResults": results,
		"Repo":           repo,
		"RepoName":       repoName,
		"RunnerJob":      runnerJob,
		"Summary":        summaryView,
		"Reconciliation": recon,
		"ReconcileEnabled": h.reconcileEnabled,
		"ScanTriggerEnabled": h.ScanTriggerEnabled() && repo.ID > 0,
		"AIReview":       aiReview,
		"AIRecommendations": aiRecs,
		"OpenClawEnabled": h.platform.OpenClawAIReviewEnabled,
	})
}

func (h *Handler) ScanGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	scan, err := h.store.GetScan(c.Request.Context(), scanID)
	if err != nil {
		c.String(http.StatusNotFound, "scan not found")
		return
	}
	repo, _ := h.store.GetRepository(c.Request.Context(), scan.RepositoryID)
	status, _ := h.resolveGraphStatusForScan(c, scanID)
	h.renderGraphPage(c, scan.RepositoryID, scanID, status, repo, scan)
}

func (h *Handler) RepoGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	status, _ := h.resolveGraphStatusForRepo(c, id)
	h.renderGraphPage(c, id, status.ScanID, status, repo, store.Scan{})
}

func (h *Handler) renderGraphPage(c *gin.Context, repoID int64, scanID string, status store.GraphStatus, repo store.Repository, scan store.Scan) {
	settingsURL := fmt.Sprintf("%s/repos/%d/settings", h.basePath, repoID)
	scanURL := ""
	if repoID > 0 {
		scanURL = fmt.Sprintf("%s/repos/%d/scan%s", h.basePath, repoID, apiKeyQueryString(h.apiKeyFromContext(c)))
	}
	stateURL := ""
	exportURL := ""
	if scanID != "" {
		stateURL = fmt.Sprintf("%s/scans/%s/graph/data%s", h.basePath, scanID, apiKeyQueryString(h.apiKeyFromContext(c)))
		if status.State == store.GraphStateAvailable || status.State == store.GraphStateTruncated {
			exportURL = fmt.Sprintf("/api/v1/scans/%s/graph/export%s", scanID, apiKeyQueryString(h.apiKeyFromContext(c)))
		}
	} else if repoID > 0 {
		stateURL = fmt.Sprintf("%s/repos/%d/graph/data%s", h.basePath, repoID, apiKeyQueryString(h.apiKeyFromContext(c)))
		if status.State == store.GraphStateAvailable || status.State == store.GraphStateTruncated {
			exportURL = fmt.Sprintf("/api/v1/repos/%d/graph/export%s", repoID, apiKeyQueryString(h.apiKeyFromContext(c)))
		}
	}
	displayStatus := status
	displayStatus.Graph = nil
	h.render(c, "graph.html", "Repository Map", map[string]any{
		"Repo":             repo,
		"Scan":             scan,
		"GraphScanID":      scanID,
		"GraphRepoID":      repoID,
		"GraphState":       status.State,
		"GraphStateJSON":   mustJSON(displayStatus),
		"GraphStateURL":    stateURL,
		"ExportURL":        exportURL,
		"GraphSettingsURL": settingsURL,
		"ScanNowURL":       scanURL,
		"NodeCount":        status.NodeCount,
		"EdgeCount":        status.EdgeCount,
		"FailureReason":    status.FailureReason,
		"NextAction":       status.NextAction,
		"GraphEnabled":     status.GraphEnabled,
		"AnalysisDepth":    status.AnalysisDepth,
	})
}

func (h *Handler) Findings(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	filter := store.FindingFilter{
		Severity:          c.Query("severity"),
		Category:          c.Query("category"),
		Status:            c.Query("status"),
		Source:            c.Query("source"),
		IncludeSuppressed: c.Query("show_suppressed") == "1",
		OnlySuppressed:    c.Query("only_suppressed") == "1",
		Limit:             100,
	}
	if v := c.Query("repo_id"); v != "" {
		filter.RepositoryID, _ = strconv.ParseInt(v, 10, 64)
	}
	findings, err := h.store.ListFindings(c.Request.Context(), filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list findings")
		return
	}
	repos, _ := h.store.ListRepositoriesWithSummary(c.Request.Context(), store.ListOptions{Limit: 200})
	h.renderNav(c, "findings.html", "Findings", "findings", map[string]any{
		"Findings": findings, "Filter": filter, "Repositories": repos,
	})
}

func (h *Handler) FindingDetail(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	detail, err := h.store.GetFindingDetail(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "finding not found")
		return
	}
	var plan remediation.Plan
	var prEligibility patcher.EligibilityResult
	var patchAttempts []patcher.PatchAttempt
	if rec, perr := h.store.GetLatestRemediationPlanByFindingID(c.Request.Context(), id); perr == nil {
		plan = store.RemediationPlanToDomain(rec)
		if h.remediationPREnabled && h.remediationPR != nil && plan.ID != "" {
			if elig, eerr := h.remediationPR.CheckPREligibility(c.Request.Context(), plan.ID); eerr == nil {
				prEligibility = elig
			}
			if attempts, aerr := h.remediationPR.ListPatchAttempts(c.Request.Context(), plan.ID); aerr == nil {
				patchAttempts = attempts
			}
		}
	}
	suppressions, _ := h.store.ListFindingSuppressions(c.Request.Context(), store.SuppressionFilter{
		RepositoryID: detail.RepositoryID,
		ActiveOnly:   true,
		Limit:        50,
	})
	h.renderNav(c, "finding_detail.html", detail.Title, "findings", map[string]any{
		"Finding": detail, "RemediationPlan": plan, "PlannerEnabled": h.remediationEnabled,
		"PREnabled": h.remediationPREnabled, "PREligibility": prEligibility, "PatchAttempts": patchAttempts,
		"ClosureEnabled": h.closureEnabled, "ClosureEvidence": h.closureEvidenceForUI(c.Request.Context(), id),
		"LifecycleLabel":     lifecycleStageLabel(plan, patchAttempts, h.closureEvidenceForUI(c.Request.Context(), id)),
		"SuppressionEnabled": h.suppressionEnabled, "RepoSuppressions": suppressions,
		"GraphDetail": buildGraphFindingView(detail),
		"GraphMapURL": graphMapURL(h.basePath, detail.RepositoryID, detail.FilePath, detail.Source, clientAPIKeyFromRequest(c)),
	})
}

func graphMapURL(basePath string, repoID int64, filePath, source, apiKey string) string {
	if source != "graph" || repoID <= 0 || strings.TrimSpace(filePath) == "" {
		return ""
	}
	focus := "file:" + strings.ReplaceAll(filePath, "\\", "/")
	u := fmt.Sprintf("%s/repos/%d/graph?focus=%s", strings.TrimSuffix(basePath, "/"), repoID, url.QueryEscape(focus))
	if apiKey != "" {
		u += "&api_key=" + url.QueryEscape(apiKey)
	}
	return u
}

func (h *Handler) closureEvidenceForUI(ctx context.Context, findingID int64) closure.Evidence {
	if h.closure == nil {
		return closure.Evidence{FindingID: findingID, Status: closure.StatusPendingRescan, Reason: "not verified yet"}
	}
	ev, err := h.closure.GetClosureEvidence(ctx, findingID)
	if err != nil || ev.Status == "" {
		return closure.Evidence{FindingID: findingID, Status: closure.StatusPendingRescan, Reason: "not verified yet"}
	}
	return ev
}

func (h *Handler) GenerateFindingRemediation(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.remediation == nil {
		c.String(http.StatusServiceUnavailable, "remediation planner disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := h.remediation.GeneratePlan(c.Request.Context(), id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) ApproveFindingRemediation(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.remediation == nil {
		c.String(http.StatusServiceUnavailable, "remediation planner disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	planID := c.PostForm("plan_id")
	if planID == "" {
		c.String(http.StatusBadRequest, "plan_id required")
		return
	}
	if err := h.remediation.ApprovePlan(c.Request.Context(), planID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) RejectFindingRemediation(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.remediation == nil {
		c.String(http.StatusServiceUnavailable, "remediation planner disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	planID := c.PostForm("plan_id")
	if planID == "" {
		c.String(http.StatusBadRequest, "plan_id required")
		return
	}
	if err := h.remediation.RejectPlan(c.Request.Context(), planID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) AttemptFindingRemediationPR(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.remediationPR == nil {
		c.String(http.StatusServiceUnavailable, "remediation PR feature disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	planID := c.PostForm("plan_id")
	if planID == "" {
		c.String(http.StatusBadRequest, "plan_id required")
		return
	}
	if _, err := h.remediationPR.AttemptPR(c.Request.Context(), planID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) VerifyFindingClosure(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.closure == nil {
		c.String(http.StatusServiceUnavailable, "evidence closure disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := h.closure.VerifyClosure(c.Request.Context(), id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) CheckFindingClosureMerge(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.closure == nil {
		c.String(http.StatusServiceUnavailable, "evidence closure disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	attemptID := c.PostForm("attempt_id")
	if attemptID == "" {
		c.String(http.StatusBadRequest, "attempt_id required")
		return
	}
	if _, err := h.closure.CheckPatchAttemptMerge(c.Request.Context(), attemptID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) SuppressFinding(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.suppression == nil {
		c.String(http.StatusServiceUnavailable, "suppression calibration disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.suppression.SuppressFinding(c.Request.Context(), id, c.PostForm("reason"), c.PostForm("created_by")); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) MarkFindingFalsePositive(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.suppression == nil {
		c.String(http.StatusServiceUnavailable, "suppression calibration disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.suppression.MarkFalsePositive(c.Request.Context(), id, c.PostForm("reason"), c.PostForm("created_by")); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) SuppressGraphRule(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.suppression == nil {
		c.String(http.StatusServiceUnavailable, "suppression calibration disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.suppression.SuppressRuleForRepo(c.Request.Context(), id, c.PostForm("reason"), c.PostForm("created_by")); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) MarkIntentionalStandalone(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCSRF(c) || h.suppression == nil {
		c.String(http.StatusServiceUnavailable, "suppression calibration disabled")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	reason := strings.TrimSpace(c.PostForm("reason"))
	if reason == "" {
		reason = "intentionally standalone architecture"
	} else {
		reason = "intentional standalone: " + reason
	}
	if err := h.suppression.MarkIntentionalStandalone(c.Request.Context(), id, reason, c.PostForm("created_by")); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	h.redirectFindingSettings(c, id)
}

func (h *Handler) redirectFindingSettings(c *gin.Context, id int64) {
	q := ""
	if key := c.Query("api_key"); key != "" {
		q = "?api_key=" + url.QueryEscape(key)
	}
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/findings/%d%s", h.basePath, id, q))
}

func (h *Handler) Preinstall(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	enabled := h.preinstallEnabled && h.preinstallRunner != nil
	audits, _ := h.store.ListAuditRequests(c.Request.Context(), store.ListOptions{Limit: 20})
	h.renderNav(c, "preinstall.html", "Pre-install audit", "preinstall", map[string]any{
		"Audits":  audits,
		"Enabled": enabled,
	})
}

func (h *Handler) StartPreinstallAudit(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	if !h.preinstallEnabled || h.preinstallRunner == nil {
		h.renderNav(c, "preinstall.html", "Pre-install audit", "preinstall", map[string]any{
			"Enabled": false,
			"Notice":  "Pre-install audit is disabled. Set preinstall_audit_enabled=true in config and restart the service.",
		})
		return
	}
	repoURL := strings.TrimSpace(c.PostForm("repo_url"))
	depth := c.PostForm("audit_depth")
	auditID, err := h.preinstallRunner.StartAudit(c.Request.Context(), repoURL, depth)
	if err != nil {
		h.render(c, "preinstall.html", "Pre-install audit", map[string]any{
			"Error":   err.Error(),
			"RepoURL": repoURL,
			"Depth":   depth,
			"Enabled": true,
		})
		return
	}
	q := url.Values{}
	if key := c.GetHeader("X-Bugbot-API-Key"); key != "" {
		q.Set("api_key", key)
	} else if key := c.Query("api_key"); key != "" {
		q.Set("api_key", key)
	}
	dest := fmt.Sprintf("%s/preinstall/audits/%s", h.basePath, auditID)
	if enc := q.Encode(); enc != "" {
		dest += "?" + enc
	}
	c.Redirect(http.StatusSeeOther, dest)
}

func (h *Handler) PreinstallAuditDetail(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	auditID := c.Param("audit_id")
	audit, err := h.store.GetAuditRequest(c.Request.Context(), auditID)
	if err != nil {
		c.String(http.StatusNotFound, "audit not found")
		return
	}
	findings, _ := h.store.ListAuditFindings(c.Request.Context(), auditID)
	reports, _ := h.store.ListDisclosureReports(c.Request.Context(), auditID)
	h.render(c, "preinstall_audit.html", "Audit "+auditID, map[string]any{
		"Audit":    audit,
		"Findings": findings,
		"Reports":  reports,
	})
}

func (h *Handler) MarkPreinstallReportReviewed(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	id, ok := parseID(c, "report_id")
	if !ok {
		return
	}
	_ = h.store.MarkDisclosureReportReviewed(c.Request.Context(), id)
	report, err := h.store.GetDisclosureReport(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "report not found")
		return
	}
	q := url.Values{}
	if key := c.GetHeader("X-Bugbot-API-Key"); key != "" {
		q.Set("api_key", key)
	} else if key := c.Query("api_key"); key != "" {
		q.Set("api_key", key)
	}
	dest := fmt.Sprintf("%s/preinstall/audits/%s", h.basePath, report.AuditID)
	if enc := q.Encode(); enc != "" {
		dest += "?" + enc
	}
	c.Redirect(http.StatusSeeOther, dest)
}

func (h *Handler) Configure(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	var readiness operator.Readiness
	if h.readinessFn != nil {
		readiness = h.readinessFn()
	}
	caps := buildCapabilityStatuses(readiness, h.notifyGlobal, h.platform, h.basePath)
	sections := buildConfigureSections(readiness, h.platform, h.notifyGlobal, h.global, h.basePath)
	h.renderNav(c, "configure.html", "Configure", "settings", map[string]any{
		"Readiness":     readiness,
		"Capabilities":  caps,
		"Platform":      h.platform,
		"Sections":      sections,
		"SetupComplete": h.isSetupComplete(c.Request.Context()),
	})
}

func (h *Handler) Learning(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	ctx := c.Request.Context()
	health, _ := h.store.LearningHealthSummary(ctx)
	recs, _ := h.store.ListCalibrationRecommendations(ctx, "proposed", 50)
	aiRecs, _ := h.store.ListPendingAIAdvisoryRecommendations(ctx, 50)
	h.renderNav(c, "learning.html", "Learning & Calibration", "learning", map[string]any{
		"Health":                health,
		"Recommendations":       recs,
		"AIRecommendations":     aiRecs,
		"OpenClawEnabled":       h.platform.OpenClawAIReviewEnabled,
		"OpenClawConfigured":    h.platform.OpenClawEndpointConfigured,
	})
}

func (h *Handler) ProjectGroups(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	groups, err := h.store.ListProjectGroups(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list project groups")
		return
	}
	repos, _ := h.store.ListRepositoriesWithSummary(c.Request.Context(), store.ListOptions{Limit: 500})
	h.renderNav(c, "projects.html", "Project groups", "projects", map[string]any{
		"Groups": groups,
		"Repos":  repos,
	})
}

func (h *Handler) CreateProjectGroup(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	name := strings.TrimSpace(c.PostForm("name"))
	desc := strings.TrimSpace(c.PostForm("description"))
	primaryID, _ := strconv.ParseInt(c.PostForm("primary_repository_id"), 10, 64)
	repoIDs := c.PostFormArray("repository_ids")
	if name == "" {
		h.renderNav(c, "projects.html", "Project groups", "projects", map[string]any{
			"Notice": "Project group name is required.",
		})
		return
	}
	var ids []int64
	for _, raw := range repoIDs {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	if primaryID <= 0 && len(ids) > 0 {
		primaryID = ids[0]
	}
	if _, err := h.store.CreateProjectGroup(c.Request.Context(), store.ProjectGroup{
		Name: name, Description: desc, PrimaryRepositoryID: primaryID, RepositoryIDs: ids,
	}); err != nil {
		h.renderNav(c, "projects.html", "Project groups", "projects", map[string]any{
			"Notice": "Failed to create project group: " + err.Error(),
		})
		return
	}
	c.Redirect(http.StatusSeeOther, h.basePath+"/projects"+apiKeyQuery(c))
}

func apiKeyQuery(c *gin.Context) string {
	if key := c.Query("api_key"); key != "" {
		return "?api_key=" + url.QueryEscape(key)
	}
	return ""
}

func parseID(c *gin.Context, param string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id <= 0 {
		c.String(http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func boolPtr(v string) *bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return nil
	}
	b := v == "true" || v == "1" || v == "on" || v == "yes"
	return &b
}

func intPtr(v string) *int {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

func allowedSettingsDoc() map[string][]string {
	return map[string][]string{
		"scan_profile":              store.AllowedScanProfiles,
		"policy_level":              store.AllowedPolicyLevels,
		"workspace_mode":            store.AllowedWorkspaceModes,
		"severity_gate":             store.AllowedSeverities,
		"notification_min_severity": store.AllowedSeverities,
		"notification_events":       store.AllowedNotificationEvents,
		"issue_policy":              store.AllowedIssuePolicies,
		"remediation_policy":        store.AllowedRemediationPolicies,
		"runner_policy":             store.AllowedRunnerPolicies,
		"ai_policy":                 store.AllowedAIPolicies,
	}
}

func lifecycleStageLabel(plan remediation.Plan, attempts []patcher.PatchAttempt, ev closure.Evidence) string {
	switch ev.Status {
	case closure.StatusVerified:
		return "Verified resolved"
	case closure.StatusBlocked:
		return "Blocked: scanner did not run"
	case closure.StatusStillPresent:
		return "Still present after remediation"
	case closure.StatusPendingRescan:
		return "Waiting for rescan"
	}
	for _, a := range attempts {
		if a.Status == patcher.StatusPROpened {
			return "PR opened, not merged"
		}
		if a.Status == "pr_merged" {
			return "Waiting for rescan"
		}
	}
	if plan.ID != "" {
		if plan.Status == remediation.StatusApproved {
			return "Approved plan — ready for remediation PR"
		}
		return "Planning only"
	}
	return "Open finding"
}
