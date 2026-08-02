package main

import (
	"fmt"

	"git.commsnet.org/commstech/repository-detective/ai"
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
	if openclawReviewService == nil {
		return openclaw.ReviewResult{}, fmt.Errorf("openclaw review service unavailable")
	}
	ctx := c.Request.Context()
	scan, err := rdStore.GetScan(ctx, scanID)
	if err != nil {
		return openclaw.ReviewResult{}, err
	}
	repo, err := rdStore.GetRepository(ctx, scan.RepositoryID)
	if err != nil {
		return openclaw.ReviewResult{}, err
	}
	cfg := config.OpenClawAIReview.Normalized()
	limit := cfg.MaxFindingsPerScan
	findings, err := rdStore.ListFindingsForScan(ctx, scanID, limit)
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

func initOpenClawReview() {
	cfg := config.OpenClawAIReview.Normalized()
	cfg.FallbackEndpoint = firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL)
	cfg.FallbackModel = firstNonEmpty(config.AIModel, config.OpenWebUIModel)
	cfg.FallbackAPIKey = firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken)
	config.OpenClawAIReview = cfg
	if rdStore == nil {
		return
	}
	if !cfg.EndpointConfigured() {
		return
	}
	transport, err := ai.NewTransport(ai.Config{
		Provider:              ai.ProviderType(config.effectiveAIProvider()),
		BaseURL:               cfg.EffectiveEndpoint(),
		APIKey:                cfg.FallbackAPIKey,
		Model:                 cfg.EffectiveModel(),
		InsecureSkipTLSVerify: config.AIInsecureSkipTLSVerify,
	}, logger)
	if err != nil {
		logger.Warnf("OpenClaw advisory review transport not configured: %v", err)
		return
	}
	openclawReviewService = openclaw.NewService(cfg, rdStore, transport)
	if cfg.Enabled {
		logger.Info("OpenClaw advisory review enabled (advisory-only, redacted packets)")
	} else {
		logger.Info("OpenClaw advisory review transport ready (disabled by default)")
	}
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
