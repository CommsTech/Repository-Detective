package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/learning"
	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

var openclawReviewService *openclaw.Service

type openclawReviewBridge struct{}

func (openclawReviewBridge) Config() openclaw.Config {
	return config.OpenClawAIReview.Normalized()
}

func (openclawReviewBridge) RunReview(c *gin.Context, scanID string) (openclaw.ReviewResult, error) {
	// Detach from the HTTP request context: clients (and reverse proxies) often
	// disconnect before OpenClaw agent turns finish (~45–180s+). Cancelling the
	// outbound chat call left reviews stuck in status=running forever.
	cfg := config.OpenClawAIReview.Normalized()
	timeout := time.Duration(cfg.TimeoutSeconds+30) * time.Second
	if timeout < 120*time.Second {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), timeout)
	defer cancel()
	return runOpenClawReview(ctx, scanID)
}

func (openclawReviewBridge) GetReview(c *gin.Context, scanID string) (store.AIAdvisoryReview, []store.AIAdvisoryRecommendation, error) {
	ctx := c.Request.Context()
	review, err := rdStore.GetAIAdvisoryReviewByScanID(ctx, scanID)
	if err != nil {
		return store.AIAdvisoryReview{}, nil, err
	}
	recs, err := rdStore.ListAIAdvisoryRecommendations(ctx, review.ReviewID)
	return review, recs, err
}

func (openclawReviewBridge) AcceptRecommendation(c *gin.Context, id int64) error {
	cfg := config.OpenClawAIReview.Normalized()
	if !cfg.RequireOperatorApproval {
		return fmt.Errorf("operator approval required")
	}
	return acceptAIAdvisoryRecommendation(c.Request.Context(), id)
}

func (openclawReviewBridge) RejectRecommendation(c *gin.Context, id int64) error {
	ctx := c.Request.Context()
	rec, err := rdStore.GetAIAdvisoryRecommendationByID(ctx, id)
	if err != nil {
		return err
	}
	review, _ := rdStore.GetAIAdvisoryReview(ctx, rec.ReviewID)
	source, ruleID, _, _ := rdStore.FindingSourceRuleForFingerprint(ctx, review.RepositoryID, rec.FindingFingerprint)
	emitLearning(ctx, store.LearningEvent{
		RepositoryID:   review.RepositoryID,
		ScanID:         review.ScanID,
		Fingerprint:    rec.FindingFingerprint,
		Source:         source,
		RuleID:         ruleID,
		EventType:      learning.EventRecommendationRejected,
		CreatedBy:      "ai-advisory",
		IdempotencyKey: fmt.Sprintf("ai-reject:%d", id),
	})
	return rdStore.UpdateAIAdvisoryRecommendationStatus(ctx, id, "rejected")
}

// acceptAIAdvisoryRecommendation records learning and, for calibrate_repo_scope on
// non-protected noise, installs a repo report_only calibration rule.
func acceptAIAdvisoryRecommendation(ctx context.Context, id int64) error {
	if rdStore == nil {
		return fmt.Errorf("database disabled")
	}
	rec, err := rdStore.GetAIAdvisoryRecommendationByID(ctx, id)
	if err != nil {
		return err
	}
	review, err := rdStore.GetAIAdvisoryReview(ctx, rec.ReviewID)
	if err != nil {
		return err
	}
	source, ruleID, category, _ := rdStore.FindingSourceRuleForFingerprint(ctx, review.RepositoryID, rec.FindingFingerprint)
	action := strings.ToLower(strings.TrimSpace(rec.SuggestedAction))

	switch action {
	case "calibrate_repo_scope", "leave_visible":
		emitLearningEvidence(ctx, store.LearningEvent{
			RepositoryID:   review.RepositoryID,
			ScanID:         review.ScanID,
			Fingerprint:    rec.FindingFingerprint,
			Source:         source,
			RuleID:         ruleID,
			EventType:      learning.EventAIAdvisoryCalibrateSuggested,
			CreatedBy:      "ai-advisory-accept",
			IdempotencyKey: fmt.Sprintf("ai-accept-calibrate:%d", id),
		}, map[string]any{
			"ai_recommendation_id": id,
			"suggested_action":     action,
			"classification":       rec.Classification,
			"accepted":             true,
		})
		canCalibrate := ruleID != "" &&
			!learning.IsProtectedRuleID(source, ruleID) &&
			(learning.IsInformationalToolingRule(source, ruleID) ||
				!learning.IsProtectedFromAutoDowngrade("", category))
		if action == "calibrate_repo_scope" && canCalibrate {
			calRec := &store.CalibrationRecommendation{
				ID:                id,
				Scope:             "repo",
				RepositoryID:      &review.RepositoryID,
				Source:            source,
				RuleID:            ruleID,
				Category:          category,
				RecommendedAction: "report_only",
				Reason:            "Accepted AI advisory calibrate_repo_scope: " + strings.TrimSpace(rec.Reason),
				Confidence:        0.7,
			}
			if err := applyRepoCalibrationAccept(ctx, calRec, review.RepositoryID); err != nil {
				logger.Warnf("ai advisory accept calibration apply id=%d: %v", id, err)
			} else {
				emitRecommendationLearning(ctx, review.RepositoryID, id, true, source, ruleID)
			}
		}
	case "fix":
		emitLearning(ctx, store.LearningEvent{
			RepositoryID:   review.RepositoryID,
			ScanID:         review.ScanID,
			Fingerprint:    rec.FindingFingerprint,
			Source:         source,
			RuleID:         ruleID,
			EventType:      learning.EventUserMarkedTruePositive,
			CreatedBy:      "ai-advisory-accept",
			IdempotencyKey: fmt.Sprintf("ai-accept-fix:%d", id),
		})
	}

	return rdStore.UpdateAIAdvisoryRecommendationStatus(ctx, id, "accepted")
}

