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
	g.POST("/calibration/recommendations/:id/accept", h.Accept)
	g.POST("/calibration/recommendations/:id/reject", h.Reject)
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
