package api

import (
	"net/http"
	"time"

	"git.commsnet.org/commstech/repository-detective/patcher"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// RemediationService generates and persists remediation plans.
type RemediationService interface {
	GetPlanForFinding(ctx *gin.Context, findingID int64) (remediation.Plan, error)
	GeneratePlanForFinding(ctx *gin.Context, findingID int64) (remediation.Plan, error)
	GetPlanByID(ctx *gin.Context, planID string) (remediation.Plan, error)
	ApprovePlan(ctx *gin.Context, planID string) error
	RejectPlan(ctx *gin.Context, planID string) error
}

// RemediationPRService creates safe remediation pull requests.
type RemediationPRService interface {
	CheckPREligibility(ctx *gin.Context, planID string) (patcher.EligibilityResult, error)
	AttemptPR(ctx *gin.Context, planID string) (patcher.PatchAttempt, error)
	GetPatchAttempt(ctx *gin.Context, attemptID string) (patcher.PatchAttempt, error)
	ListPatchAttemptsByPlan(ctx *gin.Context, planID string) ([]patcher.PatchAttempt, error)
}

// RemediationHandler serves remediation planning routes.
type RemediationHandler struct {
	store      store.QueryStore
	service    RemediationService
	prService  RemediationPRService
	prEnabled  bool
}

// NewRemediationHandler creates a remediation API handler.
func NewRemediationHandler(s store.QueryStore, svc RemediationService, prEnabled bool) *RemediationHandler {
	prSvc, _ := svc.(RemediationPRService)
	return &RemediationHandler{store: s, service: svc, prService: prSvc, prEnabled: prEnabled}
}

// RegisterRoutes mounts remediation routes on the authenticated API group.
func (h *RemediationHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/findings/:id/remediation", h.GetFindingRemediation)
	g.POST("/findings/:id/remediation/generate", h.GenerateFindingRemediation)
	g.GET("/remediation/:plan_id", h.GetRemediationPlan)
	g.POST("/remediation/:plan_id/approve", h.ApproveRemediationPlan)
	g.POST("/remediation/:plan_id/reject", h.RejectRemediationPlan)
	if h.prEnabled && h.prService != nil {
		g.POST("/remediation/:plan_id/attempt-pr", h.AttemptRemediationPR)
		g.GET("/remediation/:plan_id/patch-attempts", h.ListPatchAttemptsByPlan)
		g.GET("/patch-attempts/:attempt_id", h.GetPatchAttempt)
	}
}

func (h *RemediationHandler) requireService(c *gin.Context) bool {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "remediation planner disabled"})
		return false
	}
	return true
}

func (h *RemediationHandler) requirePRService(c *gin.Context) bool {
	if !h.prEnabled || h.prService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "remediation PR feature disabled"})
		return false
	}
	return true
}

func (h *RemediationHandler) GetFindingRemediation(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	plan, err := h.service.GetPlanForFinding(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRemediationPlanResponse(plan))
}

func (h *RemediationHandler) GenerateFindingRemediation(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	id, ok := parseFindingID(c)
	if !ok {
		return
	}
	plan, err := h.service.GeneratePlanForFinding(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRemediationPlanResponse(plan))
}

