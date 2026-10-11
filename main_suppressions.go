package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/calibration"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

var suppressionMatcher *calibration.Matcher

type suppressionBridge struct{}

func initSuppressionMatcher() {
	if rdStore == nil {
		suppressionMatcher = nil
		return
	}
	suppressionMatcher = calibration.NewMatcher(rdStore)
}

func (suppressionBridge) SuppressFinding(c *gin.Context, findingID int64, req api.SuppressionRequest) (store.FindingSuppression, error) {
	return applyFindingSuppression(c.Request.Context(), findingID, req, store.FindingStatusSuppressed, store.LifecycleEventSuppressed, false)
}

func (suppressionBridge) MarkFalsePositive(c *gin.Context, findingID int64, req api.SuppressionRequest) (store.FindingSuppression, error) {
	return applyFindingSuppression(c.Request.Context(), findingID, req, store.FindingStatusFalsePositive, store.LifecycleEventFalsePositiveMarked, true)
}

func (suppressionBridge) CreateSuppression(c *gin.Context, req api.CreateSuppressionRequest) (store.FindingSuppression, error) {
	if rdStore == nil {
		return store.FindingSuppression{}, fmt.Errorf("database disabled")
	}
	sup := store.FindingSuppression{
		RepositoryID: req.RepositoryID,
		Fingerprint:  strings.TrimSpace(req.Fingerprint),
		Source:       strings.TrimSpace(req.Source),
		RuleID:       strings.TrimSpace(req.RuleID),
		Category:     strings.TrimSpace(req.Category),
		Severity:     strings.TrimSpace(req.Severity),
		Scope:        store.NormalizeSuppressionScope(req.Scope),
		Reason:       strings.TrimSpace(req.Reason),
		CreatedBy:    strings.TrimSpace(req.CreatedBy),
		ExpiresAt:    req.ExpiresAt,
		Active:       true,
	}
	created, err := rdStore.CreateFindingSuppression(c.Request.Context(), sup)
	if err != nil {
		return store.FindingSuppression{}, err
	}
	if req.RepositoryID != nil && *req.RepositoryID > 0 {
		suppressionMatcher.Invalidate(*req.RepositoryID)
	}
	return created, nil
}

func (suppressionBridge) DisableSuppression(c *gin.Context, id int64) (store.FindingSuppression, error) {
	if rdStore == nil {
		return store.FindingSuppression{}, fmt.Errorf("database disabled")
	}
	prev, err := rdStore.GetFindingSuppression(c.Request.Context(), id)
	if err != nil {
		return store.FindingSuppression{}, err
	}
	disabled, err := rdStore.DisableFindingSuppression(c.Request.Context(), id)
	if err != nil {
		return store.FindingSuppression{}, err
	}
	if prev.RepositoryID != nil {
		suppressionMatcher.Invalidate(*prev.RepositoryID)
	}
	meta, _ := json.Marshal(map[string]any{"suppression_id": id})
	if err := rdStore.AddLifecycleEvent(c.Request.Context(), store.LifecycleEvent{
		EventType:    store.LifecycleEventUnsuppressed,
		Message:      "Suppression rule disabled",
		MetadataJSON: meta,
	}); err != nil {
		logger.Warnf("suppression disable lifecycle event: %v", err)
	}
	return disabled, nil
}

func applyFindingSuppression(ctx context.Context, findingID int64, req api.SuppressionRequest, status, lifecycleEvent string, falsePositive bool) (store.FindingSuppression, error) {
	if rdStore == nil {
		return store.FindingSuppression{}, fmt.Errorf("database disabled")
	}
	detail, err := rdStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return store.FindingSuppression{}, fmt.Errorf("finding not found")
	}
	scope := store.NormalizeSuppressionScope(req.Scope)
	var repoIDPtr *int64
	if scope == store.SuppressionScopeRepo {
		repoID := detail.RepositoryID
		repoIDPtr = &repoID
	}
	sup := store.FindingSuppression{
		RepositoryID: repoIDPtr,
		Fingerprint:  detail.Fingerprint,
		Source:       detail.Source,
		RuleID:       detail.RuleID,
		Category:     detail.Category,
		Severity:     detail.Severity,
		Scope:        scope,
		Reason:       strings.TrimSpace(req.Reason),
		CreatedBy:    strings.TrimSpace(req.CreatedBy),
		ExpiresAt:    req.ExpiresAt,
		Active:       true,
	}
	created, err := rdStore.CreateFindingSuppression(ctx, sup)
	if err != nil {
		return store.FindingSuppression{}, err
	}
	if err := rdStore.UpdateFindingStatus(ctx, findingID, status); err != nil {
		return store.FindingSuppression{}, err
	}
	fid := findingID
	meta, _ := json.Marshal(map[string]any{
		"suppression_id": created.ID,
		"scope":          created.Scope,
		"reason":         created.Reason,
	})
	if err := rdStore.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID:    &fid,
		EventType:    lifecycleEvent,
		Message:      created.Reason,
		MetadataJSON: meta,
	}); err != nil {
		logger.Warnf("suppression lifecycle event for finding %d: %v", findingID, err)
	}
	if suppressionMatcher != nil {
		suppressionMatcher.Invalidate(detail.RepositoryID)
		if err := suppressionMatcher.LoadRepository(ctx, detail.RepositoryID); err != nil {
			logger.Warnf("reload suppression policy for repo %d: %v", detail.RepositoryID, err)
		}
	}
	if issueManager != nil && detail.ExternalIssueNumber > 0 {
		repo, rerr := rdStore.GetRepository(ctx, detail.RepositoryID)
		if rerr == nil {
			if err := issueManager.AnnotateCalibration(ctx, repo.ForgeType, repo.Owner, repo.Name, detail.ExternalIssueNumber, falsePositive, created.Reason); err != nil {
				logger.Warnf("annotate calibration on issue #%d: %v", detail.ExternalIssueNumber, err)
			}
		}
	}
	if falsePositive {
		emitFalsePositiveMarked(ctx, detail.RepositoryID, findingID, detail.Fingerprint, detail.Source, detail.RuleID, created.CreatedBy)
	}
	return created, nil
}

