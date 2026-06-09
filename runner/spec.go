package runner

import (
	"git.commsnet.org/commstech/bugbot/analyzers"
	"git.commsnet.org/commstech/bugbot/store"
)

// Config controls runner delegation on the core service.
type Config struct {
	DelegationEnabled     bool
	Mode                  string
	SharedSecret          string
	JobTimeoutSeconds     int
	MaxConcurrentJobs     int
	ResultMaxSizeMB       int
	ArtifactRetentionDays int
	CallbackBaseURL       string
	MaxRepoSizeMB         int
	MaxFiles              int
	RequireHMAC           bool
	NonceTTLSeconds       int
	AllowedJobTypes       []string
}

// Normalized returns config with defaults applied.
func (c Config) Normalized() Config {
	out := c
	if out.Mode == "" {
		out.Mode = ModeCore
	}
	if out.JobTimeoutSeconds <= 0 {
		out.JobTimeoutSeconds = 900
	}
	if out.MaxConcurrentJobs <= 0 {
		out.MaxConcurrentJobs = 2
	}
	if out.ResultMaxSizeMB <= 0 {
		out.ResultMaxSizeMB = 50
	}
	if out.ArtifactRetentionDays <= 0 {
		out.ArtifactRetentionDays = 14
	}
	if out.MaxRepoSizeMB <= 0 {
		out.MaxRepoSizeMB = 500
	}
	if out.MaxFiles <= 0 {
		out.MaxFiles = 5000
	}
	if out.NonceTTLSeconds <= 0 {
		out.NonceTTLSeconds = 300
	}
	if len(out.AllowedJobTypes) == 0 {
		out.AllowedJobTypes = []string{
			JobTypeScanFullRepo, JobTypeScanFullRepoLegacy, JobTypeSBOM,
			JobTypeGraph, JobTypePreinstallAudit, JobTypeRemediationVerify,
		}
	}
	return out
}

// StartupValid reports whether runner delegation can be enabled safely.
func (c Config) StartupValid() error {
	cfg := c.Normalized()
	if !cfg.DelegationEnabled {
		return nil
	}
	if cfg.Mode == ModeCore {
		return nil
	}
	if cfg.SharedSecret == "" {
		return ErrSharedSecretRequired
	}
	return nil
}

// RepositoryInfo is forge metadata exposed to runners.
type RepositoryInfo struct {
	ForgeType     string `json:"forge_type"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
}

// JobLimits caps runner work.
type JobLimits struct {
	TimeoutSeconds  int `json:"timeout_seconds"`
	MaxRepoSizeMB   int `json:"max_repo_size_mb"`
	MaxFiles        int `json:"max_files"`
	MaxResultSizeMB int `json:"max_result_size_mb"`
}

// JobSpec is the signed contract sent to a runner.
type JobSpec struct {
	Version           int                      `json:"version"`
	JobID             string                   `json:"job_id"`
	JobType           string                   `json:"job_type"`
	Repository        RepositoryInfo           `json:"repository"`
	Ref               string                   `json:"ref"`
	CommitSHA         string                   `json:"commit_sha,omitempty"`
	ScanID            string                   `json:"scan_id"`
	EffectiveSettings analyzers.PolicySnapshot `json:"effective_settings"`
	Limits            JobLimits                `json:"limits"`
	AllowedTasks      []string                 `json:"allowed_tasks"`
	ForbiddenTasks    []string                 `json:"forbidden_tasks"`
	CallbackBaseURL   string                   `json:"callback_base_url"`
}

// BuildJobSpec constructs a runner job spec from core context.
func BuildJobSpec(cfg Config, jobID string, repo store.Repository, scanID, ref, commitSHA string, policy analyzers.PolicySnapshot) JobSpec {
	cfg = cfg.Normalized()
	return JobSpec{
		Version:           ContractVersion,
		JobID:             jobID,
		JobType:           JobTypeScanFullRepo,
		Repository: RepositoryInfo{
			ForgeType:     repo.ForgeType,
			Owner:         repo.Owner,
			Name:          repo.Name,
			FullName:      repo.FullName,
			CloneURL:      repo.CloneURL,
			DefaultBranch: repo.DefaultBranch,
		},
		Ref:               ref,
		CommitSHA:         commitSHA,
		ScanID:            scanID,
		EffectiveSettings: policy,
		Limits: JobLimits{
			TimeoutSeconds:  cfg.JobTimeoutSeconds,
			MaxRepoSizeMB:   cfg.MaxRepoSizeMB,
			MaxFiles:        cfg.MaxFiles,
			MaxResultSizeMB: cfg.ResultMaxSizeMB,
		},
		AllowedTasks:    append([]string(nil), AllowedTasks...),
		ForbiddenTasks:  append([]string(nil), ForbiddenTasks...),
		CallbackBaseURL: cfg.CallbackBaseURL,
	}
}
