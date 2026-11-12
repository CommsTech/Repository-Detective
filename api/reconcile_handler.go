package api

import (
	"net/http"
	"strconv"

	"git.commsnet.org/commstech/repository-detective/reconcile"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// ReconcileService runs issue reconciliation for a repository.
type ReconcileService interface {
	Preview(c *gin.Context, repositoryID int64) (reconcile.Result, error)
	Apply(c *gin.Context, repositoryID int64) (reconcile.Result, error)
	GetRun(c *gin.Context, runID string) (store.ReconciliationRun, []store.ReconciliationItemRecord, error)
}

// ReconcileHandler serves issue reconciliation API routes.
type ReconcileHandler struct {
	store   store.QueryStore
	service ReconcileService
}

// NewReconcileHandler creates a reconciliation handler.
func NewReconcileHandler(s store.QueryStore, svc ReconcileService) *ReconcileHandler {
	return &ReconcileHandler{store: s, service: svc}
}

// RegisterRoutes mounts reconciliation routes.
func (h *ReconcileHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/repos/:id/reconcile-issues/preview", h.Preview)
	g.POST("/repos/:id/reconcile-issues", h.Apply)
	g.GET("/issues/reconciliation/:run_id", h.GetRun)
}

func (h *ReconcileHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

func (h *ReconcileHandler) repoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return 0, false
	}
	return id, true
}

func (h *ReconcileHandler) Preview(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, ok := h.repoID(c)
	if !ok {
		return
	}
	result, err := h.service.Preview(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ReconcileHandler) Apply(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, ok := h.repoID(c)
	if !ok {
		return
	}
	result, err := h.service.Apply(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ReconcileHandler) GetRun(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	runID := c.Param("run_id")
	run, items, err := h.service.GetRun(c, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"run": run, "items": items})
}
