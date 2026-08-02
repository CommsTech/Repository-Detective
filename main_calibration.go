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
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	ctx := c.Request.Context()
	out, err := rdStore.CalibrationSummary(ctx)
	if err != nil {
		return nil, err
	}
	lh, _ := rdStore.LearningHealthSummary(ctx)
	out["learning_health"] = lh
	return out, nil
}

func (calibrationBridge) ListRecommendations(c *gin.Context, status string) ([]store.CalibrationRecommendation, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	if status == "" {
		status = "proposed"
	}
	return rdStore.ListCalibrationRecommendations(c.Request.Context(), status, 100)
}

func (calibrationBridge) AcceptRecommendation(c *gin.Context, id int64) error {
	return acceptCalibrationRecommendation(c.Request.Context(), id)
}

func (calibrationBridge) RejectRecommendation(c *gin.Context, id int64) error {
	return rejectCalibrationRecommendation(c.Request.Context(), id)
}

func (calibrationBridge) Recompute(c *gin.Context) (map[string]any, error) {
	return recomputeCalibration(c.Request.Context())
}

func findCalibrationRecommendation(ctx context.Context, id int64) (*store.CalibrationRecommendation, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	recs, err := rdStore.ListCalibrationRecommendations(ctx, "", 1000)
	if err != nil {
		return nil, err
	}
	for i := range recs {
		if recs[i].ID == id {
			return &recs[i], nil
		}
	}
	return nil, fmt.Errorf("recommendation not found")
}

func acceptCalibrationRecommendation(ctx context.Context, id int64) error {
	rec, err := findCalibrationRecommendation(ctx, id)
	if err != nil {
		return err
	}
	if err := learning.ValidateCalibrationAccept(rec.Category, rec.Scope); err != nil {
		return err
	}
	scope := store.SuppressionScopeGlobal
	var repoIDPtr *int64
	if rec.Scope == "repo" && rec.RepositoryID != nil && *rec.RepositoryID > 0 {
		scope = store.SuppressionScopeRepo
		repoIDPtr = rec.RepositoryID
	}
	if rec.RecommendedAction == "report_only" && rec.RuleID != "" {
		_, err = rdStore.CreateFindingSuppression(ctx, store.FindingSuppression{
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
			_, _ = rdStore.CreateRepoCalibrationRule(ctx, store.RepoCalibrationRule{
				RepositoryID: repoIDPtr, Scope: "repo", Source: rec.Source, RuleID: rec.RuleID,
				FindingCategory: rec.Category, Action: "downgrade_confidence", Reason: rec.Reason,
				EvidenceCount: int(rec.Confidence * 100), FalsePositiveRate: rec.Confidence,
				Active: true, ExpiresAt: &expires, RecommendationID: &rec.ID,
			})
			if suppressionMatcher != nil {
				suppressionMatcher.Invalidate(*repoIDPtr)
				_ = suppressionMatcher.LoadRepository(ctx, *repoIDPtr)
			}
		}
	}
	repoID := int64(0)
	if rec.RepositoryID != nil {
		repoID = *rec.RepositoryID
	}
	emitRecommendationLearning(ctx, repoID, rec.ID, true, rec.Source, rec.RuleID)
	return rdStore.UpdateCalibrationRecommendationStatus(ctx, id, "accepted")
}

func rejectCalibrationRecommendation(ctx context.Context, id int64) error {
	if rdStore == nil {
		return fmt.Errorf("database disabled")
	}
	rec, err := findCalibrationRecommendation(ctx, id)
	if err != nil {
		return err
	}
	repoID := int64(0)
	if rec.RepositoryID != nil {
		repoID = *rec.RepositoryID
	}
	emitRecommendationLearning(ctx, repoID, id, false, rec.Source, rec.RuleID)
	return rdStore.UpdateCalibrationRecommendationStatus(ctx, id, "rejected")
}

func recomputeCalibration(ctx context.Context) (map[string]any, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	stats, err := rdStore.RecomputeCalibrationRuleStats(ctx)
	if err != nil {
		return nil, err
	}
	recs, err := rdStore.GenerateCalibrationRecommendations(ctx, config.CalibrationMinFindingsForRecommendation)
	if err != nil {
		return nil, err
	}
	repoRecs := 0
	repos, _ := rdStore.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 50})
	for _, r := range repos {
		n, _ := rdStore.GenerateRepoScopedRecommendations(ctx, r.ID, 5)
		repoRecs += n
	}
	return map[string]any{
		"rules_updated":                  stats,
		"recommendations_generated":      recs,
		"repo_recommendations_generated": repoRecs,
	}, nil
}

func startCalibrationBackgroundJob() {
	if !config.CalibrationEnabled || rdStore == nil {
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
			out, err := recomputeCalibration(ctx)
			if err != nil {
				logger.Debugf("calibration background job: %v", err)
			} else {
				logger.Infof("Calibration job: updated %v rule stats, %v global recommendations, %v repo recommendations",
					out["rules_updated"], out["recommendations_generated"], out["repo_recommendations_generated"])
			}
			cancel()
			time.Sleep(interval)
		}
	}()
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// calibrationUIBridge exposes accept/reject/recompute to the operator UI.
type calibrationUIBridge struct{}

func (calibrationUIBridge) AcceptRecommendation(ctx context.Context, id int64) error {
	return acceptCalibrationRecommendation(ctx, id)
}

func (calibrationUIBridge) RejectRecommendation(ctx context.Context, id int64) error {
	return rejectCalibrationRecommendation(ctx, id)
}

func (calibrationUIBridge) Recompute(ctx context.Context) (map[string]any, error) {
	return recomputeCalibration(ctx)
}
