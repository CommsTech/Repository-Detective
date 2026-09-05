package main

import (
	"context"
	"fmt"
	"strings"
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
	_, err := acceptCalibrationRecommendation(c.Request.Context(), id)
	return err
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

func acceptCalibrationRecommendation(ctx context.Context, id int64) (int, error) {
	rec, err := findCalibrationRecommendation(ctx, id)
	if err != nil {
		return 0, err
	}
	if err := learning.ValidateCalibrationAccept(rec.Category, rec.Scope); err != nil {
		return 0, err
	}
	if rec.RecommendedAction != "report_only" || strings.TrimSpace(rec.RuleID) == "" {
		return 0, fmt.Errorf("only report_only recommendations with a rule_id can be accepted (got action=%q rule_id=%q)", rec.RecommendedAction, rec.RuleID)
	}

	repoIDs := make([]int64, 0, 1)
	if strings.EqualFold(rec.Scope, "repo") {
		if rec.RepositoryID == nil || *rec.RepositoryID <= 0 {
			return 0, fmt.Errorf("repo-scoped recommendation is missing repository_id")
		}
		repoIDs = append(repoIDs, *rec.RepositoryID)
	} else {
		// Global tiles expand into repo-scoped calibration rules — never a fleet-wide global rule.
		repoIDs, err = rdStore.ListRepositoryIDsAffectedByRule(ctx, rec.Source, rec.RuleID, 100)
		if err != nil {
			return 0, err
		}
		if len(repoIDs) == 0 {
			return 0, fmt.Errorf("no repositories currently have this rule — nothing to apply; reject the recommendation or mark false positives on a repo first")
		}
	}

	applied := 0
	for _, repoID := range repoIDs {
		repoID := repoID
		if err := applyRepoCalibrationAccept(ctx, rec, repoID); err != nil {
			return applied, fmt.Errorf("apply to repository %d: %w", repoID, err)
		}
		applied++
	}
	emitRecommendationLearning(ctx, repoIDFromRec(rec), rec.ID, true, rec.Source, rec.RuleID)
	if err := rdStore.UpdateCalibrationRecommendationStatus(ctx, id, "accepted"); err != nil {
		return applied, err
	}
	return applied, nil
}

func repoIDFromRec(rec *store.CalibrationRecommendation) int64 {
	if rec != nil && rec.RepositoryID != nil {
		return *rec.RepositoryID
	}
	return 0
}

func applyRepoCalibrationAccept(ctx context.Context, rec *store.CalibrationRecommendation, repoID int64) error {
	// Accept installs a repo-scoped calibration rule only. Findings stay visible;
	// high/critical are never downgraded (findinglearn.ApplyRepoRule). Do not create
	// rule-wide FindingSuppressions here — those bypass severity guards and hide
	// findings from forge filing/notifications.
	if alreadyHasCalibrationRule(ctx, repoID, rec.Source, rec.RuleID) {
		if suppressionMatcher != nil {
			suppressionMatcher.Invalidate(repoID)
			_ = suppressionMatcher.LoadRepository(ctx, repoID)
		}
		return nil
	}
	repoIDCopy := repoID
	expires := time.Now().UTC().Add(90 * 24 * time.Hour)
	if _, err := rdStore.CreateRepoCalibrationRule(ctx, store.RepoCalibrationRule{
		RepositoryID: &repoIDCopy, Scope: "repo", Source: rec.Source, RuleID: rec.RuleID,
		FindingCategory: rec.Category, Action: "report_only", Reason: rec.Reason,
		EvidenceCount: int(rec.Confidence * 100), FalsePositiveRate: rec.Confidence,
		Active: true, ExpiresAt: &expires, RecommendationID: &rec.ID,
	}); err != nil {
		return fmt.Errorf("create repo calibration rule: %w", err)
	}
	if suppressionMatcher != nil {
		suppressionMatcher.Invalidate(repoID)
		_ = suppressionMatcher.LoadRepository(ctx, repoID)
	}
	return nil
}

func alreadyHasCalibrationRule(ctx context.Context, repoID int64, source, ruleID string) bool {
	if rdStore == nil {
		return false
	}
	rules, err := rdStore.ListRepoCalibrationRules(ctx, repoID, true)
	if err != nil {
		return false
	}
	for _, r := range rules {
		if strings.EqualFold(r.Source, source) && strings.EqualFold(r.RuleID, ruleID) {
			return true
		}
	}
	return false
}

func rejectCalibrationRecommendation(ctx context.Context, id int64) error {
	if rdStore == nil {
		return fmt.Errorf("database disabled")
	}
	rec, err := findCalibrationRecommendation(ctx, id)
	if err != nil {
		return err
	}
	emitRecommendationLearning(ctx, repoIDFromRec(rec), id, false, rec.Source, rec.RuleID)
	return rdStore.UpdateCalibrationRecommendationStatus(ctx, id, "rejected")
}

func recomputeCalibration(ctx context.Context) (map[string]any, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	backfilled, err := rdStore.BackfillFalsePositiveLearningEvents(ctx, 10000)
	if err != nil {
		logger.Warnf("calibration backfill learning events: %v", err)
	}
	purged, err := rdStore.PurgePoisonedScannerFailureLearningEvents(ctx, nil, time.Time{})
	if err != nil {
		logger.Warnf("calibration purge poisoned scanner_failed: %v", err)
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
	repoRecs, err = recomputeRepoScopedRecommendations(ctx, config.CalibrationMinFindingsForRecommendation)
	if err != nil {
		logger.Warnf("calibration repo recommendations: %v", err)
	}
	autoApplied := 0
	if config.CalibrationAutoApply {
		autoApplied, err = autoApplySafeCalibrationRecommendations(ctx)
		if err != nil {
			logger.Warnf("calibration auto-apply: %v", err)
		}
	}
	return map[string]any{
		"learning_events_backfilled":     backfilled,
		"learning_events_purged":       purged,
		"rules_updated":                  stats,
		"recommendations_generated":      recs,
		"repo_recommendations_generated": repoRecs,
		"recommendations_auto_applied":   autoApplied,
	}, nil
}

func recomputeRepoScopedRecommendations(ctx context.Context, minFindings int) (int, error) {
	if rdStore == nil {
		return 0, nil
	}
	if minFindings <= 0 {
		minFindings = 5
	}
	total := 0
	offset := 0
	const pageSize = 100
	for {
		repos, err := rdStore.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: pageSize, Offset: offset})
		if err != nil {
			return total, err
		}
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			n, err := rdStore.GenerateRepoScopedRecommendations(ctx, r.ID, minFindings)
			if err != nil {
				logger.Warnf("repo calibration recommendations repo=%d: %v", r.ID, err)
				continue
			}
			total += n
		}
		offset += len(repos)
		if len(repos) < pageSize {
			break
		}
	}
	return total, nil
}

