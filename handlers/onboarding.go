package handlers

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/gitea"
	"git.commsnet.org/commstech/bugbot/web"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// OnboardingHandler serves the setup UI and onboarding API.
type OnboardingHandler struct {
	logger    *logrus.Logger
	giteaURL  string
	publicURL string
	aiConfig  ai.Config
}

// OnboardingConfig holds server-side defaults for the onboarding UI.
type OnboardingConfig struct {
	GiteaURL  string
	PublicURL string
	AIConfig  ai.Config
}

// NewOnboardingHandler creates an onboarding handler.
func NewOnboardingHandler(logger *logrus.Logger, cfg OnboardingConfig) *OnboardingHandler {
	return &OnboardingHandler{
		logger:    logger,
		giteaURL:  cfg.GiteaURL,
		publicURL: cfg.PublicURL,
		aiConfig:  cfg.AIConfig,
	}
}

// RegisterRoutes mounts onboarding UI and API routes.
func (h *OnboardingHandler) RegisterRoutes(router *gin.Engine, api *gin.RouterGroup) {
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		h.logger.Errorf("Failed to load onboarding static files: %v", err)
		return
	}

	router.GET("/onboard", func(c *gin.Context) {
		c.FileFromFS("index.html", http.FS(staticFS))
	})
	router.StaticFS("/onboard/static", http.FS(staticFS))

	api.GET("/onboard/defaults", h.handleDefaults)
	api.POST("/onboard/test-gitea", h.handleTestGitea)
	api.POST("/onboard/test-ai", h.handleTestAI)
	api.POST("/onboard/repos", h.handleListRepos)
	api.POST("/onboard/webhooks", h.handleRegisterWebhooks)
}

func (h *OnboardingHandler) handleDefaults(c *gin.Context) {
	webhookURL := strings.TrimSuffix(h.publicURL, "/") + "/webhook"
	c.JSON(http.StatusOK, gin.H{
		"gitea_url":    h.giteaURL,
		"public_url":   h.publicURL,
		"webhook_url":  webhookURL,
		"ai_provider":  h.aiConfig.Provider,
		"ai_model":     h.aiConfig.Model,
		"ai_base_url":  h.aiConfig.BaseURL,
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	repos, err := client.ListUserRepositories(ctx, 100)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"repositories": repos})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bugbot public URL is required for webhooks"})
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
		"message": message,
		"created": created,
		"failed":  failed,
		"errors":  errors,
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
