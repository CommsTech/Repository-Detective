package ui

import (
	"fmt"
	"net/http"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerScanRoutes(g *gin.RouterGroup) {
	g.GET("/repos/:id/scan", h.RepoScanForm)
	g.POST("/repos/:id/scan", h.RepoScanStart)
}

// RepoScanForm shows the manual scan confirmation form (optional deep link).
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
	h.renderNav(c, "scan_now.html", "Scan — "+repo.FullName, "repos", map[string]any{
		"ScanForm": h.buildScanFormView(repo, effective, meta),
	})
}

// RepoScanStart queues a manual scan. JSON clients receive scan metadata; HTML forms redirect to repo detail.
func (h *Handler) RepoScanStart(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	if !h.ScanTriggerEnabled() {
		respondScanStartError(c, http.StatusServiceUnavailable, "manual scan is not available")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		respondScanStartError(c, http.StatusNotFound, "repository not found")
		return
	}
	settings, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	effective, _ := store.ResolveEffectiveSettingsFull(h.global, settings)

	ref := strings.TrimSpace(c.PostForm("ref"))
	if ref == "" {
		ref = strings.TrimSpace(repo.DefaultBranch)
	}
	if ref == "" {
		ref = "main"
	}
	profile := strings.TrimSpace(c.PostForm("scan_profile"))
	requestDryRun := c.PostForm("report_only_dry_run") == "on" || c.PostForm("report_only_dry_run") == "true"
	filing := store.ResolveScanFilingPolicy(store.ScanFilingInput{
		Kind:                  store.ScanKindManual,
		Effective:             effective,
		RequestDryRun:         requestDryRun,
		BacklogControlEnabled: h.platform.BacklogControlEnabled,
		MaxIssuesPerScan:      h.platform.MaxIssuesPerScan,
	})
	reportOnly := filing.ReportOnlyDryRun

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
		respondScanStartError(c, http.StatusInternalServerError, fmt.Sprintf("failed to start scan: %v", err))
		return
	}

	q := ""
	if key := displayAPIKey(c); key != "" {
		q = apiKeyQueryString(key)
	}
	scanURL := fmt.Sprintf("%s/scans/%s%s", h.basePath, result.ScanID, q)
	repoURL := fmt.Sprintf("%s/repos/%d%s", h.basePath, id, q)

	if wantsJSONScanResponse(c) {
		c.JSON(http.StatusOK, gin.H{
			"status":              "analysis started",
			"scan_id":             result.ScanID,
			"trigger_type":        store.TriggerManual,
			"report_only_dry_run": reportOnly,
			"issue_filing":        filing.WillFileIssues,
			"scan_policy_mode":    filing.Mode,
			"scan_url":            scanURL,
			"repo_url":            repoURL,
			"redirect":            scanURL,
		})
		return
	}

	redirectURL := scanURL
	if strings.Contains(redirectURL, "?") {
		redirectURL += "&"
	} else {
		redirectURL += "?"
	}
	redirectURL += "started=1"
	c.Redirect(http.StatusSeeOther, redirectURL)
}

func wantsJSONScanResponse(c *gin.Context) bool {
	if strings.EqualFold(strings.TrimSpace(c.PostForm("format")), "json") {
		return true
	}
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Requested-With"), "XMLHttpRequest")
}

func respondScanStartError(c *gin.Context, code int, msg string) {
	if wantsJSONScanResponse(c) {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.String(code, msg)
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