func autoApplySafeCalibrationRecommendations(ctx context.Context) (int, error) {
	recs, err := rdStore.ListCalibrationRecommendations(ctx, "proposed", 200)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, rec := range recs {
		if rec.RecommendedAction != "report_only" || strings.TrimSpace(rec.RuleID) == "" {
			continue
		}
		if !strings.EqualFold(rec.Scope, "repo") {
			continue
		}
		if rec.Confidence < 0.5 {
			continue
		}
		if ruleIDProtectedFromAutoApply(rec.Source, rec.RuleID) {
			continue
		}
		if err := learning.ValidateCalibrationAccept(rec.Category, rec.Scope); err != nil {
			continue
		}
		n, err := acceptCalibrationRecommendation(ctx, rec.ID)
		if err != nil {
			logger.Warnf("auto-apply recommendation %d (%s/%s): %v", rec.ID, rec.Source, rec.RuleID, err)
			continue
		}
		applied += n
	}
	return applied, nil
}

// ruleIDProtectedFromAutoApply keeps security findings out of unattended
// downgrades. It matches on the rule ID rather than the category because
// generated recommendations frequently carry an empty category, which makes
// learning.IsProtectedFromAutoDowngrade pass everything through.
func ruleIDProtectedFromAutoApply(source, ruleID string) bool {
	rule := strings.ToUpper(strings.TrimSpace(ruleID))
	rule = strings.Trim(rule, "`\"' ")
	src := strings.ToLower(strings.TrimSpace(source))

	for _, protectedSource := range []string{"gitleaks", "trivy", "grype", "govulncheck", "semgrep"} {
		if strings.Contains(src, protectedSource) {
			return true
		}
	}
	for _, prefix := range []string{"CVE-", "GHSA-", "TRIVY-", "GRYPE-", "GITLEAKS-", "SEC-", "CKV_SECRET"} {
		if strings.HasPrefix(rule, prefix) {
			return true
		}
	}
	for _, marker := range []string{"SECRET", "CREDENTIAL", "PASSWORD", "TOKEN", "PRIVATE-KEY", "PRIVATE_KEY"} {
		if strings.Contains(rule, marker) {
			return true
		}
	}
	return false
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

func (calibrationUIBridge) AcceptRecommendation(ctx context.Context, id int64) (int, error) {
	return acceptCalibrationRecommendation(ctx, id)
}

func (calibrationUIBridge) RejectRecommendation(ctx context.Context, id int64) error {
	return rejectCalibrationRecommendation(ctx, id)
}

func (calibrationUIBridge) Recompute(ctx context.Context) (map[string]any, error) {
	return recomputeCalibration(ctx)
}
