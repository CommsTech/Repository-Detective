package main

import (
	"context"
	"fmt"
	"time"

	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

type calibrationBridge struct{}

func (calibrationBridge) Summary(c *gin.Context) (map[string]any, error) {
	if bugbotStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	ctx := c.Request.Context()
	out, err := bugbotStore.CalibrationSummary(ctx)
	if err != nil {
		return nil, err
	}
	lh, _ := bugbotStore.LearningHealthSummary(ctx)
	out["learning_health"] = lh
	return out, nil
}

func (calibrationBridge) ListRecommendations(c *gin.Context, status string) ([]store.CalibrationRecommendation, error) {
	if bugbotStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	if status == "" {
		status = "proposed"
	}
	return bugbotStore.ListCalibrationRecommendations(c.Request.Context(), status, 100)
}

func (calibrationBridge) AcceptRecommendation(c *gin.Context, id int64) error {
	if bugbotStore == nil {
		return fmt.Errorf("database disabled")
	}
	ctx := c.Request.Context()
	recs, err := bugbotStore.ListCalibrationRecommendations(ctx, "", 1000)
	if err != nil {
		return err
	}
	var rec *store.CalibrationRecommendation
	for i := range recs {
		if recs[i].ID == id {
			rec = &recs[i]
			break
		}
	}
	if rec == nil {
		return fmt.Errorf("recommendation not found")
	}
	if learning.IsProtectedFromAutoDowngrade("high", rec.Category) {
		return fmt.Errorf("recommendation affects protected security category — requires explicit operator override on finding")
	}
	scope := store.SuppressionScopeGlobal
	var repoIDPtr *int64
	if rec.Scope == "repo" && rec.RepositoryID != nil && *rec.RepositoryID > 0 {
		scope = store.SuppressionScopeRepo
		repoIDPtr = rec.RepositoryID
	} else if rec.Scope == "global" {
		// Global rules require explicit multi-repo evidence — block naive global accept in beta.
		return fmt.Errorf("global calibration recommendations require multi-repo evidence review — use repo-scoped recommendations")
	}
	if rec.RecommendedAction == "report_only" && rec.RuleID != "" {
		_, err = bugbotStore.CreateFindingSuppression(ctx, store.FindingSuppression{
			RepositoryID: repoIDPtr,
			Source:       rec.Source,
			RuleID:       rec.RuleID,
			Category:     rec.Category,
			Scope:        scope,
			Reason:       rec.Reason,
			CreatedBy:    "calibration-accept",
			Active:       true,
		})
		if err != nil {
			return err
		}
		if repoIDPtr != nil {
			expires := time.Now().UTC().Add(90 * 24 * time.Hour)
			_, _ = bugbotStore.CreateRepoCalibrationRule(ctx, store.RepoCalibrationRule{
				RepositoryID: repoIDPtr, Scope: "repo", Source: rec.Source, RuleID: rec.RuleID,
				FindingCategory: rec.Category, Action: "downgrade_confidence", Reason: rec.Reason,
				EvidenceCount: int(rec.Confidence * 100), FalsePositiveRate: rec.Confidence,
				Active: true, ExpiresAt: &expires, RecommendationID: &rec.ID,
			})
		}
	}
	repoID := int64(0)
	if rec.RepositoryID != nil {
		repoID = *rec.RepositoryID
	}
	emitRecommendationLearning(ctx, repoID, rec.ID, true, rec.Source, rec.RuleID)
	return bugbotStore.UpdateCalibrationRecommendationStatus(ctx, id, "accepted")
}

func (calibrationBridge) RejectRecommendation(c *gin.Context, id int64) error {
	if bugbotStore == nil {
		return fmt.Errorf("database disabled")
	}
	ctx := c.Request.Context()
	recs, _ := bugbotStore.ListCalibrationRecommendations(ctx, "", 1000)
	for i := range recs {
		if recs[i].ID == id {
			repoID := int64(0)
			if recs[i].RepositoryID != nil {
				repoID = *recs[i].RepositoryID
			}
			emitRecommendationLearning(ctx, repoID, id, false, recs[i].Source, recs[i].RuleID)
			break
		}
	}
	return bugbotStore.UpdateCalibrationRecommendationStatus(ctx, id, "rejected")
}

func (calibrationBridge) Recompute(c *gin.Context) (map[string]any, error) {
	if bugbotStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	ctx := c.Request.Context()
	stats, err := bugbotStore.RecomputeCalibrationRuleStats(ctx)
	if err != nil {
		return nil, err
	}
	recs, err := bugbotStore.GenerateCalibrationRecommendations(ctx, config.CalibrationMinFindingsForRecommendation)
	if err != nil {
		return nil, err
	}
	repoRecs := 0
	repos, _ := bugbotStore.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 50})
	for _, r := range repos {
		n, _ := bugbotStore.GenerateRepoScopedRecommendations(ctx, r.ID, 5)
		repoRecs += n
	}
	return map[string]any{
		"rules_updated":               stats,
		"recommendations_generated":   recs,
		"repo_recommendations_generated": repoRecs,
	}, nil
}

func startCalibrationBackgroundJob() {
	if !config.CalibrationEnabled || bugbotStore == nil {
		return
	}
	interval := time.Duration(config.CalibrationIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		time.Sleep(2 * time.Minute)
		for {
			ctx, cancel := contextWithTimeout(5 * time.Minute)
			stats, err := bugbotStore.RecomputeCalibrationRuleStats(ctx)
			if err != nil {
				logger.Debugf("calibration background job: %v", err)
			} else {
				recs, _ := bugbotStore.GenerateCalibrationRecommendations(ctx, config.CalibrationMinFindingsForRecommendation)
				logger.Infof("Calibration job: updated %d rule stats, %d recommendations", stats, recs)
			}
			cancel()
			time.Sleep(interval)
		}
	}()
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
