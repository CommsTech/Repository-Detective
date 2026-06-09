package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/runner"
	"git.commsnet.org/commstech/bugbot/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RunnerHandler serves operator and runner callback endpoints.
type RunnerHandler struct {
	store    store.QueryStore
	cfg      runner.Config
	receiver *runner.Receiver
	registry *runner.Registry
	logger   *logrus.Logger
}

// NewRunnerHandler creates a runner API handler.
func NewRunnerHandler(s store.QueryStore, cfg runner.Config, receiver *runner.Receiver, registry *runner.Registry, logger *logrus.Logger) *RunnerHandler {
	return &RunnerHandler{store: s, cfg: cfg, receiver: receiver, registry: registry, logger: logger}
}

// RegisterOperatorRoutes mounts operator-facing runner job routes (API key auth applied by caller).
func (h *RunnerHandler) RegisterOperatorRoutes(g *gin.RouterGroup) {
	g.GET("/runner/jobs", h.ListRunnerJobs)
	g.GET("/runner/jobs/:job_id", h.GetRunnerJob)
	g.POST("/runner/jobs/:job_id/cancel", h.CancelRunnerJob)
	g.GET("/runner/workers", h.ListRunnerWorkers)
}

// RegisterRunnerRoutes mounts runner worker routes (HMAC auth applied by caller).
func (h *RunnerHandler) RegisterRunnerRoutes(g *gin.RouterGroup) {
	g.POST("/ping", h.PingRunner)
	g.POST("/jobs/claim", h.ClaimJob)
	g.GET("/jobs/:job_id/spec", h.GetJobSpec)
	g.POST("/jobs/:job_id/result", h.SubmitJobResult)
}

type runnerJobResponse struct {
	JobID        string          `json:"job_id"`
	RepositoryID int64           `json:"repository_id"`
	ScanID       string          `json:"scan_id,omitempty"`
	JobType      string          `json:"job_type"`
	Status       string          `json:"status"`
	RunnerMode   string          `json:"runner_mode"`
	Ref          string          `json:"ref"`
	CommitSHA    string          `json:"commit_sha,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	Summary      json.RawMessage `json:"summary,omitempty"`
}

func (h *RunnerHandler) ListRunnerJobs(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	opts := store.NormalizeListOptions(store.ListOptions{Limit: 50})
	jobs, err := h.store.ListRunnerJobs(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list runner jobs"})
		return
	}
	out := make([]runnerJobResponse, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, toRunnerJobResponse(job))
	}
	c.JSON(http.StatusOK, gin.H{"jobs": out})
}

func (h *RunnerHandler) GetRunnerJob(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	jobID := c.Param("job_id")
	job, err := h.store.GetRunnerJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runner job not found"})
		return
	}
	c.JSON(http.StatusOK, toRunnerJobResponse(job))
}

func (h *RunnerHandler) CancelRunnerJob(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	jobID := c.Param("job_id")
	if err := h.store.CancelRunnerJob(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, _ := h.store.GetRunnerJob(c.Request.Context(), jobID)
	c.JSON(http.StatusOK, toRunnerJobResponse(job))
}

func (h *RunnerHandler) ClaimJob(c *gin.Context) {
	if h.receiver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runner receiver disabled"})
		return
	}
	job, spec, err := h.receiver.ClaimNextJob(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no jobs available"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": toRunnerJobResponse(job), "spec": spec})
}

func (h *RunnerHandler) GetJobSpec(c *gin.Context) {
	if h.receiver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runner receiver disabled"})
		return
	}
	jobID := c.Param("job_id")
	_, spec, err := h.receiver.GetJobSpec(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, spec)
}

func (h *RunnerHandler) SubmitJobResult(c *gin.Context) {
	if h.receiver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runner receiver disabled"})
		return
	}
	jobID := c.Param("job_id")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var result runner.JobResult
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid result json"})
		return
	}
	if err := h.receiver.SubmitResult(c.Request.Context(), jobID, result); err != nil {
		status := http.StatusBadRequest
		switch {
		case strings.Contains(err.Error(), "signature"), strings.Contains(err.Error(), "unknown"):
			status = http.StatusUnauthorized
		case strings.Contains(err.Error(), "expired"):
			status = http.StatusGone
		case strings.Contains(err.Error(), "size"):
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "accepted", "job_id": jobID})
}

type runnerPingRequest struct {
	RunnerID     string   `json:"runner_id"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

func (h *RunnerHandler) PingRunner(c *gin.Context) {
	var body runnerPingRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.RunnerID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runner_id required"})
		return
	}
	if h.registry != nil {
		h.registry.RecordHeartbeat(runner.WorkerHeartbeat{
			RunnerID:     strings.TrimSpace(body.RunnerID),
			Version:      strings.TrimSpace(body.Version),
			Capabilities: body.Capabilities,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "delegation_enabled": h.cfg.DelegationEnabled})
}

func (h *RunnerHandler) ListRunnerWorkers(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	workers := []runner.WorkerHeartbeat{}
	if h.registry != nil {
		workers = h.registry.ListHeartbeats(15 * time.Minute)
	}
	c.JSON(http.StatusOK, gin.H{
		"delegation_enabled": h.cfg.DelegationEnabled,
		"workers":            workers,
	})
}

func (h *RunnerHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

func toRunnerJobResponse(job store.RunnerJob) runnerJobResponse {
	return runnerJobResponse{
		JobID: job.JobID, RepositoryID: job.RepositoryID, ScanID: job.ScanID,
		JobType: job.JobType, Status: job.Status, RunnerMode: job.RunnerMode,
		Ref: job.Ref, CommitSHA: job.CommitSHA, Error: job.Error,
		CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		StartedAt: job.StartedAt, FinishedAt: job.FinishedAt, ExpiresAt: job.ExpiresAt,
		Summary: job.ResultSummaryJSON,
	}
}

// RequireRunnerHMAC validates runner worker authentication headers.
func RequireRunnerHMAC(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "runner auth not configured"})
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		ts := c.GetHeader(runner.HeaderTimestamp)
		nonce := c.GetHeader(runner.HeaderNonce)
		sig := c.GetHeader(runner.HeaderSignature)
		if err := runner.VerifyRequest(secret, ts, nonce, sig, c.Request.Method, c.Request.URL.Path, body, time.Now().UTC()); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "runner authentication failed"})
			return
		}
		c.Set("runner_body", body)
		c.Next()
	}
}