func loadSuppressionPolicy(ctx context.Context, repositoryID int64) {
	if suppressionMatcher == nil || repositoryID <= 0 {
		return
	}
	if err := suppressionMatcher.LoadRepository(ctx, repositoryID); err != nil {
		logger.Warnf("load suppression policy for repo %d: %v", repositoryID, err)
	}
}

func filterIssuesWithSuppression(repositoryID int64, issues []ai.CodeIssue) []ai.CodeIssue {
	if suppressionMatcher == nil || repositoryID <= 0 {
		return issues
	}
	return suppressionMatcher.FilterIssues(repositoryID, issues)
}

type suppressionUIBridge struct{}

func (suppressionUIBridge) SuppressFinding(ctx context.Context, findingID int64, reason, createdBy string) error {
	_, err := applyFindingSuppression(ctx, findingID, api.SuppressionRequest{Reason: reason, CreatedBy: createdBy}, store.FindingStatusSuppressed, store.LifecycleEventSuppressed, false)
	return err
}

func (suppressionUIBridge) MarkFalsePositive(ctx context.Context, findingID int64, reason, createdBy string) error {
	_, err := applyFindingSuppression(ctx, findingID, api.SuppressionRequest{Reason: reason, CreatedBy: createdBy}, store.FindingStatusFalsePositive, store.LifecycleEventFalsePositiveMarked, true)
	return err
}

func (suppressionUIBridge) SuppressRuleForRepo(ctx context.Context, findingID int64, reason, createdBy string) error {
	return suppressRepoRule(ctx, findingID, reason, createdBy, false)
}

func (suppressionUIBridge) MarkIntentionalStandalone(ctx context.Context, findingID int64, reason, createdBy string) error {
	return suppressRepoRule(ctx, findingID, reason, createdBy, true)
}

func suppressRepoRule(ctx context.Context, findingID int64, reason, createdBy string, intentional bool) error {
	if rdStore == nil {
		return fmt.Errorf("database disabled")
	}
	detail, err := rdStore.GetFindingDetail(ctx, findingID)
	if err != nil {
		return fmt.Errorf("finding not found")
	}
	if strings.TrimSpace(detail.RuleID) == "" {
		return fmt.Errorf("finding has no rule_id to suppress")
	}
	repoID := detail.RepositoryID
	sup := store.FindingSuppression{
		RepositoryID: &repoID,
		Source:       detail.Source,
		RuleID:       detail.RuleID,
		Category:     detail.Category,
		Scope:        store.SuppressionScopeRepo,
		Reason:       strings.TrimSpace(reason),
		CreatedBy:    strings.TrimSpace(createdBy),
		Active:       true,
	}
	if _, err := rdStore.CreateFindingSuppression(ctx, sup); err != nil {
		return err
	}
	status := store.FindingStatusSuppressed
	event := store.LifecycleEventSuppressed
	if intentional {
		event = store.LifecycleEventSuppressed
		if !strings.Contains(strings.ToLower(sup.Reason), "intentional") {
			sup.Reason = "intentional standalone: " + sup.Reason
		}
	}
	if err := rdStore.UpdateFindingStatus(ctx, findingID, status); err != nil {
		return err
	}
	fid := findingID
	if err := rdStore.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: &fid,
		EventType: event,
		Message:   sup.Reason,
	}); err != nil {
		logger.Warnf("standalone suppression lifecycle for finding %d: %v", findingID, err)
	}
	if suppressionMatcher != nil {
		suppressionMatcher.Invalidate(detail.RepositoryID)
		if err := suppressionMatcher.LoadRepository(ctx, detail.RepositoryID); err != nil {
			logger.Warnf("reload suppression policy for repo %d: %v", detail.RepositoryID, err)
		}
	}
	return nil
}

func isFindingSuppressedForRemediation(ctx context.Context, detail store.FindingDetail) bool {
	st := strings.ToLower(strings.TrimSpace(detail.Status))
	if st == store.FindingStatusSuppressed || st == store.FindingStatusFalsePositive {
		return true
	}
	if suppressionMatcher == nil {
		return false
	}
	in := store.FindingMatchInput{
		RepositoryID: detail.RepositoryID,
		Fingerprint:  detail.Fingerprint,
		Source:       detail.Source,
		RuleID:       detail.RuleID,
		Category:     detail.Category,
		Severity:     detail.Severity,
	}
	suppressed, _ := suppressionMatcher.IsSuppressed(detail.RepositoryID, in)
	return suppressed
}