func (openclawReviewBridge) ListPendingRecommendations(c *gin.Context, limit int) ([]store.AIAdvisoryRecommendation, error) {
	return rdStore.ListPendingAIAdvisoryRecommendations(c.Request.Context(), limit)
}

func openClawFindingPoolLimit(cfg openclaw.Config) int {
	limit := cfg.MaxFindingsPerScan
	if limit <= 0 {
		limit = 25
	}
	pool := limit * 10
	if pool < 100 {
		pool = 100
	}
	if pool > 500 {
		pool = 500
	}
	return pool
}

func runOpenClawReview(ctx context.Context, scanID string) (openclaw.ReviewResult, error) {
	if openclawReviewService == nil {
		return openclaw.ReviewResult{}, fmt.Errorf("openclaw review service unavailable")
	}
	scan, err := rdStore.GetScan(ctx, scanID)
	if err != nil {
		return openclaw.ReviewResult{}, err
	}
	repo, err := rdStore.GetRepository(ctx, scan.RepositoryID)
	if err != nil {
		return openclaw.ReviewResult{}, err
	}
	cfg := config.OpenClawAIReview.Normalized()
	if decision := aiHarnessPreflight(ctx, cfg); decision.Skip {
		logger.Warnf("ai review skipped for scan %s: %s", scanID, decision.Reason)
		return openclaw.ReviewResult{Status: "skipped", Error: decision.Reason}, nil
	}
	findings, err := rdStore.ListFindingsForScan(ctx, scanID, openClawFindingPoolLimit(cfg))
	if err != nil {
		return openclaw.ReviewResult{}, err
	}
	instances, _ := rdStore.ListFindingInstancesByScan(ctx, scanID)
	scannerResults, _ := rdStore.ListScannerResultsByScan(ctx, scanID)
	var coverage []string
	for _, sr := range scannerResults {
		coverage = append(coverage, sr.ScannerName+":"+sr.Status)
	}
	return openclawReviewService.RunReview(ctx, openclaw.PacketInput{
		ScanID: scanID, Repository: repo,
		ScanType:        openclaw.FormatScanType(scan.TriggerType),
		IssueFiling:     issueFilingMode(),
		RemediationPR:   remediationPRMode(),
		ScannerCoverage: coverage,
		Findings:        findings,
		Instances:       instances,
	})
}

