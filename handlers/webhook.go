package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// rateLimiters per IP (bounded map to avoid unbounded growth).
var (
	rateLimiterMap  = make(map[string]*rate.Limiter)
	rateLimiterMu   sync.Mutex
	defaultRate     = rate.Limit(10) // 10 requests per second
	defaultBurst    = 20
	maxRateLimiters = 4096
)

func getRateLimiter(ip string) *rate.Limiter {
	rateLimiterMu.Lock()
	defer rateLimiterMu.Unlock()
	if len(rateLimiterMap) >= maxRateLimiters {
		for key := range rateLimiterMap {
			delete(rateLimiterMap, key)
			if len(rateLimiterMap) < maxRateLimiters/2 {
				break
			}
		}
	}
	if _, exists := rateLimiterMap[ip]; !exists {
		rateLimiterMap[ip] = rate.NewLimiter(defaultRate, defaultBurst)
	}
	return rateLimiterMap[ip]
}

// GiteaWebhookPayload represents the structure of Gitea webhook payloads
type GiteaWebhookPayload struct {
	Secret      string      `json:"secret"`
	Ref         string      `json:"ref"`
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

// LoginName returns the Gitea username from either login or username field.
func (u User) LoginName() string {
	if u.Username != "" {
		return u.Username
	}
	return u.Login
}

// PullRequest represents a Gitea pull request
type PullRequest struct {
	ID         int64             `json:"id"`
	Number     int               `json:"number"`
	State      string            `json:"state"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	User       User              `json:"user"`
	HTMLURL    string            `json:"html_url"`
	DiffURL    string            `json:"diff_url"`
	PatchURL   string            `json:"patch_url"`
	Mergeable  bool              `json:"mergeable"`
	Merged     bool              `json:"merged"`
	MergedAt   string            `json:"merged_at"`
	MergedBy   User              `json:"merged_by"`
	BaseBranch string            `json:"base_branch"`
	HeadBranch string            `json:"head_branch"`
	Head       PullRequestGitRef `json:"head"`
	BaseRepo   Repository        `json:"base_repo"`
	HeadRepo   Repository        `json:"head_repo"`
}

// PullRequestGitRef identifies the head commit/branch of a pull request webhook payload.
type PullRequestGitRef struct {
	SHA string `json:"sha"`
	Ref string `json:"ref"`
}

// AnalysisProcessor runs repository analysis for webhook events.
type AnalysisProcessor interface {
	ProcessPush(ctx context.Context, payload *GiteaWebhookPayload)
	ProcessPullRequest(ctx context.Context, payload *GiteaWebhookPayload)
}

// DeliveryRecorder records sanitized webhook acceptance (optional).
type DeliveryRecorder func(eventKind, repository, commitSHA, deliveryID string, prNumber int)

// WebhookHandler handles incoming Gitea webhooks
type WebhookHandler struct {
	logger    *logrus.Logger
	config    *Config
	processor AnalysisProcessor
	recorder  DeliveryRecorder
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(logger *logrus.Logger, config *Config, processor AnalysisProcessor) *WebhookHandler {
	return &WebhookHandler{
		logger:    logger,
		config:    config,
		processor: processor,
	}
}

// SetDeliveryRecorder attaches optional webhook delivery evidence persistence.
func (h *WebhookHandler) SetDeliveryRecorder(r DeliveryRecorder) {
	h.recorder = r
}

// HandleWebhook processes incoming webhook requests
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	ip := c.ClientIP()
	limiter := getRateLimiter(ip)
	if !limiter.Allow() {
		h.logger.Warnf("Rate limit exceeded for IP: %s", ip)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
		return
	}

	h.logger.Info("Received webhook request")

	body, err := readRequestBody(c)
	if err != nil {
		h.logger.Errorf("Failed to read webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	if err := h.verifyWebhookSecret(c, body); err != nil {
		h.logger.Errorf("Webhook secret verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var payload GiteaWebhookPayload
	if err := bindJSONBody(body, &payload); err != nil {
		h.logger.Errorf("Failed to parse webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// JSON body secret is not a supported auth mechanism — rely on HMAC header only.

	h.logger.Infof("Processing webhook for repository: %s, action: %q",
		security.RedactLogField(payload.Repository.FullName, 200), security.RedactLogField(payload.Action, 50))

	if !RepoAllowed(payload.Repository.FullName, h.config.IncludePatterns, h.config.ExcludePatterns) {
		h.logger.Infof("Repository %s skipped by include/exclude filters", security.RedactLogField(payload.Repository.FullName, 200))
		c.JSON(http.StatusOK, gin.H{"status": "filtered"})
		return
	}

	switch classifyWebhookEvent(&payload) {
	case webhookEventPush:
		h.recordDelivery("push", &payload, c)
		h.handlePushEvent(c, &payload)
	case webhookEventPullRequest:
		h.recordDelivery("pull_request", &payload, c)
		h.handlePullRequestEvent(c, &payload)
	default:
		h.logger.Infof("Unhandled webhook event (action=%q)", security.RedactLogField(payload.Action, 50))
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
	}
}

func (h *WebhookHandler) recordDelivery(kind string, payload *GiteaWebhookPayload, c *gin.Context) {
	if h.recorder == nil || payload == nil {
		return
	}
	deliveryID := c.GetHeader("X-Gitea-Delivery")
	if deliveryID == "" {
		deliveryID = c.GetHeader("X-GitHub-Delivery")
	}
	sha := payload.After
	if sha == "" && payload.PullRequest.Head.SHA != "" {
		sha = payload.PullRequest.Head.SHA
	}
	h.recorder(kind, payload.Repository.FullName, sha, deliveryID, payload.PullRequest.Number)
}

type webhookEventKind int

const (
	webhookEventUnknown webhookEventKind = iota
	webhookEventPush
	webhookEventPullRequest
)

// classifyWebhookEvent detects event type from payload shape.
// Gitea push hooks omit action; PR hooks use action=opened|synchronized|...
func classifyWebhookEvent(payload *GiteaWebhookPayload) webhookEventKind {
	if payload.PullRequest.Number > 0 || payload.PullRequest.ID > 0 {
		return webhookEventPullRequest
	}
	if len(payload.Commits) > 0 || (payload.Ref != "" && payload.After != "") {
		return webhookEventPush
	}
	return webhookEventUnknown
}

func readRequestBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func bindJSONBody(body []byte, payload *GiteaWebhookPayload) error {
	return json.Unmarshal(body, payload)
}

// handlePushEvent processes push events
func (h *WebhookHandler) handlePushEvent(c *gin.Context, payload *GiteaWebhookPayload) {
	h.logger.Infof("Processing push event for repository: %s", payload.Repository.FullName)

	if len(payload.Commits) == 0 {
		h.logger.Info("No commits in push event, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "no commits"})
		return
	}

	go h.processor.ProcessPush(c.Request.Context(), payload)

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// handlePullRequestEvent processes pull request events
func (h *WebhookHandler) handlePullRequestEvent(c *gin.Context, payload *GiteaWebhookPayload) {
	h.logger.Infof("Processing pull request event: %s #%d action=%q",
		payload.Repository.FullName, payload.PullRequest.Number, payload.Action)

	if payload.PullRequest.State != "open" {
		h.logger.Infof("Pull request is not open, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	switch payload.Action {
	case "closed", "closed_merged", "closed_unmerged":
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	go h.processor.ProcessPullRequest(c.Request.Context(), payload)

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// verifyWebhookSecret validates Gitea webhook authenticity.
// Gitea signs the raw body with HMAC-SHA256 and sends hex digest in X-Gitea-Signature.
func (h *WebhookHandler) verifyWebhookSecret(c *gin.Context, body []byte) error {
	if h.config.WebhookSecret == "" {
		if h.config.AllowInsecureWebhooks {
			h.logger.Warnf("Webhook secret is empty — accepting webhooks because allow_insecure_webhooks is enabled (development only)")
			return nil
		}
		return fmt.Errorf("webhook secret is not configured")
	}

	signature := c.GetHeader("X-Gitea-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature-256")
	}
	if signature == "" {
		return fmt.Errorf("missing webhook signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	signature = strings.TrimSpace(signature)

	mac := hmac.New(sha256.New, []byte(h.config.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return fmt.Errorf("invalid webhook signature")
	}
	return nil
}

// Config holds webhook handler configuration
type Config struct {
	WebhookSecret         string
	AllowInsecureWebhooks bool
	IncludePatterns       []string
	ExcludePatterns       []string
}
