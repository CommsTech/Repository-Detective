package main

import (
	"context"
	"crypto/hmac"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/analyzers"
	"git.commsnet.org/commstech/bugbot/gitea"
	"git.commsnet.org/commstech/bugbot/handlers"
	"git.commsnet.org/commstech/bugbot/issues"
	"git.commsnet.org/commstech/bugbot/limiter"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var version = "dev"

var (
	logger            *logrus.Logger
	config            *Config
	giteaClient       *gitea.Client
	aiClient          *ai.Client
	analysisEngine    *analyzers.Engine
	issueManager      *issues.Manager
	webhookHandler    *handlers.WebhookHandler
	onboardingHandler *handlers.OnboardingHandler
	analysisLimiter   *limiter.ConcurrencyLimiter
)

// Config holds the plugin configuration
type Config struct {
	Port                  string            `mapstructure:"port"`
	APIKey                string            `mapstructure:"api_key"` // API key for manual analysis endpoints
	GiteaURL              string            `mapstructure:"gitea_url"`
	GiteaToken            string            `mapstructure:"gitea_token"`
	WebhookSecret         string            `mapstructure:"webhook_secret"`
	AIProvider            string            `mapstructure:"ai_provider"`
	AIBaseURL             string            `mapstructure:"ai_base_url"`
	AIAPIKey              string            `mapstructure:"ai_api_key"`
	AIModel               string            `mapstructure:"ai_model"`
	OpenWebUIURL          string            `mapstructure:"openwebui_url"`
	OpenWebUIToken        string            `mapstructure:"openwebui_token"`
	OpenWebUIModel        string            `mapstructure:"openwebui_model"`
	LogLevel              string            `mapstructure:"log_level"`
	AnalysisDepth         int               `mapstructure:"analysis_depth"`
	MaxFileSize           int64             `mapstructure:"max_file_size"`
	EnableSecurity        bool              `mapstructure:"enable_security"`
	EnableQuality         bool              `mapstructure:"enable_quality"`
	AutoCreateIssues      bool              `mapstructure:"auto_create_issues"`
	MaxIssuesPerRun       int               `mapstructure:"max_issues_per_run"`
	SkipLowSeverity       bool              `mapstructure:"skip_low_severity"`
	GroupSimilarIssues    bool              `mapstructure:"group_similar_issues"`
	SkipPatterns              []string          `mapstructure:"-"`
	LanguageMapping           map[string]string `mapstructure:"-"`
	RepositoryIncludePatterns []string          `mapstructure:"-"`
	RepositoryExcludePatterns []string          `mapstructure:"-"`
	PublicURL                 string            `mapstructure:"public_url"`
	MaxConcurrentAnalyses     int               `mapstructure:"max_concurrent_analyses"`
	AnalysisTimeout       int               `mapstructure:"analysis_timeout"`
	RateLimitPerMinute    int               `mapstructure:"rate_limit_per_minute"`
	SkipStartupChecks     bool              `mapstructure:"skip_startup_checks"`
}

func main() {
	// Initialize logger
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	if err := loadConfig(); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Set log level
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Warnf("Invalid log level %s, using info: %v", config.LogLevel, err)
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.Info("Starting Gitea Bugbot Plugin...")
	logger.Infof("Configuration loaded: Port=%s, GiteaURL=%s, AIProvider=%s",
		config.Port, config.GiteaURL, config.effectiveAIProvider())

	// Initialize components
	if err := initializeComponents(); err != nil {
		logger.Fatalf("Failed to initialize components: %v", err)
	}

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Setup routes
	setupRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting server on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("analysis_depth", 3)
	viper.SetDefault("max_file_size", 1024*1024) // 1MB
	viper.SetDefault("enable_security", true)
	viper.SetDefault("enable_quality", true)
	viper.SetDefault("auto_create_issues", true)
	viper.SetDefault("max_issues_per_run", 50)
	viper.SetDefault("max_concurrent_analyses", 5)
	viper.SetDefault("analysis_timeout", 300)
	viper.SetDefault("rate_limit_per_minute", 60)
	viper.SetDefault("openwebui_model", "default")
	viper.SetDefault("ai_provider", "")
	viper.SetDefault("ai_model", "")
	viper.SetDefault("skip_startup_checks", false)

	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("BUGBOT")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		logger.Warn("No config file found, using defaults and environment variables")
	}

	config = &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := viper.UnmarshalKey("skip_patterns", &config.SkipPatterns); err != nil {
		return fmt.Errorf("failed to unmarshal skip_patterns: %w", err)
	}
	if raw := viper.GetStringMap("language_mapping"); len(raw) > 0 {
		config.LanguageMapping = make(map[string]string, len(raw))
		for key, value := range raw {
			config.LanguageMapping[key] = fmt.Sprint(value)
		}
	}
	if err := viper.UnmarshalKey("repository_include_patterns", &config.RepositoryIncludePatterns); err != nil {
		return fmt.Errorf("failed to unmarshal repository_include_patterns: %w", err)
	}
	if err := viper.UnmarshalKey("repository_exclude_patterns", &config.RepositoryExcludePatterns); err != nil {
		return fmt.Errorf("failed to unmarshal repository_exclude_patterns: %w", err)
	}

	// Validate required fields
	if config.GiteaURL == "" {
		return fmt.Errorf("gitea_url is required")
	}
	if config.GiteaToken == "" {
		return fmt.Errorf("gitea_token is required")
	}
	if config.effectiveAIProvider() == "" && config.OpenWebUIURL == "" && config.AIBaseURL == "" {
		return fmt.Errorf("configure ai_provider + ai_base_url, or legacy openwebui_url")
	}
	if config.AnalysisTimeout <= 0 {
		config.AnalysisTimeout = 300
	}
	if config.MaxConcurrentAnalyses <= 0 {
		config.MaxConcurrentAnalyses = 5
	}

	return nil
}

