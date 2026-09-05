package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

var remediationPlanner *remediation.Planner

type remediationBridge struct{}

func initRemediationPlanner() {
	cfg := remediation.Config{
		Enabled:         config.RemediationPlannerEnabled,
		MinSeverity:     config.RemediationMinSeverity,
		MinConfidence:   config.RemediationMinConfidence,
		UseAI:           config.RemediationUseAI,
		CommentOnIssue:  config.RemediationCommentOnIssue,
		GlobalAIAllowed: aiClient != nil && config.EnableLLMAuditors,
	}
	var aiAdvisor remediation.AIAdvisor
	if cfg.UseAI && cfg.GlobalAIAllowed {
		aiAdvisor = remediation.StubAIAdvisor{}
	}
	remediationPlanner = remediation.NewPlanner(cfg, aiAdvisor, nil)
}

func (remediationBridge) GetPlanForFinding(c *gin.Context, findingID int64) (remediation.Plan, error) {
	return getLatestPlan(c.Request.Context(), findingID)
}

type remediationUIBridge struct{}

func (remediationUIBridge) GeneratePlan(ctx context.Context, findingID int64) (remediation.Plan, error) {
	return generateRemediationPlan(ctx, findingID)
}

func (remediationUIBridge) ApprovePlan(ctx context.Context, planID string) error {
	return updatePlanStatus(ctx, planID, store.RemediationStatusApproved)
}

func (remediationUIBridge) RejectPlan(ctx context.Context, planID string) error {
	return updatePlanStatus(ctx, planID, store.RemediationStatusRejected)
}

func (remediationBridge) GeneratePlanForFinding(c *gin.Context, findingID int64) (remediation.Plan, error) {
	return generateRemediationPlan(c.Request.Context(), findingID)
}

func (remediationBridge) GetPlanByID(c *gin.Context, planID string) (remediation.Plan, error) {
	if rdStore == nil {
		return remediation.Plan{}, fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetRemediationPlanByPlanID(c.Request.Context(), planID)
	if err != nil {
		return remediation.Plan{}, err
	}
	return store.RemediationPlanToDomain(rec), nil
}

func (remediationBridge) ApprovePlan(c *gin.Context, planID string) error {
	return updatePlanStatus(c.Request.Context(), planID, store.RemediationStatusApproved)
}

func (remediationBridge) RejectPlan(c *gin.Context, planID string) error {
	return updatePlanStatus(c.Request.Context(), planID, store.RemediationStatusRejected)
}

func getLatestPlan(ctx context.Context, findingID int64) (remediation.Plan, error) {
	if rdStore == nil {
		return remediation.Plan{}, fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetLatestRemediationPlanByFindingID(ctx, findingID)
	if err != nil {
		return remediation.Plan{}, fmt.Errorf("no remediation plan found")
	}
	return store.RemediationPlanToDomain(rec), nil
}

func generateRemediationPlan(ctx context.Context, findingID int64) (remediation.Plan, error) {
	if remediationPlanner == nil || !config.RemediationPlannerEnabled {
		return remediation.Plan{}, fmt.Errorf("remediation planner disabled")
	}
	if rdStore == nil {
		return remediation.Plan{}, fmt.Errorf("database disabled")
	}
	detail, err := rdStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return remediation.Plan{}, fmt.Errorf("finding not found")
	}
	repo, err := rdStore.GetRepository(ctx, detail.RepositoryID)
	if err != nil {
		return remediation.Plan{}, fmt.Errorf("repository not found")
	}
	if isFindingSuppressedForRemediation(ctx, detail) {
		return remediation.Plan{}, fmt.Errorf("finding is suppressed or marked false positive")
	}
	fctx := findingContextFromDetail(detail, repo)
	if !remediationPlanner.ShouldPlan(fctx) {
		return remediation.Plan{}, fmt.Errorf("finding not eligible for remediation planning")
	}
	if err := rdStore.SupersedeRemediationPlansForFinding(ctx, findingID); err != nil {
		logger.Warnf("supersede remediation plans for finding %d: %v", findingID, err)
	}
	plan, err := remediationPlanner.Generate(ctx, fctx)
	if err != nil {
		return remediation.Plan{}, err
	}
	rec, err := rdStore.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(plan))
	if err != nil {
		return remediation.Plan{}, err
	}
	plan = store.RemediationPlanToDomain(rec)
	addRemediationLifecycle(ctx, findingID, plan.ID, "remediation_plan_created", "Remediation plan generated")
	if config.RemediationCommentOnIssue {
		maybeCommentRemediationPlan(ctx, detail, repo, plan)
	}
	return plan, nil
}

