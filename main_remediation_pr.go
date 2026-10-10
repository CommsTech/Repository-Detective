package main

import (
	"context"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/patcher"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func remediationPRConfig() patcher.Config {
	prefix := config.RemediationPRBranchPrefix
	if prefix == "" {
		prefix = "repository-detective/fix"
	}
	maxFiles := config.RemediationPRMaxFilesChanged
	if maxFiles <= 0 {
		maxFiles = 3
	}
	maxLines := config.RemediationPRMaxDiffLines
	if maxLines <= 0 {
		maxLines = 100
	}
	timeout := config.RemediationPRValidationTimeoutSeconds
	if timeout <= 0 {
		timeout = 300
	}
	return patcher.Config{
		Enabled:                          config.RemediationPREnabled,
		BranchPrefix:                     prefix,
		RequireApproval:                  config.RemediationPRRequireApproval,
		MaxFilesChanged:                  maxFiles,
		MaxDiffLines:                     maxLines,
		ValidationTimeoutSec:             timeout,
		RequireTests:                     config.RemediationPRRequireTests,
		UseRunnerVerification:            config.RemediationPRUseRunnerVerification,
		BlockHighCriticalWithoutOverride: config.RemediationPRBlockHighCritical,
		AllowedSeverities:                config.RemediationPRAllowedSeverities,
	}
}

func (remediationBridge) CheckPREligibility(c *gin.Context, planID string) (patcher.EligibilityResult, error) {
	plan, repo, err := loadPlanAndRepo(c.Request.Context(), planID)
	if err != nil {
		return patcher.EligibilityResult{}, err
	}
	return patcher.CheckPREligibility(plan, repoContextFromStore(repo), remediationPRConfig()), nil
}

func (remediationBridge) AttemptPR(c *gin.Context, planID string) (patcher.PatchAttempt, error) {
	return attemptRemediationPR(c.Request.Context(), planID)
}

