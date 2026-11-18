package ui

import (
	"net/http"
	"os"
	"strings"

	"git.commsnet.org/commstech/repository-detective/vulnintel"
	"github.com/gin-gonic/gin"
)

var cveIntelligenceClient = vulnintel.NewClient()

func cveIntelligenceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REPOSITORY_DETECTIVE_CVE_INTELLIGENCE_ENABLED"))) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// FindingIntelligence is an authenticated, opt-in, on-demand endpoint.
// Only a validated public CVE ID leaves the instance for CISA/FIRST, never
// source code, repository names, credentials or inventory details.
func (h *Handler) FindingIntelligence(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !cveIntelligenceEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Public CVE lookups are disabled by the operator"})
		return
	}
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	finding, err := h.store.GetFindingDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "finding not found"})
		return
	}
	view := buildActionableFindingView(finding)
	if !view.IsDependencyVuln || view.CVEID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "No dependency CVE identifier available for this finding"})
		return
	}
	report, err := cveIntelligenceClient.Lookup(c.Request.Context(), view.CVEID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CVE identifier"})
		return
	}
	c.JSON(http.StatusOK, report)
}