// aiHarnessPreflight skips AI spend when recent learning signals or a cheap
// endpoint probe indicate the gateway/path is unhealthy.
func aiHarnessPreflight(ctx context.Context, cfg openclaw.Config) openclaw.PreflightDecision {
	cfg = cfg.Normalized()
	lookback := time.Duration(cfg.PreflightLookbackHours) * time.Hour
	if lookback <= 0 {
		lookback = 6 * time.Hour
	}
	since := time.Now().UTC().Add(-lookback)
	timeouts, advisoryFails := 0, 0
	if rdStore != nil {
		if n, err := rdStore.CountLearningEventsSince(ctx, learning.EventLLMAuditorTimeout, since); err == nil {
			timeouts = n
		}
		if n, err := rdStore.CountLearningEventsSince(ctx, learning.EventAIAdvisoryFailed, since); err == nil {
			advisoryFails = n
		}
	}
	var probeErr error
	if cfg.PreflightProbeEnabled && cfg.EndpointConfigured() {
		probeErr = openclaw.ProbeEndpoint(ctx, cfg.EffectiveEndpoint(), 2*time.Second)
	}
	return openclaw.EvaluatePreflight(timeouts, advisoryFails, probeErr, cfg)
}

func maybeEnqueueOpenClawReview(ctx context.Context, scanCtx *store.ScanContext, repositoryID int64, result *analyzers.AnalysisResult, analysisErr error) {
	if openclawReviewService == nil || scanCtx == nil || scanCtx.ScanID == "" || analysisErr != nil || result == nil {
		return
	}
	cfg := config.OpenClawAIReview.Normalized()
	firstRunTune := shouldRunFirstRunAIRuleTuning(ctx, repositoryID, cfg)
	if !cfg.AutoAfterScan && !firstRunTune {
		return
	}
	if !cfg.CanInvoke() || !cfg.AllowsScanType(scanCtx.TriggerType) {
		return
	}
	if len(result.Issues) == 0 {
		return
	}
	// Advisory review is safe during report-only dry-runs: it never files forge
	// issues and is the primary path for AI-assisted learning / calibration.
	if decision := aiHarnessPreflight(ctx, cfg); decision.Skip {
		logger.Warnf("auto ai review skipped for scan %s: %s", scanCtx.ScanID, decision.Reason)
		return
	}
	scanID := scanCtx.ScanID
	timeout := time.Duration(cfg.TimeoutSeconds+15) * time.Second
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	go func() {
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		defer cancel()
		if _, err := rdStore.GetAIAdvisoryReviewByScanID(bg, scanID); err == nil {
			// Review already exists (manual/API). Still try first-run apply once.
			if firstRunTune {
				applied := applyFirstRunAIConfirmedCalibrations(bg, scanID, repositoryID)
				logger.Infof("first-run AI rule tuning for scan %s applied=%d (existing review)", scanID, applied)
				recomputeAfterAIReview(bg, scanID, true)
			}
			return
		}
		review, err := runOpenClawReview(bg, scanID)
		if err != nil {
			logger.Warnf("auto ai review for scan %s failed: %v", scanID, err)
			emitLearningEvidence(bg, store.LearningEvent{
				RepositoryID:   repositoryID,
				ScanID:         scanID,
				Source:         "openclaw-advisory",
				RuleID:         "ai_advisory",
				EventType:      learning.EventAIAdvisoryFailed,
				CreatedBy:      "auto-ai-review",
				IdempotencyKey: scanID + ":ai_advisory_failed",
			}, map[string]any{"error": err.Error()})
			return
		}
		if review.Status == "failed" || review.Status == "timeout" {
			logger.Warnf("auto ai review for scan %s ended with status=%s error=%s", scanID, review.Status, review.Error)
			emitLearningEvidence(bg, store.LearningEvent{
				RepositoryID:   repositoryID,
				ScanID:         scanID,
				Source:         "openclaw-advisory",
				RuleID:         "ai_advisory",
				EventType:      learning.EventAIAdvisoryFailed,
				CreatedBy:      "auto-ai-review",
				IdempotencyKey: scanID + ":ai_advisory_" + review.Status,
			}, map[string]any{"status": review.Status, "error": review.Error})
			return
		}
		logger.Infof("auto ai review for scan %s status=%s recommendations=%d", scanID, review.Status, review.RecommendationsCount)
		if firstRunTune {
			applied := applyFirstRunAIConfirmedCalibrations(bg, scanID, repositoryID)
			logger.Infof("first-run AI rule tuning for scan %s applied=%d", scanID, applied)
		}
		// Fold AI triage into the learner. During first-run tuning, skip
		// confidence-only auto-apply so suppressions require AI confirmation.
		recomputeAfterAIReview(bg, scanID, firstRunTune)
	}()
}

