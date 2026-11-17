package api

import (
	"net/http"
	"strconv"

	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// OpenClawReviewService runs advisory AI reviews.
type OpenClawReviewService interface {
	Config() openclaw.Config
	RunReview(c *gin.Context, scanID string) (openclaw.ReviewResult, error)
	GetReview(c *gin.Context, scanID string) (store.AIAdvisoryReview, []store.AIAdvisoryRecommendation, error)
	AcceptRecommendation(c *gin.Context, id int64) error
	RejectRecommendation(c *gin.Context, id int64) error
	ListPendingRecommendations(c *gin.Context, limit int) ([]store.AIAdvisoryRecommendation, error)
}

// OpenClawReviewHandler serves OpenClaw advisory review routes.
type OpenClawReviewHandler struct {
	service OpenClawReviewService
}

// NewOpenClawReviewHandler creates the handler.
func NewOpenClawReviewHandler(s OpenClawReviewService) *OpenClawReviewHandler {
	return &OpenClawReviewHandler{service: s}
}

// RegisterRoutes mounts AI recommendations API routes (legacy /openclaw/* aliases retained).
func (h *OpenClawReviewHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/ai-recommendations/config", h.GetConfig)
	g.GET("/openclaw/config", h.GetConfig)
	g.GET("/scans/:scan_id/ai-recommendations", h.GetReview)
	g.POST("/scans/:scan_id/ai-recommendations", h.RunReview)
	g.GET("/scans/:scan_id/ai-review", h.GetReview)
	g.POST("/scans/:scan_id/ai-review", h.RunReview)
	g.GET("/ai-recommendations/pending", h.ListPending)
	g.GET("/ai-review/recommendations/pending", h.ListPending)
	g.POST("/ai-recommendations/:id/accept", h.Accept)
	g.POST("/ai-recommendations/:id/reject", h.Reject)
	g.POST("/ai-review/recommendations/:id/accept", h.Accept)
	g.POST("/ai-review/recommendations/:id/reject", h.Reject)
}

func (h *OpenClawReviewHandler) requireService(c *gin.Context) bool {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai recommendations unavailable"})
		return false
	}
	return true
}

func (h *OpenClawReviewHandler) GetConfig(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	cfg := h.service.Config().Normalized()
	c.JSON(http.StatusOK, gin.H{
		"feature":                   "ai_recommendations",
		"provider":                  cfg.Provider,
		"enabled":                   cfg.Enabled,
		"endpoint_configured":       cfg.EndpointConfigured(),
		"model":                     cfg.EffectiveModel(),
		"max_tokens_per_scan":       cfg.MaxTokensPerScan,
		"max_findings_per_scan":     cfg.MaxFindingsPerScan,
		"send_source_snippets":      cfg.SendSourceSnippets,
		"send_full_files":           cfg.SendFullFiles,
		"redact_secrets":            cfg.RedactSecrets,
		"redact_pii":                cfg.RedactPII,
		"advisory_only":             cfg.AdvisoryOnly,
		"require_operator_approval": cfg.RequireOperatorApproval,
		"auto_after_scan":           cfg.AutoAfterScan,
		"allow_preinstall":          cfg.AllowPreinstall,
		"allow_container_scans":     cfg.AllowContainerScans,
		"allow_repo_scans":          cfg.AllowRepoScans,
		"cah_enabled":               cfg.CAH.Enabled,
		"cah_max_candidates":        cfg.CAH.MaxCandidates,
		"cah_min_uncertainty_score": cfg.CAH.MinUncertaintyScore,
		"token_budget_per_scan":     cfg.CAH.TokenBudgetPerScan,
		"note":                      "Advisory only — deterministic scanners remain source of truth.",
	})
}

func (h *OpenClawReviewHandler) GetReview(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	scanID := c.Param("scan_id")
	review, recs, err := h.service.GetReview(c, scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"review": review, "recommendations": recs})
}

func (h *OpenClawReviewHandler) RunReview(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	scanID := c.Param("scan_id")
	result, err := h.service.RunReview(c, scanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

func (h *OpenClawReviewHandler) ListPending(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	recs, err := h.service.ListPendingRecommendations(c, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recommendations": recs, "count": len(recs)})
}

func (h *OpenClawReviewHandler) Accept(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.AcceptRecommendation(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "accepted", "note": "Calibration draft only — not auto-applied."})
}

func (h *OpenClawReviewHandler) Reject(c *gin.Context) {
	if !h.requireService(c) {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.RejectRecommendation(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}
