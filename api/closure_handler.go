package api

import (
	"net/http"
	"time"

	"git.commsnet.org/commstech/repository-detective/closure"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// ClosureService provides evidence-based closure operations.
type ClosureService interface {
	GetClosureEvidence(c *gin.Context, findingID int64) (closure.Evidence, error)
	VerifyClosure(c *gin.Context, findingID int64) (closure.Evidence, error)
	RecordDirectRemediation(c *gin.Context, findingID int64, mergeCommitSHA, reason string) (closure.Evidence, error)
	CheckPatchAttemptMerge(c *gin.Context, attemptID string) (closure.Evidence, error)
}

// ClosureHandler serves closure evidence routes.
type ClosureHandler struct {
	store   store.QueryStore
	service ClosureService
}

// NewClosureHandler creates a closure API handler.
func NewClosureHandler(s store.QueryStore, svc ClosureService) *ClosureHandler {
	return &ClosureHandler{store: s, service: svc}
}

// RegisterRoutes mounts closure routes on the authenticated API group.
func (h *ClosureHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/findings/:id/closure-evidence", h.GetClosureEvidence)
	g.POST("/findings/:id/verify-closure", h.VerifyClosure)
	g.POST("/findings/:id/record-direct-remediation", h.RecordDirectRemediation)
	g.POST("/patch-attempts/:attempt_id/check-merge", h.CheckPatchAttemptMerge)
}

func (h *ClosureHandler) GetClosureEvidence(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	ev, err := h.service.GetClosureEvidence(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toClosureEvidenceResponse(ev))
}

func (h *ClosureHandler) VerifyClosure(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	ev, err := h.service.VerifyClosure(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toClosureEvidenceResponse(ev))
}

type recordDirectRemediationRequest struct {
	MergeCommitSHA string `json:"merge_commit_sha" binding:"required"`
	Reason         string `json:"reason"`
}

func (h *ClosureHandler) RecordDirectRemediation(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	var req recordDirectRemediationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merge_commit_sha is required"})
		return
	}
	ev, err := h.service.RecordDirectRemediation(c, id, req.MergeCommitSHA, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toClosureEvidenceResponse(ev))
}

func (h *ClosureHandler) CheckPatchAttemptMerge(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	attemptID := c.Param("attempt_id")
	ev, err := h.service.CheckPatchAttemptMerge(c, attemptID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toClosureEvidenceResponse(ev))
}

func (h *ClosureHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

type closureEvidenceResponse struct {
	FindingID          int64    `json:"finding_id"`
	PatchAttemptID     string   `json:"patch_attempt_id,omitempty"`
	Fingerprint        string   `json:"fingerprint,omitempty"`
	MergeCommitSHA     string   `json:"merge_commit_sha,omitempty"`
	VerificationScanID string   `json:"verification_scan_id,omitempty"`
	OriginalSource     string   `json:"original_source,omitempty"`
	ScannerStatus      string   `json:"scanner_status,omitempty"`
	FingerprintPresent bool     `json:"fingerprint_present"`
	Status             string   `json:"status"`
	Reason             string   `json:"reason,omitempty"`
	Blockers           []string `json:"blockers,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
	Notice             string   `json:"notice"`
}

func toClosureEvidenceResponse(ev closure.Evidence) closureEvidenceResponse {
	blockers := ev.Blockers
	if ev.Status == closure.StatusPendingRescan && len(blockers) == 0 {
		blockers = []string{"waiting for PR merge and/or verification scan"}
	}
	resp := closureEvidenceResponse{
		FindingID:          ev.FindingID,
		PatchAttemptID:     ev.PatchAttemptID,
		Fingerprint:        ev.Fingerprint,
		MergeCommitSHA:     ev.MergeCommitSHA,
		VerificationScanID: ev.VerificationScanID,
		OriginalSource:     ev.OriginalSource,
		ScannerStatus:      ev.ScannerStatus,
		FingerprintPresent: ev.FingerprintPresent,
		Status:             ev.Status,
		Reason:             ev.Reason,
		Blockers:           blockers,
		Notice:             "Issues are not closed because a PR exists — only after merge + rescan verification.",
	}
	if !ev.CreatedAt.IsZero() {
		resp.CreatedAt = ev.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !ev.UpdatedAt.IsZero() {
		resp.UpdatedAt = ev.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return resp
}
