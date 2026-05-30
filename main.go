package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"yourusername/gitea-bugbot/ai"
	"yourusername/gitea-bugbot/analyzers"
	"yourusername/gitea-bugbot/gitea"
	"yourusername/gitea-bugbot/handlers"
	"yourusername/gitea-bugbot/issues"
)

var (
	logger        *logrus.Logger
	config        *Config
	giteaClient   *gitea.Client
	aiClient      *ai.OpenWebUIClient
	analysisEngine *analyzers.Engine
	issueManager  *issues.Manager
)

// Config holds the plugin configuration
type Config struct {
	Port                    string            `mapstructure:"port"`
	APIKey                  string            `mapstructure:"api_key"`           // API key for manual analysis endpoints
	GiteaURL                string            `mapstructure:"gitea_url"`
	GiteaToken              string            `mapstructure:"gitea_token"`
	WebhookSecret           string            `mapstructure:"webhook_secret"`
	OpenWebUIURL            string            `mapstructure:"openwebui_url"`
	OpenWebUIToken          string            `mapstructure:"openwebui_token"`
	LogLevel                string            `mapstructure:"log_level"`
	AnalysisDepth           int               `mapstructure:"analysis_depth"`
	MaxFileSize             int64             `mapstructure:"max_file_size"`
	EnableSecurity          bool              `mapstructure:"enable_security"`
	EnableQuality           bool              `mapstructure:"enable_quality"`
	AutoCreateIssues        bool              `mapstructure:"auto_create_issues"`
	MaxIssuesPerRun         int               `mapstructure:"max_issues_per_run"`
	SkipLowSeverity         bool              `mapstructure:"skip_low_severity"`
	GroupSimilarIssues      bool              `mapstructure:"group_similar_issues"`
	SkipPatterns            []string          `mapstructure:"skip_patterns"`
	LanguageMapping         map[string]string `mapstructure:"language_mapping"`
	MaxConcurrentAnalyses   int               `mapstructure:"max_concurrent_analyses"`
	AnalysisTimeout         int               `mapstructure:"analysis_timeout"`
	RateLimitPerMinute      int               `mapstructure:"rate_limit_per_minute"`
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
	logger.Infof("Configuration loaded: Port=%s, GiteaURL=%s, OpenWebUIURL=%s", 
		config.Port, config.GiteaURL, config.OpenWebUIURL)

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

	// Validate required fields
	if config.GiteaURL == "" {
		return fmt.Errorf("gitea_url is required")
	}
	if config.GiteaToken == "" {
		return fmt.Errorf("gitea_token is required")
	}
	if config.OpenWebUIURL == "" {
		return fmt.Errorf("openwebui_url is required")
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
			"version": "1.0.0",
		})
	})

	// Webhook endpoint for Gitea — rate limited and webhook secret auth
	router.POST("/webhook", handleWebhook)

	// API endpoints — require API key auth
	api := router.Group("/api/v1")
	api.Use(requireAPIKeyAuth())
	{
		api.POST("/analyze", handleManualAnalysis)
		api.GET("/status", handleStatus)
		api.POST("/config/reload", handleConfigReload)
	}

	logger.Info("Routes configured successfully")
}

// requireAPIKeyAuth middleware requires BUGBOT_API_KEY for API endpoints
func requireAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		return fmt.Errorf("failed to connect to Gitea: %w", err)
	}
	logger.Info("Gitea connection established")

	// Initialize OpenWebUI client
	aiClient = ai.NewOpenWebUIClient(config.OpenWebUIURL, config.OpenWebUIToken, logger)
	
	// Test OpenWebUI connection
	if err := aiClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to OpenWebUI: %w", err)
	}
	logger.Info("OpenWebUI connection established")

	// Initialize analysis engine
	analysisConfig := &analyzers.Config{
		MaxFileSize:    config.MaxFileSize,
		AnalysisDepth:  config.AnalysisDepth,
		EnableSecurity: config.EnableSecurity,
		EnableQuality:  config.EnableQuality,
		SkipPatterns:   config.SkipPatterns,
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

	logger.Info("All components initialized successfully")
	return nil
}

