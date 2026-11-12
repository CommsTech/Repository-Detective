package ui

import (
	"net/http"

	"git.commsnet.org/commstech/repository-detective/reconcile"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RepoReconcilePreview(c *gin.Context) {
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
	scans, _ := h.store.ListScansByRepository(c.Request.Context(), id, storeListOpts(10))
	external, _ := h.store.ListExternalIssuesByRepository(c.Request.Context(), id, storeListOpts(50))
	data := map[string]any{
		"Repo":             repo,
		"Scans":            scans,
		"ExternalIssues":   external,
		"ReconcileEnabled": h.reconcileEnabled,
	}
	if h.reconcileEnabled && h.reconciler != nil {
		if raw, err := h.reconciler.Preview(c.Request.Context(), id); err == nil {
			if result, ok := raw.(reconcile.Result); ok {
				data["Items"] = result.Items
				data["RunID"] = result.RunID
			}
		}
	}
	h.renderNav(c, "repo_reconcile.html", "Reconcile issues", "repos", data)
}

func (h *Handler) RepoReconcileApply(c *gin.Context) {
	if !h.requireStore(c) || !h.reconcileEnabled || h.reconciler == nil {
		c.String(http.StatusServiceUnavailable, "reconciliation disabled")
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.reconciler.Apply(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	repo, _ := h.store.GetRepository(c.Request.Context(), id)
	rec, _ := result.(reconcile.Result)
	h.renderNav(c, "repo_reconcile.html", "Reconcile issues", "repos", map[string]any{
		"Repo": repo, "Items": rec.Items, "RunID": rec.RunID, "Applied": true, "ReconcileEnabled": true,
	})
}

func storeListOpts(limit int) store.ListOptions {
	return store.ListOptions{Limit: limit}
}