func setupRoutes(router *gin.Engine) {
	// Set body size limit for all routes (防止 DoS)
	router.MaxMultipartMemory = 8 << 20 // 8 MB max

	// Health check — no auth required
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "gitea-bugbot",
			"version": version,
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/onboard")
	})

	// Webhook endpoint for Gitea — rate limited and webhook secret auth
	router.POST("/webhook", func(c *gin.Context) {
		webhookHandler.HandleWebhook(c)
	})

	// API endpoints — require API key auth
	api := router.Group("/api/v1")
	api.Use(requireAPIKeyAuth())
	{
		api.POST("/analyze", handleManualAnalysis)
		api.GET("/status", handleStatus)
		api.POST("/config/reload", handleConfigReload)
	}

	onboardingHandler = handlers.NewOnboardingHandler(logger, handlers.OnboardingConfig{
		GiteaURL:  config.GiteaURL,
		PublicURL: config.PublicURL,
		AIConfig: ai.Config{
			Provider: ai.ProviderType(config.AIProvider),
			BaseURL:  firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
			APIKey:   firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken),
			Model:    firstNonEmpty(config.AIModel, config.OpenWebUIModel),
		},
	})
	onboardingHandler.RegisterRoutes(router, api)

	logger.Info("Routes configured successfully")
}

// requireAPIKeyAuth middleware requires BUGBOT_API_KEY for API endpoints
func requireAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.APIKey == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "API key not configured"})
			return
		}

		apiKey := c.GetHeader("X-Bugbot-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}

		// Constant-time comparison to prevent timing attacks
		if !hmac.Equal([]byte(apiKey), []byte(config.APIKey)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		c.Next()
	}
}

