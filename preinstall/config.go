package preinstall

import (
	"time"

	"git.commsnet.org/commstech/bugbot/health"
	"git.commsnet.org/commstech/bugbot/graph"
)

// Config controls third-party pre-install audit behavior.
type Config struct {
	Enabled              bool
	AllowPrivateNetworks bool
	MaxRepoSizeMB        int
	MaxFiles             int
	TimeoutSeconds       int
	MaxFindings          int
	AllowGitClone        bool
	Health               health.Config
	Graph                graph.Config
}

// DefaultConfig returns safe Phase 9 defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		AllowPrivateNetworks: false,
		MaxRepoSizeMB:        500,
		MaxFiles:             5000,
		TimeoutSeconds:       600,
		MaxFindings:          200,
		AllowGitClone:        true,
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
