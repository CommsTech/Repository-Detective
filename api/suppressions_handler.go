package api

import (
	"net/http"
	"strconv"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// SuppressionRequest is the JSON body for finding-level calibration actions.
type SuppressionRequest struct {
	Reason    string     `json:"reason"`
	CreatedBy string     `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Scope     string     `json:"scope,omitempty"`
}

// CreateSuppressionRequest creates a standalone suppression rule.
type CreateSuppressionRequest struct {
	SuppressionRequest
	RepositoryID *int64 `json:"repository_id,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	Source       string `json:"source,omitempty"`
	RuleID       string `json:"rule_id,omitempty"`
	Category     string `json:"category,omitempty"`
	Severity     string `json:"severity,omitempty"`
}

// SuppressionService applies calibration actions to findings and forge issues.
type SuppressionService interface {
	SuppressFinding(c *gin.Context, findingID int64, req SuppressionRequest) (store.FindingSuppression, error)
	MarkFalsePositive(c *gin.Context, findingID int64, req SuppressionRequest) (store.FindingSuppression, error)
	CreateSuppression(c *gin.Context, req CreateSuppressionRequest) (store.FindingSuppression, error)
	DisableSuppression(c *gin.Context, id int64) (store.FindingSuppression, error)
}

// SuppressionsHandler serves calibration API routes.
type SuppressionsHandler struct {
	store   store.QueryStore
	service SuppressionService
}

// NewSuppressionsHandler creates a suppressions API handler.
func NewSuppressionsHandler(s store.QueryStore, svc SuppressionService) *SuppressionsHandler {
	return &SuppressionsHandler{store: s, service: svc}
}

func (h *SuppressionsHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database disabled",
			"message": "Enable database_enabled to use suppressions",
		})
		return false
	}
	return true
}

// RegisterRoutes mounts suppression routes on the authenticated API group.
func (h *SuppressionsHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/findings/:id/suppress", h.SuppressFinding)
	g.POST("/findings/:id/mark-false-positive", h.MarkFalsePositive)
	g.POST("/suppressions", h.CreateSuppression)
	g.GET("/suppressions", h.ListSuppressions)
	g.POST("/suppressions/:id/disable", h.DisableSuppression)
	g.GET("/analytics/scan-quality", h.ScanQualityReport)
}

func (h *SuppressionsHandler) requireService(c *gin.Context) bool {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "suppression service unavailable"})
		return false
	}
	return true
}

func (h *SuppressionsHandler) SuppressFinding(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	var req SuppressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	sup, err := h.service.SuppressFinding(c, id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suppression": toSuppressionResponse(sup)})
}

func (h *SuppressionsHandler) MarkFalsePositive(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	var req SuppressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	sup, err := h.service.MarkFalsePositive(c, id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suppression": toSuppressionResponse(sup)})
}

func (h *SuppressionsHandler) CreateSuppression(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	var req CreateSuppressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	sup, err := h.service.CreateSuppression(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"suppression": toSuppressionResponse(sup)})
}

func (h *SuppressionsHandler) ListSuppressions(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	repoID, _ := strconv.ParseInt(c.Query("repository_id"), 10, 64)
	filter := store.SuppressionFilter{
		RepositoryID: repoID,
		Scope:        c.Query("scope"),
		ActiveOnly:   c.Query("active") != "false",
		Limit:        listOptions(c).Limit,
		Offset:       listOptions(c).Offset,
	}
	sups, err := h.store.ListFindingSuppressions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list suppressions"})
		return
	}
	out := make([]suppressionResponse, 0, len(sups))
	for _, sup := range sups {
		out = append(out, toSuppressionResponse(sup))
	}
	c.JSON(http.StatusOK, gin.H{"suppressions": out})
}

func (h *SuppressionsHandler) DisableSuppression(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid suppression id"})
		return
	}
	sup, err := h.service.DisableSuppression(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suppression": toSuppressionResponse(sup)})
}

func (h *SuppressionsHandler) ScanQualityReport(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	report, err := h.store.ScanQualityReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build scan quality report"})
		return
	}
	c.JSON(http.StatusOK, toScanQualityReportResponse(report))
}