// initializeComponents initializes all the plugin components
func initializeComponents() error {
	logger.Info("Initializing components...")

	// Initialize Gitea client
	giteaClient = gitea.NewClient(config.GiteaURL, config.GiteaToken, logger)

	// Test Gitea connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := giteaClient.TestConnection(ctx); err != nil {
		if config.SkipStartupChecks {
			logger.Warnf("Gitea connection check failed (skipped): %v", err)
		} else {
			return fmt.Errorf("failed to connect to Gitea: %w", err)
		}
	} else {
		logger.Info("Gitea connection established")
	}

	// Initialize AI client (multi-provider)
	var err error
	aiClient, err = ai.NewClient(ai.Config{
		Provider: ai.ProviderType(config.AIProvider),
		BaseURL:  firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
		APIKey:   firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken),
		Model:    firstNonEmpty(config.AIModel, config.OpenWebUIModel),
	}, ai.LegacyConfig{
		OpenWebUIURL:   config.OpenWebUIURL,
		OpenWebUIToken: config.OpenWebUIToken,
		OpenWebUIModel: config.OpenWebUIModel,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to configure AI client: %w", err)
	}

	// Test AI connection
	if err := aiClient.TestConnection(ctx); err != nil {
		if config.SkipStartupChecks {
			logger.Warnf("AI provider connection check failed (skipped): %v", err)
		} else {
			return fmt.Errorf("failed to connect to AI provider: %w", err)
		}
	} else {
		logger.Infof("AI provider connection established (%s, model=%s)", aiClient.Provider(), aiClient.Model())
	}

	// Initialize analysis engine
	analysisConfig := &analyzers.Config{
		MaxFileSize:     config.MaxFileSize,
		AnalysisDepth:   config.AnalysisDepth,
		EnableSecurity:  config.EnableSecurity,
		EnableQuality:   config.EnableQuality,
		SkipPatterns:    config.SkipPatterns,
		LanguageMapping: config.LanguageMapping,
	}
	analysisEngine = analyzers.NewEngine(giteaClient, aiClient, analysisConfig, logger)

	// Initialize issue manager
	issueConfig := &issues.Config{
		AutoCreateIssues:   config.AutoCreateIssues,
		IssueLabels:        []string{"bugbot", "automated-review"},
		MaxIssuesPerRun:    config.MaxIssuesPerRun,
		SkipLowSeverity:    config.SkipLowSeverity,
		GroupSimilarIssues: config.GroupSimilarIssues,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "## Issue Details\n\n**Severity:** {{severity}}\n**Category:** {{category}}\n**Confidence:** {{confidence}}\n\n## Description\n\n{{description}}\n\n## Context\n\n- **Repository:** {{repository}}\n- **Context:** {{context}}\n- **Commit:** {{commit}}\n\n---\n*This issue was automatically generated by the Gitea Bugbot*",
	}
	issueManager = issues.NewManager(giteaClient, issueConfig, logger)

	webhookHandler = handlers.NewWebhookHandler(logger, &handlers.Config{
		WebhookSecret:   config.WebhookSecret,
		IncludePatterns: config.RepositoryIncludePatterns,
		ExcludePatterns: config.RepositoryExcludePatterns,
	}, &webhookProcessor{})

	analysisLimiter = limiter.New(config.MaxConcurrentAnalyses)
	logger.Infof("Analysis concurrency limit: %d", config.MaxConcurrentAnalyses)

	logger.Info("All components initialized successfully")
	return nil
}

type webhookProcessor struct{}

func (p *webhookProcessor) ProcessPush(ctx context.Context, payload *handlers.GiteaWebhookPayload) {
	owner := payload.Repository.Owner.Username
	repo := payload.Repository.Name
	ref := payload.After
	changedFiles := handlers.CollectChangedFiles(payload.Commits)

	runAnalysis(ctx, func(analysisCtx context.Context) {
		result, err := analysisEngine.AnalyzeChangedFiles(analysisCtx, owner, repo, ref, changedFiles)
		if err != nil {
			logger.Errorf("Push analysis failed: %v", err)
			return
		}
		createIssuesFromResult(analysisCtx, owner, repo, result, fmt.Sprintf("Push to %s", payload.Ref), ref, 0)
	})
}