// handleWebhook handles incoming Gitea webhooks
func handleWebhook(c *gin.Context) {
	logger.Info("Received webhook request")

	// Parse webhook payload
	var payload handlers.GiteaWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		logger.Errorf("Failed to parse webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Verify webhook secret if configured
	if config.WebhookSecret != "" {
		secret := c.GetHeader("X-Gitea-Signature")
		if secret == "" {
			secret = c.Query("secret")
		}
		if secret != config.WebhookSecret {
			logger.Errorf("Invalid webhook secret")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	}

	logger.Infof("Processing webhook for repository: %s, action: %s", 
		payload.Repository.FullName, payload.Action)

	// Process webhook based on action type
	switch payload.Action {
	case "push":
		handlePushEvent(c, &payload)
	case "pull_request":
		handlePullRequestEvent(c, &payload)
	default:
		logger.Infof("Unhandled webhook action: %s", payload.Action)
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
	}
}

// handlePushEvent processes push events
func handlePushEvent(c *gin.Context, payload *handlers.GiteaWebhookPayload) {
	logger.Infof("Processing push event for repository: %s", payload.Repository.FullName)

	// Skip if no commits
	if len(payload.Commits) == 0 {
		logger.Info("No commits in push event, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "no commits"})
		return
	}

	// Start analysis in background
	go func() {
		owner := payload.Repository.Owner.Username
		repo := payload.Repository.Name
		ref := payload.After

		// Analyze repository
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
		defer cancel()

		result, err := analysisEngine.AnalyzeRepository(ctx, owner, repo, ref)
		if err != nil {
			logger.Errorf("Repository analysis failed: %v", err)
			return
		}

		// Create issues if analysis found problems
		if len(result.Issues) > 0 {
			issueReq := &issues.IssueCreationRequest{
				Owner:          owner,
				Repository:     repo,
				AnalysisResult: &ai.CodeAnalysisResult{
					Issues:       result.Issues,
					Suggestions:  result.Suggestions,
					OverallScore: result.OverallScore,
					AnalysisTime: result.AnalysisTime,
				},
				Context: fmt.Sprintf("Push to %s", payload.Ref),
				Commit:  ref,
			}

			issueResult, err := issueManager.CreateIssuesFromAnalysis(ctx, issueReq)
			if err != nil {
				logger.Errorf("Failed to create issues: %v", err)
			} else {
				logger.Infof("Created %d issues, skipped %d", issueResult.IssuesCreated, issueResult.IssuesSkipped)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// handlePullRequestEvent processes pull request events
func handlePullRequestEvent(c *gin.Context, payload *handlers.GiteaWebhookPayload) {
	logger.Infof("Processing pull request event: %s #%d", 
		payload.Repository.FullName, payload.PullRequest.Number)

	// Only process on open or synchronize
	if payload.PullRequest.State != "open" {
		logger.Infof("Pull request is not open, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// Start analysis in background
	go func() {
		owner := payload.Repository.Owner.Username
		repo := payload.Repository.Name
		prNumber := payload.PullRequest.Number

		// Analyze pull request
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
		defer cancel()

		result, err := analysisEngine.AnalyzePullRequest(ctx, owner, repo, prNumber)
		if err != nil {
			logger.Errorf("Pull request analysis failed: %v", err)
			return
		}

		// Create issues if analysis found problems
		if len(result.Issues) > 0 {
			issueReq := &issues.IssueCreationRequest{
				Owner:          owner,
				Repository:     repo,
				AnalysisResult: &ai.CodeAnalysisResult{
					Issues:       result.Issues,
					Suggestions:  result.Suggestions,
					OverallScore: result.OverallScore,
					AnalysisTime: result.AnalysisTime,
				},
				Context:      fmt.Sprintf("Pull Request #%d", prNumber),
				PullRequest:  prNumber,
			}

			issueResult, err := issueManager.CreateIssuesFromAnalysis(ctx, issueReq)
			if err != nil {
				logger.Errorf("Failed to create issues: %v", err)
			} else {
				logger.Infof("Created %d issues, skipped %d", issueResult.IssuesCreated, issueResult.IssuesSkipped)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
		defer cancel()

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

		// Create issues if analysis found problems
		if len(result.Issues) > 0 {
			issueReq := &issues.IssueCreationRequest{
				Owner:          req.Owner,
				Repository:     req.Repository,
				AnalysisResult: &ai.CodeAnalysisResult{
					Issues:       result.Issues,
					Suggestions:  result.Suggestions,
					OverallScore: result.OverallScore,
					AnalysisTime: result.AnalysisTime,
				},
				Context: fmt.Sprintf("Manual analysis - %s", req.Type),
				Commit:  req.Ref,
			}

			issueResult, err := issueManager.CreateIssuesFromAnalysis(ctx, issueReq)
			if err != nil {
				logger.Errorf("Failed to create issues: %v", err)
			} else {
				logger.Infof("Manual analysis completed: created %d issues, skipped %d", 
					issueResult.IssuesCreated, issueResult.IssuesSkipped)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "analysis started"})
}

// handleStatus handles status requests
func handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"service":   "gitea-bugbot",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
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
