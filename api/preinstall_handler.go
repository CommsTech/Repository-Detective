package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PreinstallHandler serves pre-install audit API routes.
type PreinstallHandler struct {
	store  store.QueryStore
	runner *preinstall.Runner
	logger *logrus.Logger
}

// NewPreinstallHandler creates a pre-install audit API handler.
func NewPreinstallHandler(s store.QueryStore, runner *preinstall.Runner, logger *logrus.Logger) *PreinstallHandler {
	return &PreinstallHandler{store: s, runner: runner, logger: logger}
}

// RegisterRoutes mounts pre-install audit routes on the given group.
func (h *PreinstallHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/preinstall/audits", h.CreateAudit)
	g.GET("/preinstall/audits", h.ListAudits)
	g.GET("/preinstall/audits/:audit_id", h.GetAudit)
	g.GET("/preinstall/audits/:audit_id/findings", h.ListAuditFindings)
	g.GET("/preinstall/audits/:audit_id/reports", h.ListAuditReports)
	g.GET("/preinstall/reports/:report_id", h.GetReport)
	g.POST("/preinstall/reports/:report_id/mark-reviewed", h.MarkReportReviewed)
}

type createAuditRequest struct {
	RepoURL    string `json:"repo_url"`
	AuditDepth string `json:"audit_depth"`
}

func (h *PreinstallHandler) requireReady(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	if h.runner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pre-install audit unavailable"})
		return false
	}
	return true
}

func (h *PreinstallHandler) CreateAudit(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	var body createAuditRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	auditID, err := h.runner.StartAudit(c.Request.Context(), body.RepoURL, body.AuditDepth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"audit_id": auditID, "status": store.AuditStatusQueued})
}

func (h *PreinstallHandler) ListAudits(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	audits, err := h.store.ListAuditRequests(c.Request.Context(), listOptions(c))
	if err != nil {
		h.logger.Errorf("list audits: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audits"})
		return
	}
	out := make([]auditResponse, 0, len(audits))
	for _, a := range audits {
		out = append(out, toAuditResponse(a))
	}
	c.JSON(http.StatusOK, gin.H{"audits": out})
}

func (h *PreinstallHandler) GetAudit(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	auditID := c.Param("audit_id")
	audit, err := h.store.GetAuditRequest(c.Request.Context(), auditID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit not found"})
		return
	}
	c.JSON(http.StatusOK, toAuditResponse(audit))
}

func (h *PreinstallHandler) ListAuditFindings(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	auditID := c.Param("audit_id")
	if _, err := h.store.GetAuditRequest(c.Request.Context(), auditID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit not found"})
		return
	}
	findings, err := h.store.ListAuditFindings(c.Request.Context(), auditID)
	if err != nil {
		h.logger.Errorf("list audit findings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list findings"})
		return
	}
	out := make([]auditFindingResponse, 0, len(findings))
	for _, f := range findings {
		out = append(out, toAuditFindingResponse(f))
	}
	c.JSON(http.StatusOK, gin.H{"findings": out})
}

func (h *PreinstallHandler) ListAuditReports(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	auditID := c.Param("audit_id")
	if _, err := h.store.GetAuditRequest(c.Request.Context(), auditID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit not found"})
		return
	}
	reports, err := h.store.ListDisclosureReports(c.Request.Context(), auditID)
	if err != nil {
		h.logger.Errorf("list audit reports: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reports"})
		return
	}
	out := make([]disclosureReportResponse, 0, len(reports))
	for _, r := range reports {
		out = append(out, toDisclosureReportResponse(r))
	}
	c.JSON(http.StatusOK, gin.H{"reports": out})
}

func (h *PreinstallHandler) GetReport(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	id, ok := parseReportID(c)
	if !ok {
		return
	}
	report, err := h.store.GetDisclosureReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	c.JSON(http.StatusOK, toDisclosureReportResponse(report))
}

func (h *PreinstallHandler) MarkReportReviewed(c *gin.Context) {
	if !h.requireReady(c) {
		return
	}
	id, ok := parseReportID(c)
	if !ok {
		return
	}
	if err := h.store.MarkDisclosureReportReviewed(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	report, err := h.store.GetDisclosureReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusOK, toDisclosureReportResponse(report))
}

func parseReportID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("report_id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return 0, false
	}
	return id, true
}