type suppressionResponse struct {
	ID           int64      `json:"id"`
	RepositoryID *int64     `json:"repository_id,omitempty"`
	Fingerprint  string     `json:"fingerprint,omitempty"`
	Source       string     `json:"source,omitempty"`
	RuleID       string     `json:"rule_id,omitempty"`
	Category     string     `json:"category,omitempty"`
	Severity     string     `json:"severity,omitempty"`
	Scope        string     `json:"scope"`
	Reason       string     `json:"reason"`
	CreatedBy    string     `json:"created_by"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func toSuppressionResponse(sup store.FindingSuppression) suppressionResponse {
	return suppressionResponse{
		ID: sup.ID, RepositoryID: sup.RepositoryID, Fingerprint: sup.Fingerprint,
		Source: sup.Source, RuleID: sup.RuleID, Category: sup.Category, Severity: sup.Severity,
		Scope: sup.Scope, Reason: sup.Reason, CreatedBy: sup.CreatedBy, ExpiresAt: sup.ExpiresAt,
		Active: sup.Active, CreatedAt: sup.CreatedAt, UpdatedAt: sup.UpdatedAt,
	}
}

type scanQualityReportResponse struct {
	ReposScanned              int                        `json:"repos_scanned"`
	TotalFindings             int                        `json:"total_findings"`
	OpenFindings              int                        `json:"open_findings"`
	SuppressedFindings        int                        `json:"suppressed_findings"`
	FalsePositiveFindings     int                        `json:"false_positive_findings"`
	FindingsBySeverity        map[string]int             `json:"findings_by_severity"`
	FindingsByCategory        map[string]int             `json:"findings_by_category"`
	FindingsBySource          map[string]int             `json:"findings_by_source"`
	ExternalIssuesOpen        int                        `json:"external_issues_open"`
	RemediationPlansGenerated int                        `json:"remediation_plans_generated"`
	PatchAttemptsOpened       int                        `json:"patch_attempts_opened"`
	PatchAttemptsVerified     int                        `json:"patch_attempts_verified"`
	ScannerFailures           int                        `json:"scanner_failures"`
	ReposWithNoFindings       int                        `json:"repos_with_no_findings"`
	ReposWithCriticalHigh     int                        `json:"repos_with_critical_high"`
	ActionableFindings        int                        `json:"actionable_findings"`
	ActionableRatio           float64                    `json:"actionable_ratio"`
	StrictActionableFindings  int                        `json:"strict_actionable_findings"`
	StrictActionableRatio     float64                    `json:"strict_actionable_ratio"`
	GraphFindingsOpen         int                        `json:"graph_findings_open"`
	ReportOnlyEstimate        int                        `json:"report_only_estimate"`
	EnabledMissingScanners    int                        `json:"enabled_missing_scanners"`
	TopNoisyRules             []store.RuleCount          `json:"top_noisy_rules"`
	TopSuppressedRules        []store.RuleCount          `json:"top_suppressed_rules"`
	ScannerFailureBreakdown   []store.ScannerStatusCount `json:"scanner_failure_breakdown"`
}

func toScanQualityReportResponse(r store.ScanQualityReport) scanQualityReportResponse {
	return scanQualityReportResponse{
		ReposScanned: r.ReposScanned, TotalFindings: r.TotalFindings, OpenFindings: r.OpenFindings,
		SuppressedFindings: r.SuppressedFindings, FalsePositiveFindings: r.FalsePositiveFindings,
		FindingsBySeverity: r.FindingsBySeverity, FindingsByCategory: r.FindingsByCategory,
		FindingsBySource: r.FindingsBySource, ExternalIssuesOpen: r.ExternalIssuesOpen,
		RemediationPlansGenerated: r.RemediationPlansGenerated, PatchAttemptsOpened: r.PatchAttemptsOpened,
		PatchAttemptsVerified: r.PatchAttemptsVerified, ScannerFailures: r.ScannerFailures,
		ReposWithNoFindings: r.ReposWithNoFindings, ReposWithCriticalHigh: r.ReposWithCriticalHigh,
		ActionableFindings: r.ActionableFindings, ActionableRatio: r.ActionableRatio,
		StrictActionableFindings: r.StrictActionableFindings, StrictActionableRatio: r.StrictActionableRatio,
		GraphFindingsOpen: r.GraphFindingsOpen, ReportOnlyEstimate: r.ReportOnlyEstimate,
		EnabledMissingScanners: r.EnabledMissingScanners,
		TopNoisyRules:          r.TopNoisyRules, TopSuppressedRules: r.TopSuppressedRules,
		ScannerFailureBreakdown: r.ScannerFailureBreakdown,
	}
}