func recomputeAfterAIReview(ctx context.Context, scanID string, firstRunTune bool) {
	recomputeCtx := ctx
	if firstRunTune {
		recomputeCtx = withSkipCalibrationAutoApply(ctx)
	}
	if out, err := recomputeCalibration(recomputeCtx); err != nil {
		logger.Warnf("post-ai-review learning recompute for scan %s: %v", scanID, err)
	} else {
		logger.Infof("post-ai-review learning for scan %s: %v", scanID, out)
	}
}

// shouldRunFirstRunAIRuleTuning reports whether this connected repo still needs
// its one-shot AI-confirmed calibration pass (first completed scan only).
func shouldRunFirstRunAIRuleTuning(ctx context.Context, repositoryID int64, cfg openclaw.Config) bool {
	if repositoryID <= 0 || rdStore == nil || !cfg.FirstRunRuleTuning || !cfg.CanInvoke() {
		return false
	}
	raw, err := rdStore.GetOperatorEvidenceJSON(ctx, store.EvidenceAIFirstRunRuleTuneKey(repositoryID))
	if err != nil {
		logger.WithError(err).Debugf("first-run AI tune: evidence read failed for repo %d", repositoryID)
		return false
	}
	if strings.TrimSpace(raw) != "" {
		return false
	}
	completed, err := rdStore.CountCompletedScansForRepository(ctx, repositoryID)
	if err != nil {
		logger.WithError(err).Debugf("first-run AI tune: completed scan count failed for repo %d", repositoryID)
		return false
	}
	// Current finishing scan is already persisted as completed when this runs.
	return completed <= 1
}

// applyFirstRunAIConfirmedCalibrations installs repo report_only rules only for
// AI recommendations that explicitly confirm a false positive on non-protected rules.
func applyFirstRunAIConfirmedCalibrations(ctx context.Context, scanID string, repositoryID int64) int {
	if rdStore == nil || strings.TrimSpace(scanID) == "" || repositoryID <= 0 {
		return 0
	}
	review, err := rdStore.GetAIAdvisoryReviewByScanID(ctx, scanID)
	if err != nil {
		logger.WithError(err).Warnf("first-run AI tune: no review for scan %s", scanID)
		return 0
	}
	recs, err := rdStore.ListAIAdvisoryRecommendations(ctx, review.ReviewID)
	if err != nil {
		logger.WithError(err).Warnf("first-run AI tune: list recommendations failed for scan %s", scanID)
		return 0
	}
	applied := 0
	seenRules := map[string]bool{}
	for _, rec := range recs {
		action := strings.ToLower(strings.TrimSpace(rec.SuggestedAction))
		if action != "calibrate_repo_scope" {
			continue
		}
		if !learning.AIConfirmedFalsePositive(rec.Classification) {
			continue
		}
		source, ruleID, category, _ := rdStore.FindingSourceRuleForFingerprint(ctx, repositoryID, rec.FindingFingerprint)
		if ruleID == "" || learning.IsProtectedRuleID(source, ruleID) {
			continue
		}
		if err := learning.ValidateCalibrationAcceptRule(category, "repo", source, ruleID); err != nil {
			continue
		}
		key := strings.ToLower(source) + "|" + strings.ToUpper(ruleID)
		if seenRules[key] {
			continue
		}
		calRec := &store.CalibrationRecommendation{
			ID:                rec.ID,
			Scope:             "repo",
			RepositoryID:      &repositoryID,
			Source:            source,
			RuleID:            ruleID,
			Category:          category,
			RecommendedAction: "report_only",
			Reason:            "First-run AI rule tuning confirmed false positive: " + strings.TrimSpace(rec.Reason),
			Confidence:        0.8,
		}
		if err := applyRepoCalibrationAccept(ctx, calRec, repositoryID); err != nil {
			logger.Warnf("first-run AI tune apply id=%d %s/%s: %v", rec.ID, source, ruleID, err)
			continue
		}
		_ = rdStore.UpdateAIAdvisoryRecommendationStatus(ctx, rec.ID, "accepted")
		emitLearningEvidence(ctx, store.LearningEvent{
			RepositoryID:   repositoryID,
			ScanID:         scanID,
			Fingerprint:    rec.FindingFingerprint,
			Source:         source,
			RuleID:         ruleID,
			EventType:      learning.EventAIAdvisoryCalibrateSuggested,
			CreatedBy:      "first-run-ai-tune",
			IdempotencyKey: fmt.Sprintf("first-run-ai-tune:%d:%s", repositoryID, key),
		}, map[string]any{
			"ai_recommendation_id": rec.ID,
			"classification":       rec.Classification,
			"suggested_action":     action,
			"first_run_tune":       true,
		})
		seenRules[key] = true
		applied++
	}
	_ = rdStore.PutOperatorEvidence(ctx, store.EvidenceAIFirstRunRuleTuneKey(repositoryID), map[string]any{
		"tuned_at":       time.Now().UTC().Format(time.RFC3339),
		"scan_id":        scanID,
		"repository_id":  repositoryID,
		"rules_applied":  applied,
		"review_id":      review.ReviewID,
		"recommendations": len(recs),
	})
	return applied
}

