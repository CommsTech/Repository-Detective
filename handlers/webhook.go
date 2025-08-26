package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GiteaWebhookPayload represents the structure of Gitea webhook payloads
type GiteaWebhookPayload struct {
	Secret       string      `json:"secret"`
	Ref          string      `json:"ref"`
	Before      string      `json:"before"`
	After       string      `json:"after"`
	CompareURL  string      `json:"compare_url"`
	Commits     []Commit    `json:"commits"`
	Repository  Repository  `json:"repository"`
	Pusher      User        `json:"pusher"`
	Sender      User        `json:"sender"`
	Action      string      `json:"action"`
	PullRequest PullRequest `json:"pull_request,omitempty"`
}

// Commit represents a Git commit
type Commit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	URL       string   `json:"url"`
	Author    User     `json:"author"`
	Committer User     `json:"committer"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
}

// Repository represents a Gitea repository
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Owner       User   `json:"owner"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	GitURL      string `json:"git_url"`
	SSHURL      string `json:"ssh_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Size        int64  `json:"size"`
	Fork        bool   `json:"fork"`
	Archived    bool   `json:"archived"`
}

// User represents a Gitea user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
}

// PullRequest represents a Gitea pull request
type PullRequest struct {
	ID          int64  `json:"id"`
	Number      int    `json:"number"`
	State       string `json:"state"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	User        User   `json:"user"`
	HTMLURL     string `json:"html_url"`
	DiffURL     string `json:"diff_url"`
	PatchURL    string `json:"patch_url"`
	Mergeable   bool   `json:"mergeable"`
	Merged      bool   `json:"merged"`
	MergedAt    string `json:"merged_at"`
	MergedBy    User   `json:"merged_by"`
	BaseBranch  string `json:"base_branch"`
	HeadBranch  string `json:"head_branch"`
	BaseRepo    Repository `json:"base_repo"`
	HeadRepo    Repository `json:"head_repo"`
}

// WebhookHandler handles incoming Gitea webhooks
type WebhookHandler struct {
	logger *logrus.Logger
	config *Config
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(logger *logrus.Logger, config *Config) *WebhookHandler {
	return &WebhookHandler{
		logger: logger,
		config: config,
	}
}

// HandleWebhook processes incoming webhook requests
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	h.logger.Info("Received webhook request")

	// Verify webhook secret if configured
	if err := h.verifyWebhookSecret(c); err != nil {
		h.logger.Errorf("Webhook secret verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse webhook payload
	var payload GiteaWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Errorf("Failed to parse webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	h.logger.Infof("Processing webhook for repository: %s, action: %s", 
		payload.Repository.FullName, payload.Action)

	// Process webhook based on action type
	switch payload.Action {
	case "push":
		h.handlePushEvent(c, &payload)
	case "pull_request":
		h.handlePullRequestEvent(c, &payload)
	case "issues":
		h.handleIssueEvent(c, &payload)
	default:
		h.logger.Infof("Unhandled webhook action: %s", payload.Action)
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
	}
}

// handlePushEvent processes push events
func (h *WebhookHandler) handlePushEvent(c *gin.Context, payload *GiteaWebhookPayload) {
	h.logger.Infof("Processing push event for repository: %s", payload.Repository.FullName)

	// Skip if no commits
	if len(payload.Commits) == 0 {
		h.logger.Info("No commits in push event, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "no commits"})
		return
	}

	// Analyze each commit
	for _, commit := range payload.Commits {
		if err := h.analyzeCommit(&commit, payload.Repository); err != nil {
			h.logger.Errorf("Failed to analyze commit %s: %v", commit.ID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// handlePullRequestEvent processes pull request events
func (h *WebhookHandler) handlePullRequestEvent(c *gin.Context, payload *GiteaWebhookPayload) {
	h.logger.Infof("Processing pull request event: %s #%d", 
		payload.Repository.FullName, payload.PullRequest.Number)

	// Only process on open or synchronize
	if payload.PullRequest.State != "open" {
		h.logger.Infof("Pull request is not open, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// Analyze pull request
	if err := h.analyzePullRequest(&payload.PullRequest, payload.Repository); err != nil {
		h.logger.Errorf("Failed to analyze pull request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analysis failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// handleIssueEvent processes issue events
func (h *WebhookHandler) handleIssueEvent(c *gin.Context, payload *GiteaWebhookPayload) {
	h.logger.Infof("Processing issue event for repository: %s", payload.Repository.FullName)
	// Issue events are typically not relevant for code analysis
	c.JSON(http.StatusOK, gin.H{"status": "ignored"})
}

// verifyWebhookSecret verifies the webhook secret if configured
func (h *WebhookHandler) verifyWebhookSecret(c *gin.Context) error {
	// If no secret is configured, skip verification
	if h.config.WebhookSecret == "" {
		return nil
	}

	// Get secret from header or query parameter
	secret := c.GetHeader("X-Gitea-Signature")
	if secret == "" {
		secret = c.Query("secret")
	}

	if secret != h.config.WebhookSecret {
		return fmt.Errorf("invalid webhook secret")
	}

	return nil
}

// analyzeCommit analyzes a single commit for potential issues
func (h *WebhookHandler) analyzeCommit(commit *Commit, repo Repository) error {
	h.logger.Infof("Analyzing commit %s in repository %s", commit.ID[:8], repo.FullName)

	// Create analysis context
	ctx := &AnalysisContext{
		Repository: repo,
		Commit:     commit,
		Type:       "commit",
	}

	// Start analysis in background
	go func() {
		if err := h.performAnalysis(ctx); err != nil {
			h.logger.Errorf("Analysis failed for commit %s: %v", commit.ID[:8], err)
		}
	}()

	return nil
}

// analyzePullRequest analyzes a pull request for potential issues
func (h *WebhookHandler) analyzePullRequest(pr *PullRequest, repo Repository) error {
	h.logger.Infof("Analyzing pull request #%d in repository %s", pr.Number, repo.FullName)

	// Create analysis context
	ctx := &AnalysisContext{
		Repository:   repo,
		PullRequest:  pr,
		Type:         "pull_request",
	}

	// Start analysis in background
	go func() {
		if err := h.performAnalysis(ctx); err != nil {
			h.logger.Errorf("Analysis failed for PR #%d: %v", pr.Number, err)
		}
	}()

	return nil
}

// performAnalysis performs the actual code analysis
func (h *WebhookHandler) performAnalysis(ctx *AnalysisContext) error {
	h.logger.Infof("Starting analysis for %s in repository %s", ctx.Type, ctx.Repository.FullName)

	// TODO: Implement actual code analysis logic
	// 1. Fetch repository content
	// 2. Analyze code using OpenWebUI AI
	// 3. Create issues for found problems
	// 4. Propose fixes

	return nil
}

// AnalysisContext holds context for code analysis
type AnalysisContext struct {
	Repository   Repository
	Commit       *Commit
	PullRequest  *PullRequest
	Type         string
}

// Config holds webhook handler configuration
type Config struct {
	WebhookSecret string
	// Add other configuration fields as needed
}
