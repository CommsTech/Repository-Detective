package ui

import (
	"fmt"
	"net/http"
	"strings"

	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerScanRoutes(g *gin.RouterGroup) {
	g.GET("/repos/:id/scan", h.RepoScanForm)
	g.POST("/repos/:id/scan", h.RepoScanStart)
}

// RepoScanForm shows the manual scan confirmation form.
func (h *Handler) RepoScanForm(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, meta := store.ResolveEffectiveSettingsFull(h.global, settings)
	issueFiling := store.ShouldCreateForgeIssues(effective)
	h.renderNav(c, "scan_now.html", "Scan — "+repo.FullName, "repos", map[string]any{
		"Repo":              repo,
		"Effective":         effective,
		"ProfileMeta":       meta,
		"Profiles":          store.AllowedScanProfiles,
		"ScanEnabled":       h.ScanTriggerEnabled(),
		"DefaultReportOnly": !issueFiling,
		"IssueFilingOn":     issueFiling,
	})
}

// RepoScanStart queues a manual scan and redirects to scan detail.
func (h *Handler) RepoScanStart(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	if !h.ScanTriggerEnabled() {
		c.String(http.StatusServiceUnavailable, "manual scan is not available")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "repository not found")
		return
	}
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, _ := store.ResolveEffectiveSettingsFull(h.global, settings)
	if !effective.Enabled {
		c.String(http.StatusBadRequest, "repository is disabled in settings")
		return
	}

	ref := strings.TrimSpace(c.PostForm("ref"))
	if ref == "" {
		ref = "main"
	}
	profile := strings.TrimSpace(c.PostForm("scan_profile"))
	reportOnly := c.PostForm("report_only_dry_run") == "on" || c.PostForm("report_only_dry_run") == "true"
	if !store.ShouldCreateForgeIssues(effective) {
		reportOnly = true
	}

	parts := strings.SplitN(repo.FullName, "/", 2)
	owner, name := parts[0], ""
	if len(parts) == 2 {
		name = parts[1]
	}
	result, err := h.triggerManualScan(c.Request.Context(), ScanTriggerRequest{
		ForgeType: repo.ForgeType, Owner: owner, Repository: name,
		Ref: ref, ScanProfile: profile, ReportOnlyDryRun: reportOnly,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to start scan: %v", err)
		return
	}
	q := apiKeyQueryString(h.apiKeyFromContext(c))
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/scans/%s%s", h.basePath, result.ScanID, q))
}

func (h *Handler) apiKeyFromContext(c *gin.Context) string {
	if key, err := c.Cookie(uiSessionCookieName); err == nil && key != "" {
		return key
	}
	return strings.TrimSpace(c.GetHeader("X-Repository-Detective-API-Key"))
}

func (h *Handler) loadReconciliation(c *gin.Context, repositoryID int64, scanID string) (store.ReconciliationSummary, error) {
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), repositoryID)
	effective, _ := store.ResolveEffectiveSettingsFull(h.global, settings)
	issueFiling := store.ShouldCreateForgeIssues(effective)
	if scanID != "" {
		return h.store.ReconciliationSummaryForScan(c.Request.Context(), repositoryID, scanID, issueFiling)
	}
	return h.store.ReconciliationSummaryForRepository(c.Request.Context(), repositoryID, issueFiling)
}