package main

import (
	"context"
	"fmt"
	"time"

	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
)

type calibrationBridge struct{}

func (calibrationBridge) Summary(c *gin.Context) (map[string]any, error) {
	if bugbotStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	return bugbotStore.CalibrationSummary(c.Request.Context())
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
	if rec.RecommendationType == "report_only" && rec.RuleID != "" {
		_, err = bugbotStore.CreateFindingSuppression(ctx, store.FindingSuppression{
			Source:    rec.Source,
			RuleID:    rec.RuleID,
			Category:  rec.Category,
			Scope:     store.SuppressionScopeGlobal,
			Reason:    rec.Reason,
			CreatedBy: "calibration-accept",
			Active:    true,
		})
		if err != nil {
			return err
		}
	}
	return bugbotStore.UpdateCalibrationRecommendationStatus(ctx, id, "accepted")
}

func (calibrationBridge) RejectRecommendation(c *gin.Context, id int64) error {
	if bugbotStore == nil {
		return fmt.Errorf("database disabled")
	}
	return bugbotStore.UpdateCalibrationRecommendationStatus(c.Request.Context(), id, "rejected")
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
	return map[string]any{
		"rules_updated":              stats,
		"recommendations_generated": recs,
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
