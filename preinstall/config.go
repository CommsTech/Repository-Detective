package preinstall

import (
	"time"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
)

// Config controls third-party pre-install audit behavior.
type Config struct {
	Enabled                       bool
	AllowPrivateNetworks          bool
	MaxRepoSizeMB                 int
	MaxFiles                      int
	TimeoutSeconds                int
	MaxFindings                   int
	AllowGitClone                 bool
	ReportIncludeProjectLink      bool
	RepositoryDetectiveProjectURL string
	SandboxEnabled                bool
	SandboxRetainOnFailure        bool
	SandboxMaxFileSizeMB          int
	SandboxAllowSubmodules        bool
	SandboxNetworkMode            string
	SandboxReadonlyWorkspace      bool
	Health                        health.Config
	Graph                         graph.Config
}

// DefaultConfig returns safe Phase 9 defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:                       true,
		AllowPrivateNetworks:          false,
		MaxRepoSizeMB:                 500,
		MaxFiles:                      5000,
		TimeoutSeconds:                600,
		MaxFindings:                   200,
		AllowGitClone:                 true,
		ReportIncludeProjectLink:      true,
		RepositoryDetectiveProjectURL: "https://git.commsnet.org/commstech/repository-detective",
		SandboxEnabled:                true,
		SandboxRetainOnFailure:        false,
		SandboxMaxFileSizeMB:          25,
		SandboxAllowSubmodules:        false,
		SandboxNetworkMode:            "restricted",
		SandboxReadonlyWorkspace:      true,
	}
}

func (c Config) maxRepoSizeBytes() int64 {
	if c.MaxRepoSizeMB <= 0 {
		return 500 * 1024 * 1024
	}
	return int64(c.MaxRepoSizeMB) * 1024 * 1024
}

func (c Config) auditTimeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}
