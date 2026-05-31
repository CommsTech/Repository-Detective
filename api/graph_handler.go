package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetScanGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	record, err := h.store.GetScanGraph(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not found"})
		return
	}
	c.Data(http.StatusOK, "application/json", record.GraphJSON)
}

func (h *Handler) ExportScanGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	scanID := c.Param("scan_id")
	record, err := h.store.GetScanGraph(c.Request.Context(), scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not found"})
		return
	}
	filename := fmt.Sprintf("graph-scan-%s.json", scanID)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", record.GraphJSON)
}

func (h *Handler) GetRepoGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	record, err := h.store.GetLatestScanGraphForRepo(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not found"})
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(record.GraphJSON, &payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid graph data"})
		return
	}
	payload["scan_id"] = record.ScanID
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) ExportRepoGraph(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseRepoID(c)
	if !ok {
		return
	}
	record, err := h.store.GetLatestScanGraphForRepo(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "graph not found"})
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(record.GraphJSON, &payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid graph data"})
		return
	}
	payload["scan_id"] = record.ScanID
	body, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode graph"})
		return
	}
	filename := fmt.Sprintf("graph-repo-%d.json", id)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", body)
}