func updatePlanStatus(ctx context.Context, planID, status string) error {
	if rdStore == nil {
		return fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetRemediationPlanByPlanID(ctx, planID)
	if err != nil {
		return err
	}
	if err := rdStore.UpdateRemediationPlanStatus(ctx, planID, status); err != nil {
		return err
	}
	if rec.FindingID != nil {
		addRemediationLifecycle(ctx, *rec.FindingID, planID, "remediation_plan_"+status, "Remediation plan status updated to "+status)
	}
	return nil
}

func maybeGenerateRemediationPlans(ctx context.Context, repositoryID int64, codeIssues []ai.CodeIssue, processed []issues.ProcessedIssueRecord) {
	if remediationPlanner == nil || !config.RemediationPlannerEnabled || rdStore == nil {
		return
	}
	repo, err := rdStore.GetRepository(ctx, repositoryID)
	if err != nil || !repo.ConnectedRepo {
		return
	}
	processedFP := map[string]issues.ProcessedIssueRecord{}
	for _, p := range processed {
		if p.Fingerprint != "" {
			processedFP[p.Fingerprint] = p
		}
	}
	for _, issue := range codeIssues {
		if issue.Fingerprint == "" {
			continue
		}
		pitem, ok := processedFP[issue.Fingerprint]
		if !ok || pitem.IssueNumber == 0 {
			continue
		}
		finding, err := rdStore.GetFindingByFingerprint(ctx, repositoryID, issue.Fingerprint)
		if err != nil {
			continue
		}
		if isFindingSuppressedForRemediation(ctx, store.FindingDetail{FindingListItem: store.FindingListItem{Finding: finding}}) {
			continue
		}
		fctx := remediation.FindingContext{
			FindingID:     finding.ID,
			RepositoryID:  repositoryID,
			Fingerprint:   finding.Fingerprint,
			Category:      finding.Category,
			Severity:      finding.Severity,
			Source:        finding.Source,
			RuleID:        finding.RuleID,
			Title:         finding.Title,
			Summary:       issue.Description,
			Confidence:    finding.Confidence,
			FilePath:      finding.FilePath,
			Line:          finding.Line,
			PackageName:   finding.PackageName,
			FromAI:        issue.FromAI,
			ConnectedRepo: repo.ConnectedRepo,
			RepoFullName:  repo.FullName,
		}
		if !remediationPlanner.ShouldPlan(fctx) {
			continue
		}
		if err := rdStore.SupersedeRemediationPlansForFinding(ctx, finding.ID); err != nil {
			logger.Warnf("supersede remediation plans for finding %d: %v", finding.ID, err)
		}
		plan, err := remediationPlanner.Generate(ctx, fctx)
		if err != nil {
			logger.Warnf("remediation plan for finding %d: %v", finding.ID, err)
			continue
		}
		rec, err := rdStore.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(plan))
		if err != nil {
			logger.Warnf("save remediation plan: %v", err)
			continue
		}
		plan = store.RemediationPlanToDomain(rec)
		addRemediationLifecycle(ctx, finding.ID, plan.ID, "remediation_plan_created", "Remediation plan generated after issue update")
		if config.RemediationCommentOnIssue {
			detail, derr := rdStore.GetFindingDetail(ctx, finding.ID)
			if derr == nil {
				maybeCommentRemediationPlan(ctx, detail, repo, plan)
			}
		}
	}
}

func findingContextFromDetail(detail store.FindingDetail, repo store.Repository) remediation.FindingContext {
	summary := detail.Title
	if len(detail.Instances) > 0 && detail.Instances[0].EvidenceRedacted != "" {
		summary = detail.Instances[0].EvidenceRedacted
	}
	fromAI := false
	if len(detail.Instances) > 0 && len(detail.Instances[0].RawMetadataJSON) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(detail.Instances[0].RawMetadataJSON, &meta); err == nil {
			fromAI, _ = meta["from_ai"].(bool)
		}
	}
	if !issues.IsAIAuditorSource(detail.Source) {
		fromAI = false
	}
	return remediation.FindingContext{
		FindingID:     detail.ID,
		RepositoryID:  detail.RepositoryID,
		Fingerprint:   detail.Fingerprint,
		Category:      detail.Category,
		Severity:      detail.Severity,
		Source:        detail.Source,
		RuleID:        detail.RuleID,
		Title:         detail.Title,
		Summary:       summary,
		Confidence:    detail.Confidence,
		FilePath:      detail.FilePath,
		Line:          detail.Line,
		PackageName:   detail.PackageName,
		FromAI:        fromAI,
		ConnectedRepo: repo.ConnectedRepo,
		RepoFullName:  repo.FullName,
	}
}

func addRemediationLifecycle(ctx context.Context, findingID int64, planID, eventType, message string) {
	if rdStore == nil {
		return
	}
	fid := findingID
	if err := rdStore.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID:    &fid,
		EventType:    eventType,
		Message:      message,
		MetadataJSON: remediationLifecycleMeta(planID),
	}); err != nil {
		logger.Warnf("remediation lifecycle event %s for finding %d: %v", eventType, findingID, err)
	}
}

func remediationLifecycleMeta(planID string) []byte {
	raw, _ := json.Marshal(map[string]any{"plan_id": planID})
	return raw
}

func maybeCommentRemediationPlan(ctx context.Context, detail store.FindingDetail, repo store.Repository, plan remediation.Plan) {
	if giteaClient == nil || len(detail.ExternalIssues) == 0 {
		return
	}
	ext := detail.ExternalIssues[0]
	if ext.IssueNumber <= 0 {
		return
	}
	parts := strings.SplitN(repo.FullName, "/", 2)
	if len(parts) != 2 {
		return
	}
	body := remediation.RenderIssueComment(plan)
	if err := giteaClient.CreateIssueComment(ctx, parts[0], parts[1], ext.IssueNumber, body); err != nil {
		logger.Warnf("remediation issue comment failed: %v", err)
	}
}