type auditResponse struct {
	AuditID           string  `json:"audit_id"`
	RepoURL           string  `json:"repo_url"`
	NormalizedRepoURL string  `json:"normalized_repo_url"`
	RepoHost          string  `json:"repo_host"`
	RepoOwner         string  `json:"repo_owner"`
	RepoName          string  `json:"repo_name"`
	CommitSHA         string  `json:"commit_sha"`
	DefaultBranch     string  `json:"default_branch"`
	AuditDepth        string  `json:"audit_depth"`
	Status            string  `json:"status"`
	RiskScore         int     `json:"risk_score"`
	RiskScoreDisplay  string  `json:"risk_score_display"`
	RiskUnavailable   bool    `json:"risk_unavailable"`
	Recommendation    string  `json:"recommendation"`
	RecommendationDisplay string `json:"recommendation_display"`
	FailureStage      string  `json:"failure_stage,omitempty"`
	NextAction          string  `json:"next_action,omitempty"`
	StartedAt         string  `json:"started_at"`
	FinishedAt        *string `json:"finished_at,omitempty"`
	SummaryJSON       any     `json:"summary_json"`
	Error             string  `json:"error,omitempty"`
}

type auditFindingResponse struct {
	ID               int64   `json:"id"`
	Fingerprint      string  `json:"fingerprint"`
	Category         string  `json:"category"`
	Severity         string  `json:"severity"`
	Confidence       float64 `json:"confidence"`
	Source           string  `json:"source"`
	RuleID           string  `json:"rule_id"`
	FilePath         string  `json:"file_path"`
	Line             int     `json:"line"`
	Title            string  `json:"title"`
	EvidenceRedacted string  `json:"evidence_redacted"`
}

type disclosureReportResponse struct {
	ID                  int64   `json:"id"`
	AuditID             string  `json:"audit_id"`
	FindingID           *int64  `json:"finding_id,omitempty"`
	ReportType          string  `json:"report_type"`
	Sensitivity         string  `json:"sensitivity"`
	Title               string  `json:"title"`
	BodyMarkdown        string  `json:"body_markdown"`
	Confidence          float64 `json:"confidence"`
	ApprovedByUser      bool    `json:"approved_by_user"`
	SubmittedExternally bool    `json:"submitted_externally"`
	GeneratedAt         string  `json:"generated_at"`
}

func toAuditResponse(a store.AuditRequest) auditResponse {
	resp := auditResponse{
		AuditID:               a.AuditID,
		RepoURL:               a.RepoURL,
		NormalizedRepoURL:     a.NormalizedRepoURL,
		RepoHost:              a.RepoHost,
		RepoOwner:             a.RepoOwner,
		RepoName:              a.RepoName,
		CommitSHA:             a.CommitSHA,
		DefaultBranch:         a.DefaultBranch,
		AuditDepth:            a.AuditDepth,
		Status:                a.Status,
		RiskScore:             a.RiskScore,
		RiskScoreDisplay:      preinstall.RiskScoreDisplay(a),
		RiskUnavailable:       a.Status == store.AuditStatusFailed || a.RiskScore < 0,
		Recommendation:        a.Recommendation,
		RecommendationDisplay: preinstall.RecommendationDisplay(a),
		FailureStage:          preinstall.FailureStageFromSummary(a.SummaryJSON),
		StartedAt:             a.StartedAt.UTC().Format(time.RFC3339),
		Error:                 preinstall.SanitizeFailureMessage(a.Error),
	}
	if a.Status == store.AuditStatusFailed {
		resp.RiskScore = preinstall.RiskScoreUnavailable
	}
	if len(a.SummaryJSON) > 0 {
		var summary map[string]any
		if json.Unmarshal(a.SummaryJSON, &summary) == nil {
			if s, ok := summary["next_action"].(string); ok {
				resp.NextAction = s
			}
		}
	}
	if a.FinishedAt != nil {
		s := a.FinishedAt.UTC().Format(time.RFC3339)
		resp.FinishedAt = &s
	}
	if len(a.SummaryJSON) > 0 {
		resp.SummaryJSON = jsonRaw(a.SummaryJSON)
	}
	return resp
}

func toAuditFindingResponse(f store.AuditFinding) auditFindingResponse {
	return auditFindingResponse{
		ID:               f.ID,
		Fingerprint:      f.Fingerprint,
		Category:         f.Category,
		Severity:         f.Severity,
		Confidence:       f.Confidence,
		Source:           f.Source,
		RuleID:           f.RuleID,
		FilePath:         f.FilePath,
		Line:             f.Line,
		Title:            f.Title,
		EvidenceRedacted: f.EvidenceRedacted,
	}
}

func toDisclosureReportResponse(r store.DisclosureReport) disclosureReportResponse {
	return disclosureReportResponse{
		ID:                  r.ID,
		AuditID:             r.AuditID,
		FindingID:           r.FindingID,
		ReportType:          r.ReportType,
		Sensitivity:         r.Sensitivity,
		Title:               r.Title,
		BodyMarkdown:        r.BodyMarkdown,
		Confidence:          r.Confidence,
		ApprovedByUser:      r.ApprovedByUser,
		SubmittedExternally: r.SubmittedExternally,
		GeneratedAt:         r.GeneratedAt.UTC().Format(time.RFC3339),
	}
}

func jsonRaw(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
