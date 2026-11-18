package api

import (
	"context"
	"net/http"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetScanGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	status, err := h.graphStatusForScan(c, scanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	code := http.StatusOK
	if status.State == store.GraphStateScanNotFound {
		code = http.StatusNotFound
	}
	c.JSON(code, status)
}

func (h *Handler) ExportScanGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	status, err := h.graphStatusForScan(c, scanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if status.State != store.GraphStateAvailable && status.State != store.GraphStateTruncated {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not available", "state": status.State})
		return
	}
	filename := security.SafeAttachmentFilename("graph-scan", scanID) + ".json"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", status.Graph)
}

func (h *Handler) GetRepoGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	status, err := h.graphStatusForRepo(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	code := http.StatusOK
	if status.State == store.GraphStateRepoNotFound {
		code = http.StatusNotFound
	}
	c.JSON(code, status)
}

func (h *Handler) ExportRepoGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	status, err := h.graphStatusForRepo(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if status.State != store.GraphStateAvailable && status.State != store.GraphStateTruncated {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not available", "state": status.State})
		return
	}
	filename := security.SafeAttachmentFilename("graph-repo", c.Param("id")) + ".json"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", status.Graph)
}

func (h *Handler) graphStatusForScan(c *gin.Context, scanID string) (store.GraphStatus, error) {
	if gs, ok := h.store.(graphStatusResolver); ok {
		return gs.GraphStatusForScan(c.Request.Context(), scanID, h.global)
	}
	return store.GraphStatus{}, nil
}

func (h *Handler) graphStatusForRepo(c *gin.Context, repoID int64) (store.GraphStatus, error) {
	if gs, ok := h.store.(graphStatusResolver); ok {
		return gs.GraphStatusForRepository(c.Request.Context(), repoID, h.global)
	}
	return store.GraphStatus{}, nil
}

type graphStatusResolver interface {
	GraphStatusForScan(ctx context.Context, scanID string, global store.GlobalSettingsSnapshot) (store.GraphStatus, error)
	GraphStatusForRepository(ctx context.Context, repositoryID int64, global store.GlobalSettingsSnapshot) (store.GraphStatus, error)
}
