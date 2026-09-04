package main

import (
	"context"
	"fmt"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
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
	return runOpenClawReview(c.Request.Context(), scanID)
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
	if cfg.RequireOperatorApproval {
		return rdStore.UpdateAIAdvisoryRecommendationStatus(c.Request.Context(), id, "accepted")
	}
	return fmt.Errorf("operator approval required")
}

func (openclawReviewBridge) RejectRecommendation(c *gin.Context, id int64) error {
	return rdStore.UpdateAIAdvisoryRecommendationStatus(c.Request.Context(), id, "rejected")
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

func maybeEnqueueOpenClawReview(ctx context.Context, scanCtx *store.ScanContext, repositoryID int64, result *analyzers.AnalysisResult, analysisErr error) {
	if openclawReviewService == nil || scanCtx == nil || scanCtx.ScanID == "" || analysisErr != nil || result == nil {
		return
	}
	cfg := config.OpenClawAIReview.Normalized()
	if !cfg.AutoAfterScan || !cfg.CanInvoke() || !cfg.AllowsScanType(scanCtx.TriggerType) {
		return
	}
	if len(result.Issues) == 0 {
		return
	}
	if reportOnlyDryRunFromContext(ctx) {
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
			return
		}
		review, err := runOpenClawReview(bg, scanID)
		if err != nil {
			logger.Warnf("auto ai review for scan %s failed: %v", scanID, err)
			return
		}
		if review.Status == "failed" || review.Status == "timeout" {
			logger.Warnf("auto ai review for scan %s ended with status=%s error=%s", scanID, review.Status, review.Error)
			return
		}
		logger.Infof("auto ai review for scan %s status=%s recommendations=%d", scanID, review.Status, review.RecommendationsCount)
	}()
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
