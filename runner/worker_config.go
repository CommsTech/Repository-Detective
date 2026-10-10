package runner

import (
	"fmt"
	"os"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/config/envcompat"
)

// WorkerConfig controls a native Repository Detective runner worker process.
type WorkerConfig struct {
	CoreURL         string
	SharedSecret    string
	RunnerID        string
	Version         string
	WorkspaceRoot   string
	PollInterval    time.Duration
	HeartbeatEvery  time.Duration
	JobTimeout      time.Duration
	MaxConcurrent   int
	AllowedJobTypes []string
	CloneTimeout    time.Duration
}

// LoadWorkerConfigFromEnv reads worker settings from environment variables.
func LoadWorkerConfigFromEnv() WorkerConfig {
	cfg := WorkerConfig{
		CoreURL:        envOr("CORE_URL", ""),
		SharedSecret:   envOr("RUNNER_SHARED_SECRET", ""),
		RunnerID:       envOr("RUNNER_ID", defaultRunnerID()),
		Version:        envOr("RUNNER_VERSION", "dev"),
		WorkspaceRoot:  envOr("WORKSPACE_ROOT", "/tmp/rd-runner-workspaces"),
		PollInterval:   durationEnv("RUNNER_POLL_INTERVAL_SECONDS", 5*time.Second),
		HeartbeatEvery: durationEnv("RUNNER_HEARTBEAT_SECONDS", 30*time.Second),
		JobTimeout:     durationEnv("RUNNER_JOB_TIMEOUT_SECONDS", 30*time.Minute),
		MaxConcurrent:  intEnv("RUNNER_MAX_CONCURRENT_JOBS", 1),
		CloneTimeout:   durationEnv("RUNNER_CLONE_TIMEOUT_SECONDS", 10*time.Minute),
	}
	if types := strings.TrimSpace(envOr("RUNNER_ALLOWED_JOB_TYPES", "")); types != "" {
		for _, t := range strings.Split(types, ",") {
			if v := strings.TrimSpace(t); v != "" {
				cfg.AllowedJobTypes = append(cfg.AllowedJobTypes, v)
			}
		}
	}
	return cfg.Normalized()
}

// Normalized applies defaults and validates worker config shape.
func (c WorkerConfig) Normalized() WorkerConfig {
	out := c
	if out.RunnerID == "" {
		out.RunnerID = defaultRunnerID()
	}
	if out.WorkspaceRoot == "" {
		out.WorkspaceRoot = "/tmp/rd-runner-workspaces"
	}
	if out.PollInterval <= 0 {
		out.PollInterval = 5 * time.Second
	}
	if out.HeartbeatEvery <= 0 {
		out.HeartbeatEvery = 30 * time.Second
	}
	if out.JobTimeout <= 0 {
		out.JobTimeout = 30 * time.Minute
	}
	if out.CloneTimeout <= 0 {
		out.CloneTimeout = 10 * time.Minute
	}
	if out.MaxConcurrent <= 0 {
		out.MaxConcurrent = 1
	}
	if len(out.AllowedJobTypes) == 0 {
		out.AllowedJobTypes = []string{JobTypeGraph, JobTypeSBOM, JobTypeRemediationVerify}
	}
	return out
}

// Validate returns an error when required worker settings are missing.
func (c WorkerConfig) Validate() error {
	c = c.Normalized()
	if strings.TrimSpace(c.CoreURL) == "" {
		return fmt.Errorf("core URL required (REPOSITORY_DETECTIVE_CORE_URL)")
	}
	if strings.TrimSpace(c.SharedSecret) == "" {
		return fmt.Errorf("runner shared secret required (REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET)")
	}
	return nil
}

func envOr(suffix, fallback string) string {
	if value, ok := envcompat.Resolve(suffix); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func durationEnv(suffix string, fallback time.Duration) time.Duration {
	raw := envOr(suffix, "")
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if sec := intEnv(strings.TrimSuffix(suffix, "_SECONDS"), 0); sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return fallback
}

func intEnv(suffix string, fallback int) int {
	raw := envOr(suffix, "")
	if raw == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err == nil {
		return n
	}
	return fallback
}

func defaultRunnerID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "rd-runner"
	}
	return "rd-runner-" + strings.TrimSpace(host)
}