func (h *RemediationHandler) GetRemediationPlan(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	planID := c.Param("plan_id")
	plan, err := h.service.GetPlanByID(c, planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	resp := toRemediationPlanResponse(plan)
	if h.prEnabled && h.prService != nil {
		if elig, err := h.prService.CheckPREligibility(c, planID); err == nil {
			resp.PREligibility = &eligibilityResponse{
				Eligible:       elig.Eligible,
				BlockedReasons: elig.BlockedReasons,
				Checklist:      elig.Checklist,
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RemediationHandler) ApproveRemediationPlan(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	planID := c.Param("plan_id")
	if err := h.service.ApprovePlan(c, planID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved", "plan_id": planID})
}

func (h *RemediationHandler) RejectRemediationPlan(c *gin.Context) {
	if !h.requireStore(c) || !h.requireService(c) {
		return
	}
	planID := c.Param("plan_id")
	if err := h.service.RejectPlan(c, planID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected", "plan_id": planID})
}

func (h *RemediationHandler) AttemptRemediationPR(c *gin.Context) {
	if !h.requireStore(c) || !h.requirePRService(c) {
		return
	}
	planID := c.Param("plan_id")
	elig, err := h.prService.CheckPREligibility(c, planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if !elig.Eligible {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "plan not eligible for remediation PR",
			"blocked_reasons": elig.BlockedReasons,
			"checklist":       elig.Checklist,
		})
		return
	}
	attempt, err := h.prService.AttemptPR(c, planID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"attempt": toPatchAttemptResponse(attempt),
		})
		return
	}
	c.JSON(http.StatusOK, toPatchAttemptResponse(attempt))
}

func (h *RemediationHandler) GetPatchAttempt(c *gin.Context) {
	if !h.requireStore(c) || !h.requirePRService(c) {
		return
	}
	attemptID := c.Param("attempt_id")
	attempt, err := h.prService.GetPatchAttempt(c, attemptID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPatchAttemptResponse(attempt))
}

func (h *RemediationHandler) ListPatchAttemptsByPlan(c *gin.Context) {
	if !h.requireStore(c) || !h.requirePRService(c) {
		return
	}
	planID := c.Param("plan_id")
	attempts, err := h.prService.ListPatchAttemptsByPlan(c, planID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out := make([]patchAttemptResponse, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, toPatchAttemptResponse(a))
	}
	c.JSON(http.StatusOK, gin.H{"attempts": out})
}

func (h *RemediationHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

type eligibilityResponse struct {
	Eligible       bool            `json:"eligible"`
	BlockedReasons []string        `json:"blocked_reasons,omitempty"`
	Checklist      map[string]bool `json:"checklist,omitempty"`
}

type remediationPlanResponse struct {
	PlanID              string   `json:"plan_id"`
	FindingID           int64    `json:"finding_id,omitempty"`
	RepositoryID        int64    `json:"repository_id,omitempty"`
	AuditID             string   `json:"audit_id,omitempty"`
	Fingerprint         string   `json:"fingerprint"`
	Category            string   `json:"category"`
	Severity            string   `json:"severity"`
	Source              string   `json:"source"`
	RuleID              string   `json:"rule_id,omitempty"`
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	FixStrategy         string   `json:"fix_strategy"`
	AffectedFiles       []string `json:"affected_files"`
	RequiredTests       []string `json:"required_tests"`
	ValidationCommands  []string `json:"validation_commands"`
	RegressionRisk      string   `json:"regression_risk"`
	FixComplexity       string   `json:"fix_complexity"`
	SafeForAutoPR       bool     `json:"safe_for_auto_pr"`
	RequiresHumanReview bool     `json:"requires_human_review"`
	BlockedReasons      []string `json:"blocked_reasons,omitempty"`
	Advisory            bool     `json:"advisory"`
	Status              string   `json:"status"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
	Notice              string   `json:"notice"`
	PREligibility       *eligibilityResponse `json:"pr_eligibility,omitempty"`
}

func toRemediationPlanResponse(plan remediation.Plan) remediationPlanResponse {
	return remediationPlanResponse{
		PlanID:              plan.ID,
		FindingID:           plan.FindingID,
		RepositoryID:        plan.RepositoryID,
		AuditID:             plan.AuditID,
		Fingerprint:         plan.Fingerprint,
		Category:            plan.Category,
		Severity:            plan.Severity,
		Source:              plan.Source,
		RuleID:              plan.RuleID,
		Title:               plan.Title,
		Summary:             plan.Summary,
		FixStrategy:         plan.FixStrategy,
		AffectedFiles:       plan.AffectedFiles,
		RequiredTests:       plan.RequiredTests,
		ValidationCommands:  plan.ValidationCommands,
		RegressionRisk:      plan.RegressionRisk,
		FixComplexity:       plan.FixComplexity,
		SafeForAutoPR:       plan.SafeForAutoPR,
		RequiresHumanReview: plan.RequiresHumanReview,
		BlockedReasons:      plan.BlockedReasons,
		Advisory:            plan.Advisory,
		Status:              plan.Status,
		CreatedAt:           plan.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           plan.UpdatedAt.UTC().Format(time.RFC3339),
		Notice:              "Planning only — no code changes are made until an approved safe remediation PR is attempted.",
	}
}

type patchAttemptResponse struct {
	AttemptID         string              `json:"attempt_id"`
	PlanID            string              `json:"plan_id"`
	RepositoryID      int64               `json:"repository_id,omitempty"`
	FindingID         int64               `json:"finding_id,omitempty"`
	BranchName        string              `json:"branch_name"`
	BaseRef           string              `json:"base_ref"`
	CommitSHA         string              `json:"commit_sha"`
	Status            string              `json:"status"`
	DiffSummary       string              `json:"diff_summary,omitempty"`
	FilesChanged      []string            `json:"files_changed,omitempty"`
	TestsRun          []patcher.TestResult `json:"tests_run,omitempty"`
	ValidationSummary string              `json:"validation_summary,omitempty"`
	PullRequestNumber *int                `json:"pull_request_number,omitempty"`
	PullRequestURL    string              `json:"pull_request_url,omitempty"`
	Error             string              `json:"error,omitempty"`
	CreatedAt         string              `json:"created_at"`
	UpdatedAt         string              `json:"updated_at"`
}

func toPatchAttemptResponse(a patcher.PatchAttempt) patchAttemptResponse {
	return patchAttemptResponse{
		AttemptID:         a.ID,
		PlanID:            a.PlanID,
		RepositoryID:      a.RepositoryID,
		FindingID:         a.FindingID,
		BranchName:        a.BranchName,
		BaseRef:           a.BaseRef,
		CommitSHA:         a.CommitSHA,
		Status:            a.Status,
		DiffSummary:       a.DiffSummary,
		FilesChanged:      a.FilesChanged,
		TestsRun:          a.TestsRun,
		ValidationSummary: a.ValidationSummary,
		PullRequestNumber: a.PullRequestNumber,
		PullRequestURL:    a.PullRequestURL,
		Error:             a.Error,
		CreatedAt:         a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
