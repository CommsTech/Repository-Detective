package ui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerRepoControlRoutes(g *gin.RouterGroup) {
	g.POST("/repos/:id/enable-scanning", h.RepoEnableScanning)
	g.POST("/repos/:id/disable-scanning", h.RepoDisableScanning)
}

func (h *Handler) RepoEnableScanning(c *gin.Context) {
	h.setRepoScanEnabled(c, true)
}

func (h *Handler) RepoDisableScanning(c *gin.Context) {
	h.setRepoScanEnabled(c, false)
}

func (h *Handler) setRepoScanEnabled(c *gin.Context, enabled bool) {
	if !h.requireStore(c) || !h.requireCSRF(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	repo, err := h.store.GetRepository(c.Request.Context(), id)
	if err != nil {
		respondRepoToggleError(c, http.StatusNotFound, "repository not found")
		return
	}
	existing, _ := h.store.GetRepoSettings(c.Request.Context(), id)
	existing.RepositoryID = id
	existing.UpdatedAt = time.Now().UTC()
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, store.SettingsUpdate{Enabled: &enabled})
	if err := store.ValidateRepoSettings(merged); err != nil {
		respondRepoToggleError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SaveRepoSettings(c.Request.Context(), merged); err != nil {
		respondRepoToggleError(c, http.StatusInternalServerError, "failed to save settings")
		return
	}
	eventType := "repo_scan_disabled"
	if enabled {
		eventType = "repo_scan_enabled"
	}
	_, _ = h.store.RecordLearningEvent(c.Request.Context(), store.LearningEvent{
		RepositoryID: id,
		EventType:    eventType,
		Source:       "operator_ui",
		CreatedBy:    "ui/repos",
		EvidenceJSON: []byte(fmt.Sprintf(`{"enabled":%t}`, enabled)),
		IdempotencyKey: fmt.Sprintf("%d:%s:%d", id, eventType, time.Now().UnixNano()),
	})

	if wantsJSONScanResponse(c) || strings.EqualFold(strings.TrimSpace(c.PostForm("format")), "json") {
		effective, meta := store.ResolveEffectiveSettingsFull(h.global, merged)
		c.JSON(http.StatusOK, gin.H{
			"repository_id": id,
			"enabled":       effective.Enabled,
			"scan_profile":  meta.ScanProfile,
			"full_name":     repo.FullName,
		})
		return
	}
	redirect := fmt.Sprintf("%s/repos%s", h.basePath, apiKeyQueryString(h.apiKeyFromContext(c)))
	if strings.Contains(redirect, "?") {
		redirect += "&"
	} else {
		redirect += "?"
	}
	if enabled {
		msg := fmt.Sprintf("Scanning enabled for %s", repo.FullName)
		if setup := h.webhookSetupStatus(); !setup.Ready {
			msg += ". Push webhooks are not fully configured yet — manual Scan now works; fix PUBLIC_URL, webhook secret, and Gitea registration (see Setup guide / Onboard)."
		}
		redirect += "notice=" + url.QueryEscape(msg)
	} else {
		redirect += "notice=" + url.QueryEscape(fmt.Sprintf("Scanning disabled for %s", repo.FullName))
	}
	c.Redirect(http.StatusSeeOther, redirect)
}

func respondRepoToggleError(c *gin.Context, code int, msg string) {
	if wantsJSONScanResponse(c) || strings.EqualFold(strings.TrimSpace(c.PostForm("format")), "json") {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.String(code, msg)
}
