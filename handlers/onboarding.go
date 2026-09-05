package handlers

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/web"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// OnboardingHandler serves the setup UI and onboarding API.
type OnboardingHandler struct {
	logger        *logrus.Logger
	giteaURL      string
	publicURL     string
	giteaScanOrgs []string
	aiConfig      ai.Config
	authMode      string
	userCounter   func(context.Context) (int, error)
	repoCounter   func(context.Context) (int, error)
}

// OnboardingConfig holds server-side defaults for the onboarding UI.
type OnboardingConfig struct {
	GiteaURL      string
	PublicURL     string
	GiteaScanOrgs []string
	AIConfig      ai.Config
	AuthMode      string
	// Optional counters for fresh-install detection (nil-safe).
	CountUsers        func(context.Context) (int, error)
	CountRepositories func(context.Context) (int, error)
}

// NewOnboardingHandler creates an onboarding handler.
func NewOnboardingHandler(logger *logrus.Logger, cfg OnboardingConfig) *OnboardingHandler {
	return &OnboardingHandler{
		logger:        logger,
		giteaURL:      cfg.GiteaURL,
		publicURL:     cfg.PublicURL,
		giteaScanOrgs: append([]string(nil), cfg.GiteaScanOrgs...),
		aiConfig:      cfg.AIConfig,
		authMode:      strings.TrimSpace(cfg.AuthMode),
		userCounter:   cfg.CountUsers,
		repoCounter:   cfg.CountRepositories,
	}
}

// RegisterRoutes mounts onboarding UI and API routes.
// onboardAPI should be /api/v1/onboard with API key auth but without requiring full startup.
func (h *OnboardingHandler) RegisterRoutes(router *gin.Engine, onboardAPI *gin.RouterGroup) {
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		h.logger.Errorf("Failed to load onboarding static files: %v", err)
		return
	}

	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		h.logger.Errorf("Failed to load onboarding index.html: %v", err)
		return
	}

	serveOnboard := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}
	// Both paths avoid a Gin trailing-slash redirect loop with StaticFS under /onboard/*.
	router.GET("/onboard", serveOnboard)
	router.GET("/onboard/", serveOnboard)
	// Assets live under /onboard/assets (not /onboard/static) to avoid route prefix clashes.
	router.StaticFS("/onboard/assets", http.FS(staticFS))

	onboardAPI.GET("/defaults", h.handleDefaultsExtended)
	onboardAPI.POST("/test-gitea", h.handleTestGitea)
	onboardAPI.POST("/test-ai", h.handleTestAI)
	onboardAPI.POST("/repos", h.handleListRepos)
	onboardAPI.POST("/webhooks", h.handleRegisterWebhooks)
	h.registerPhase4Routes(onboardAPI)
}

func (h *OnboardingHandler) handleDefaults(c *gin.Context) {
	webhookURL := strings.TrimSuffix(h.publicURL, "/") + "/webhook"
	c.JSON(http.StatusOK, gin.H{
		"gitea_url":       h.giteaURL,
		"public_url":      h.publicURL,
		"webhook_url":     webhookURL,
		"gitea_scan_orgs": h.giteaScanOrgs,
		"ai_provider":     h.aiConfig.Provider,
		"ai_model":        h.aiConfig.Model,
		"ai_base_url":     h.aiConfig.BaseURL,
	})
}

type onboardConnectionRequest struct {
	GiteaURL      string   `json:"gitea_url"`
	GiteaToken    string   `json:"gitea_token"`
	PublicURL     string   `json:"public_url"`
	WebhookSecret string   `json:"webhook_secret"`
	AIProvider    string   `json:"ai_provider"`
	AIBaseURL     string   `json:"ai_base_url"`
	AIAPIKey      string   `json:"ai_api_key"`
	AIModel       string   `json:"ai_model"`
	Orgs          []string `json:"orgs"`
	Repositories  []string `json:"repositories"`
}

func (h *OnboardingHandler) handleTestGitea(c *gin.Context) {
	var req onboardConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	url := firstNonEmptyStr(req.GiteaURL, h.giteaURL)
	if url == "" || req.GiteaToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gitea URL and token are required"})
		return
	}

	client := gitea.NewClient(url, req.GiteaToken, h.logger)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": fmt.Sprintf("Connection failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gitea connection successful"})
}

func (h *OnboardingHandler) handleTestAI(c *gin.Context) {
	var req onboardConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	client, err := ai.NewClient(ai.Config{
		Provider: ai.ProviderType(req.AIProvider),
		BaseURL:  req.AIBaseURL,
		APIKey:   req.AIAPIKey,
		Model:    req.AIModel,
	}, ai.LegacyConfig{}, h.logger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": fmt.Sprintf("AI connection failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("AI connection successful (%s, model=%s)", client.Provider(), client.Model()),
		"provider": client.Provider(),
		"model":    client.Model(),
	})
}

func (h *OnboardingHandler) handleListRepos(c *gin.Context) {
	var req onboardConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	url := firstNonEmptyStr(req.GiteaURL, h.giteaURL)
	if url == "" || req.GiteaToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gitea URL and token are required"})
		return
	}

	client := gitea.NewClient(url, req.GiteaToken, h.logger)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	repos, err := h.listOnboardRepositories(ctx, client, req.Orgs)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repos,
		"count":        len(repos),
	})
}

func (h *OnboardingHandler) handleRegisterWebhooks(c *gin.Context) {
	var req onboardConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	url := firstNonEmptyStr(req.GiteaURL, h.giteaURL)
	publicURL := strings.TrimSuffix(firstNonEmptyStr(req.PublicURL, h.publicURL), "/")
	if url == "" || req.GiteaToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gitea URL and token are required"})
		return
	}
	if publicURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Repository Detective public URL is required for webhooks"})
		return
	}
	if len(req.Repositories) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Select at least one repository"})
		return
	}

	client := gitea.NewClient(url, req.GiteaToken, h.logger)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	webhookURL := publicURL + "/webhook"
	var created, failed int
	var errors []string

	for _, fullName := range req.Repositories {
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) != 2 {
			failed++
			errors = append(errors, fmt.Sprintf("invalid repo name: %s", fullName))
			continue
		}

		hook := &gitea.HookConfig{
			Type: "gitea",
			Events: []string{
				"push",
				"pull_request",
			},
			Active: true,
		}
		hook.Config.URL = webhookURL
		hook.Config.ContentType = "json"
		hook.Config.Secret = req.WebhookSecret

		if err := client.CreateRepositoryHook(ctx, parts[0], parts[1], hook); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", fullName, err))
			continue
		}
		created++
	}

	message := fmt.Sprintf("Registered webhooks on %d repository(ies)", created)
	if failed > 0 {
		message += fmt.Sprintf("; %d failed", failed)
	}

	status := http.StatusOK
	if created == 0 {
		status = http.StatusBadGateway
	}

	c.JSON(status, gin.H{
		"message":     message,
		"created":     created,
		"failed":      failed,
		"errors":      errors,
		"webhook_url": webhookURL,
	})
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
