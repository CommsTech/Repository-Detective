package scanners

import "strings"

// Workspace mode constants.
const (
	WorkspaceModeAPI     = "api"
	WorkspaceModeArchive = "archive"
	WorkspaceModeAuto    = "auto"
)

// WorkspaceConfig controls how scanner workspaces are built.
type WorkspaceConfig struct {
	Mode                   string
	MaxSizeMB              int
	MaxFiles               int
	ArchiveTimeoutSeconds  int
	DefaultAnalysisTimeout int
}

// DefaultWorkspaceConfig returns backward-compatible defaults (API mode).
func DefaultWorkspaceConfig() WorkspaceConfig {
	return WorkspaceConfig{
		Mode:                   WorkspaceModeAPI,
		MaxSizeMB:              500,
		MaxFiles:               5000,
		ArchiveTimeoutSeconds:  0,
		DefaultAnalysisTimeout: 300,
	}
}

// NormalizedMode returns a supported workspace mode, defaulting to api.
func (c WorkspaceConfig) NormalizedMode() string {
	switch strings.ToLower(strings.TrimSpace(c.Mode)) {
	case WorkspaceModeArchive, WorkspaceModeAuto:
		return strings.ToLower(strings.TrimSpace(c.Mode))
	default:
		return WorkspaceModeAPI
	}
}

func (c WorkspaceConfig) maxArchiveBytes() int64 {
	mb := c.MaxSizeMB
	if mb <= 0 {
		mb = 500
	}
	return int64(mb) * 1024 * 1024
}

func (c WorkspaceConfig) maxFiles() int {
	if c.MaxFiles <= 0 {
		return 5000
	}
	return c.MaxFiles
}

func (c WorkspaceConfig) archiveTimeoutSeconds() int {
	if c.ArchiveTimeoutSeconds > 0 {
		return c.ArchiveTimeoutSeconds
	}
	if c.DefaultAnalysisTimeout > 0 {
		return c.DefaultAnalysisTimeout
	}
	return 300
}