func (p *webhookProcessor) ProcessPullRequest(ctx context.Context, payload *handlers.GiteaWebhookPayload) {
	owner := payload.Repository.Owner.Username
	repo := payload.Repository.Name
	prNumber := payload.PullRequest.Number

	runAnalysis(ctx, func(analysisCtx context.Context) {
		result, err := analysisEngine.AnalyzePullRequest(analysisCtx, owner, repo, prNumber)
		if err != nil {
			logger.Errorf("Pull request analysis failed: %v", err)
			return
		}
		createIssuesFromResult(analysisCtx, owner, repo, result, fmt.Sprintf("Pull Request #%d", prNumber), "", prNumber)
	})
}

func runAnalysis(_ context.Context, fn func(context.Context)) {
	analysisCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
	defer cancel()

	if err := analysisLimiter.Run(analysisCtx, func() { fn(analysisCtx) }); err != nil {
		logger.Warnf("Analysis skipped — concurrency limit reached or timed out waiting for slot: %v", err)
	}
}

func createIssuesFromResult(ctx context.Context, owner, repo string, result *analyzers.AnalysisResult, contextLabel, commit string, prNumber int) {
	if len(result.Issues) == 0 {
		return
	}

	issueReq := &issues.IssueCreationRequest{
		Owner:      owner,
		Repository: repo,
		AnalysisResult: &ai.CodeAnalysisResult{
			Issues:       result.Issues,
			OverallScore: result.OverallScore,
			AnalysisTime: result.AnalysisTime,
		},
		Context:     contextLabel,
		Commit:      commit,
		PullRequest: prNumber,
	}

	issueResult, err := issueManager.CreateIssuesFromAnalysis(ctx, issueReq)
	if err != nil {
		logger.Errorf("Failed to create issues: %v", err)
		return
	}

	logger.Infof("Created %d issues, skipped %d", issueResult.IssuesCreated, issueResult.IssuesSkipped)
}

// handleManualAnalysis handles manual analysis requests
func handleManualAnalysis(c *gin.Context) {
	var req struct {
		Owner      string `json:"owner" binding:"required"`
		Repository string `json:"repository" binding:"required"`
		Ref        string `json:"ref"`
		Type       string `json:"type"` // "repository" or "pull_request"
		PRNumber   int    `json:"pr_number"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Start analysis in background
	go func() {
		runAnalysis(c.Request.Context(), func(ctx context.Context) {
			var result *analyzers.AnalysisResult
			var err error

			if req.Type == "pull_request" && req.PRNumber > 0 {
				result, err = analysisEngine.AnalyzePullRequest(ctx, req.Owner, req.Repository, req.PRNumber)
			} else {
				ref := req.Ref
				if ref == "" {
					ref = "main"
				}
				result, err = analysisEngine.AnalyzeRepository(ctx, req.Owner, req.Repository, ref)
			}

			if err != nil {
				logger.Errorf("Manual analysis failed: %v", err)
				return
			}

			if len(result.Issues) > 0 {
				createIssuesFromResult(ctx, req.Owner, req.Repository, result,
					fmt.Sprintf("Manual analysis - %s", req.Type), req.Ref, req.PRNumber)
			}
		})
	}()

	c.JSON(http.StatusOK, gin.H{"status": "analysis started"})
}

// handleStatus handles status requests
func handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "running",
		"service":     "gitea-bugbot",
		"version":     version,
		"ai_provider": aiClient.Provider(),
		"ai_model":    aiClient.Model(),
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}

// handleConfigReload handles configuration reload requests
func handleConfigReload(c *gin.Context) {
	if err := loadConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reinitialize components with new config
	if err := initializeComponents(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reinitialize components"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "configuration reloaded"})
}

func (c *Config) effectiveAIProvider() string {
	if c.AIProvider != "" {
		return c.AIProvider
	}
	if c.OpenWebUIURL != "" {
		return string(ai.ProviderOpenWebUI)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
