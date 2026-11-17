package api

import (
	"net/http"
	"strconv"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// CalibrationService runs deterministic learning operations.
type CalibrationService interface {
	Summary(c *gin.Context) (map[string]any, error)
	ListRecommendations(c *gin.Context, status string) ([]store.CalibrationRecommendation, error)
	AcceptRecommendation(c *gin.Context, id int64) error
	RejectRecommendation(c *gin.Context, id int64) error
	RevertRecommendation(c *gin.Context, id int64) (map[string]any, error)
	Recompute(c *gin.Context) (map[string]any, error)
}

// CalibrationHandler serves calibration API routes.
type CalibrationHandler struct {
	store   store.QueryStore
	service CalibrationService
}

// NewCalibrationHandler creates a calibration handler.
func NewCalibrationHandler(s store.QueryStore, svc CalibrationService) *CalibrationHandler {
	return &CalibrationHandler{store: s, service: svc}
}

// RegisterRoutes mounts calibration routes.
func (h *CalibrationHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/calibration/summary", h.Summary)
	g.GET("/calibration/recommendations", h.ListRecommendations)
	g.GET("/calibration/history", h.History)
	g.POST("/calibration/recommendations/:id/accept", h.Accept)
	g.POST("/calibration/recommendations/:id/reject", h.Reject)
	g.POST("/calibration/recommendations/:id/revert", h.Revert)
	g.POST("/calibration/recompute", h.Recompute)
}

func (h *CalibrationHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

func (h *CalibrationHandler) Summary(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	out, err := h.service.Summary(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *CalibrationHandler) ListRecommendations(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	status := c.Query("status")
	recs, err := h.service.ListRecommendations(c, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recommendations": recs})
}

func (h *CalibrationHandler) Accept(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.service.AcceptRecommendation(c, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

func (h *CalibrationHandler) Reject(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.service.RejectRecommendation(c, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

func (h *CalibrationHandler) History(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	// status=all → full history for transparency (RD-025).
	recs, err := h.service.ListRecommendations(c, "all")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	entries := make([]gin.H, 0, len(recs))
	for _, rec := range recs {
		entries = append(entries, gin.H{
			"id":                    rec.ID,
			"repository_id":         rec.RepositoryID,
			"scope":                 rec.Scope,
			"source":                rec.Source,
			"rule_id":               rec.RuleID,
			"category":              rec.Category,
			"fingerprint_family":    rec.Source + ":" + rec.RuleID,
			"previous_behavior":     rec.CurrentAction,
			"proposed_behavior":     rec.RecommendedAction,
			"new_behavior":          rec.RecommendedAction,
			"evidence_confidence":   rec.Confidence,
			"reason":                rec.Reason,
			"actor_source":          "deterministic_calibration",
			"automatically_applied": false,
			"status":                rec.Status,
			"created_at":            rec.CreatedAt,
			"updated_at":            rec.UpdatedAt,
			"reversible":            rec.Status == "accepted",
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"history":  entries,
		"statuses": []string{"proposed", "accepted", "rejected", "reverted"},
		"note":     "High/Critical and secret categories remain protected from automatic downgrade. Revert expires linked repo_calibration_rules only.",
	})
}

func (h *CalibrationHandler) Revert(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	out, err := h.service.RevertRecommendation(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *CalibrationHandler) Recompute(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	out, err := h.service.Recompute(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
