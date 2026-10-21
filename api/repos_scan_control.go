package api

import (
	"fmt"
	"net/http"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerRepoScanControlRoutes(g *gin.RouterGroup) {
	g.POST("/repos/:id/enable-scanning", h.EnableRepoScanning)
	g.POST("/repos/:id/disable-scanning", h.DisableRepoScanning)
}

func (h *Handler) EnableRepoScanning(c *gin.Context) {
	h.setRepoScanEnabledAPI(c, true)
}

func (h *Handler) DisableRepoScanning(c *gin.Context) {
	h.setRepoScanEnabledAPI(c, false)
}

func (h *Handler) setRepoScanEnabledAPI(c *gin.Context, enabled bool) {
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
	existing, err := h.store.GetRepoSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	existing.RepositoryID = id
	existing.UpdatedAt = time.Now().UTC()
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, store.SettingsUpdate{Enabled: &enabled})
	if err := store.ValidateRepoSettings(merged); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SaveRepoSettings(c.Request.Context(), merged); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
		return
	}
	eventType := "repo_scan_disabled"
	if enabled {
		eventType = "repo_scan_enabled"
	}
	_, err = h.store.RecordLearningEvent(c.Request.Context(), store.LearningEvent{
		RepositoryID:   id,
		EventType:      eventType,
		Source:         "api",
		CreatedBy:      "api/v1/repos",
		EvidenceJSON:   []byte(fmt.Sprintf(`{"enabled":%t}`, enabled)),
		IdempotencyKey: fmt.Sprintf("%d:%s:%d", id, eventType, time.Now().UnixNano()),
	})
	if err != nil {
		// Settings save already succeeded; learning is best-effort.
		c.Header("X-Learning-Event-Error", "1")
	}
	c.JSON(http.StatusOK, toSettingsResponse(id, merged, h.global, h.notifyGlobal))
}
