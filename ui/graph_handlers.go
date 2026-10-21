package ui

import (
	"context"
	"encoding/json"
	"net/http"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (h *Handler) registerGraphRoutes(g *gin.RouterGroup) {
	g.GET("/scans/:scan_id/graph/data", h.ScanGraphData)
	g.GET("/repos/:id/graph/data", h.RepoGraphData)
}

func (h *Handler) ScanGraphData(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	status, err := h.resolveGraphStatusForScan(c, c.Param("scan_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.writeGraphStatus(c, status)
}

func (h *Handler) RepoGraphData(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	status, err := h.resolveGraphStatusForRepo(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.writeGraphStatus(c, status)
}

func (h *Handler) writeGraphStatus(c *gin.Context, status store.GraphStatus) {
	code := http.StatusOK
	if status.State == store.GraphStateScanNotFound || status.State == store.GraphStateRepoNotFound {
		code = http.StatusNotFound
	}
	c.JSON(code, status)
}

func (h *Handler) resolveGraphStatusForScan(c *gin.Context, scanID string) (store.GraphStatus, error) {
	if gs, ok := h.store.(graphStatusStore); ok {
		return gs.GraphStatusForScan(c.Request.Context(), scanID, h.global)
	}
	return store.GraphStatus{}, nil
}

func (h *Handler) resolveGraphStatusForRepo(c *gin.Context, repoID int64) (store.GraphStatus, error) {
	if gs, ok := h.store.(graphStatusStore); ok {
		return gs.GraphStatusForRepository(c.Request.Context(), repoID, h.global)
	}
	return store.GraphStatus{}, nil
}

type graphStatusStore interface {
	GraphStatusForScan(ctx context.Context, scanID string, global store.GlobalSettingsSnapshot) (store.GraphStatus, error)
	GraphStatusForRepository(ctx context.Context, repositoryID int64, global store.GlobalSettingsSnapshot) (store.GraphStatus, error)
}