func initOpenClawReview() error {
	cfg := config.OpenClawAIReview.Normalized()
	cfg.FallbackEndpoint = firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL)
	cfg.FallbackModel = firstNonEmpty(config.AIModel, config.OpenWebUIModel)
	cfg.FallbackAPIKey = firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken)
	config.OpenClawAIReview = cfg
	if rdStore == nil {
		return nil
	}
	if !cfg.EndpointConfigured() {
		return nil
	}
	endpoint := cfg.EffectiveEndpoint()
	// Treat advisory AI like an AI egress path even when LLM auditors are off.
	privacyDecision := privacy.EvaluateAIEgress(
		config.PrivacyMode,
		config.effectiveAIProvider(),
		endpoint,
		true,
	)
	if !privacyDecision.Allowed {
		if cfg.Enabled || cfg.CanInvoke() {
			return fmt.Errorf("privacy_mode=%s blocks OpenClaw/advisory AI endpoint %s: %s",
				privacy.NormalizeMode(config.PrivacyMode), endpoint, privacyDecision.Reason)
		}
		logger.Warnf("OpenClaw advisory endpoint blocked by privacy policy (%s); transport not started", privacyDecision.Reason)
		return nil
	}
	transport, err := ai.NewTransport(ai.Config{
		Provider:              ai.ProviderType(config.effectiveAIProvider()),
		BaseURL:               endpoint,
		APIKey:                cfg.FallbackAPIKey,
		Model:                 cfg.EffectiveModel(),
		InsecureSkipTLSVerify: config.AIInsecureSkipTLSVerify,
	}, logger)
	if err != nil {
		logger.Warnf("OpenClaw advisory review transport not configured: %v", err)
		return nil
	}
	openclawReviewService = openclaw.NewService(cfg, rdStore, transport)
	if cfg.Enabled {
		if cfg.AutoAfterScan {
			logger.Info("OpenClaw advisory review enabled (auto after scan, advisory-only, redacted packets)")
		} else {
			logger.Info("OpenClaw advisory review enabled (manual trigger, advisory-only, redacted packets)")
		}
	} else {
		logger.Info("OpenClaw advisory review transport ready (disabled by default)")
	}
	return nil
}

func applyOpenClawDefaults(cfg *Config) {
	def := openclaw.DefaultConfig()
	c := &cfg.OpenClawAIReview
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = def.TimeoutSeconds
	}
	if c.MaxFindingsPerScan <= 0 {
		c.MaxFindingsPerScan = def.MaxFindingsPerScan
	}
	if c.MaxTokensPerScan < 0 {
		c.MaxTokensPerScan = 0
	}
	if strings.TrimSpace(c.ValueMode) == "" {
		c.ValueMode = def.ValueMode
	}
	if c.MaxAuditorFailuresPerScan <= 0 {
		c.MaxAuditorFailuresPerScan = def.MaxAuditorFailuresPerScan
	}
	if c.PreflightTimeoutEvents <= 0 {
		c.PreflightTimeoutEvents = def.PreflightTimeoutEvents
	}
	if c.PreflightAdvisoryFailEvents <= 0 {
		c.PreflightAdvisoryFailEvents = def.PreflightAdvisoryFailEvents
	}
	if c.PreflightLookbackHours <= 0 {
		c.PreflightLookbackHours = def.PreflightLookbackHours
	}
	// AbortOnFirstTimeout / PreflightProbeEnabled default true via viper.SetDefault
	// before unmarshal so operators can still set them false explicitly.
}

func issueFilingMode() string {
	if config.AutoCreateIssues {
		return "enabled"
	}
	return "disabled"
}

func remediationPRMode() string {
	if config.RemediationPREnabled {
		return "enabled"
	}
	return "disabled"
}