func (remediationBridge) GetPatchAttempt(c *gin.Context, attemptID string) (patcher.PatchAttempt, error) {
	if rdStore == nil {
		return patcher.PatchAttempt{}, fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetPatchAttemptByAttemptID(c.Request.Context(), attemptID)
	if err != nil {
		return patcher.PatchAttempt{}, err
	}
	return store.PatchAttemptToDomain(rec), nil
}

func (remediationBridge) ListPatchAttemptsByPlan(c *gin.Context, planID string) ([]patcher.PatchAttempt, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	recs, err := rdStore.ListPatchAttemptsByPlanID(c.Request.Context(), planID)
	if err != nil {
		return nil, err
	}
	out := make([]patcher.PatchAttempt, 0, len(recs))
	for _, rec := range recs {
		out = append(out, store.PatchAttemptToDomain(rec))
	}
	return out, nil
}

func loadPlanAndRepo(ctx context.Context, planID string) (remediation.Plan, store.Repository, error) {
	if rdStore == nil {
		return remediation.Plan{}, store.Repository{}, fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetRemediationPlanByPlanID(ctx, planID)
	if err != nil {
		return remediation.Plan{}, store.Repository{}, err
	}
	plan := store.RemediationPlanToDomain(rec)
	if rec.RepositoryID == nil {
		return remediation.Plan{}, store.Repository{}, fmt.Errorf("plan has no repository")
	}
	repo, err := rdStore.GetRepository(ctx, *rec.RepositoryID)
	if err != nil {
		return remediation.Plan{}, store.Repository{}, err
	}
	if repo.DefaultBranch == "" && giteaClient != nil {
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) == 2 {
			if gr, gerr := giteaClient.GetRepository(ctx, parts[0], parts[1]); gerr == nil && gr.DefaultBranch != "" {
				repo.DefaultBranch = gr.DefaultBranch
			}
		}
	}
	return plan, repo, nil
}

func attemptRemediationPR(ctx context.Context, planID string) (patcher.PatchAttempt, error) {
	cfg := remediationPRConfig()
	if !cfg.Enabled {
		return patcher.PatchAttempt{}, fmt.Errorf("remediation PR feature disabled")
	}
	if rdStore == nil {
		return patcher.PatchAttempt{}, fmt.Errorf("database disabled")
	}
	if giteaClient == nil {
		return patcher.PatchAttempt{}, fmt.Errorf("gitea client unavailable")
	}

	plan, repo, err := loadPlanAndRepo(ctx, planID)
	if err != nil {
		return patcher.PatchAttempt{}, err
	}
	if plan.FindingID > 0 && plan.TargetLine == 0 {
		if detail, derr := rdStore.GetFindingDetail(ctx, plan.FindingID); derr == nil && detail.Line > 0 {
			plan.TargetLine = detail.Line
		}
	}
	repoCtx := repoContextFromStore(repo)
	eligibility := patcher.CheckPREligibility(plan, repoCtx, cfg)
	if !eligibility.Eligible {
		return patcher.PatchAttempt{}, fmt.Errorf("not eligible: %s", strings.Join(eligibility.BlockedReasons, "; "))
	}

	issueNumber := 0
	if plan.FindingID > 0 {
		if issues, ierr := rdStore.ListExternalIssuesByFinding(ctx, plan.FindingID); ierr == nil && len(issues) > 0 {
			issueNumber = issues[0].IssueNumber
		}
	}

	exec := patcher.Executor{
		Config:      cfg,
		Gitea:       giteaClient,
		GiteaToken:  config.GiteaToken,
		CloneURL:    repo.CloneURL,
		IssueNumber: issueNumber,
	}
	attempt, err := exec.Run(ctx, patcher.AttemptInput{Plan: plan, Repo: repoCtx})
	rec := store.PatchAttemptFromDomain(attempt)
	if _, saveErr := rdStore.SavePatchAttempt(ctx, rec); saveErr != nil {
		logger.Warnf("save patch attempt: %v", saveErr)
	}
	if err != nil {
		return attempt, err
	}
	if plan.FindingID > 0 {
		addRemediationLifecycle(ctx, plan.FindingID, planID, "remediation_pr_opened", "Remediation PR opened: "+attempt.PullRequestURL)
		if issues, ierr := rdStore.ListExternalIssuesByFinding(ctx, plan.FindingID); ierr == nil && len(issues) > 0 {
			parts := strings.SplitN(repo.FullName, "/", 2)
			if len(parts) == 2 {
				markFixPROpened(ctx, parts[0], parts[1], issues[0].IssueNumber)
			}
		}
	}
	return attempt, nil
}

func repoContextFromStore(repo store.Repository) patcher.RepoContext {
	return patcher.RepoContext{
		Owner:         repo.Owner,
		Name:          repo.Name,
		FullName:      repo.FullName,
		CloneURL:      repo.CloneURL,
		DefaultBranch: repo.DefaultBranch,
		ConnectedRepo: repo.ConnectedRepo,
	}
}

type remediationPRUIBridge struct{}

func (remediationPRUIBridge) CheckPREligibility(ctx context.Context, planID string) (patcher.EligibilityResult, error) {
	plan, repo, err := loadPlanAndRepo(ctx, planID)
	if err != nil {
		return patcher.EligibilityResult{}, err
	}
	return patcher.CheckPREligibility(plan, repoContextFromStore(repo), remediationPRConfig()), nil
}

func (remediationPRUIBridge) AttemptPR(ctx context.Context, planID string) (patcher.PatchAttempt, error) {
	return attemptRemediationPR(ctx, planID)
}

func (remediationPRUIBridge) ListPatchAttempts(ctx context.Context, planID string) ([]patcher.PatchAttempt, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	recs, err := rdStore.ListPatchAttemptsByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}
	out := make([]patcher.PatchAttempt, 0, len(recs))
	for _, rec := range recs {
		out = append(out, store.PatchAttemptToDomain(rec))
	}
	return out, nil
}
