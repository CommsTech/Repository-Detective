package preinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SandboxMeta records isolation guarantees for an audit workspace.
type SandboxMeta struct {
	SandboxID          string
	Enabled            bool
	CloneMode          string
	SubmodulesDisabled bool
	MaxRepoSizeMB      int
	MaxFiles           int
	MaxFileSizeMB      int
	TimeoutSeconds     int
	PrivateIPBlocked   bool
	ReadOnlyWorkspace  bool
	RetainOnFailure    bool
	NetworkMode        string
	WorkspacePath      string
}

func sandboxMetaFromConfig(cfg Config, sandboxID, workspace string) SandboxMeta {
	return SandboxMeta{
		SandboxID:          sandboxID,
		Enabled:            cfg.SandboxEnabled,
		CloneMode:          "shallow-single-branch-no-tags",
		SubmodulesDisabled: !cfg.SandboxAllowSubmodules,
		MaxRepoSizeMB:      cfg.MaxRepoSizeMB,
		MaxFiles:           cfg.MaxFiles,
		MaxFileSizeMB:      cfg.SandboxMaxFileSizeMB,
		TimeoutSeconds:     cfg.TimeoutSeconds,
		PrivateIPBlocked:   !cfg.AllowPrivateNetworks,
		ReadOnlyWorkspace:  cfg.SandboxReadonlyWorkspace,
		RetainOnFailure:    cfg.SandboxRetainOnFailure,
		NetworkMode:        cfg.SandboxNetworkMode,
		WorkspacePath:      workspace,
	}
}

func (cfg Config) maxFileSizeBytes() int64 {
	if cfg.SandboxMaxFileSizeMB <= 0 {
		return 25 * 1024 * 1024
	}
	return int64(cfg.SandboxMaxFileSizeMB) * 1024 * 1024
}

func validateWorkspacePath(root, path string) error {
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, filepath.Clean(root)+string(os.PathSeparator)) && clean != filepath.Clean(root) {
		return fmt.Errorf("path escapes sandbox root")
	}
	if strings.Contains(clean, "..") {
		return fmt.Errorf("path traversal blocked")
	}
	return nil
}

// ValidateWorkspacePathForTest exposes sandbox path checks for unit tests.
func ValidateWorkspacePathForTest(root, path string) error {
	return validateWorkspacePath(root, path)
}

// MeasureWorkspaceSandboxForTest exposes sandbox workspace measurement for unit tests.
func MeasureWorkspaceSandboxForTest(root string, cfg Config) (int64, int, error) {
	return measureWorkspaceSandbox(root, cfg)
}

func measureWorkspaceSandbox(root string, cfg Config) (int64, int, error) {
	maxBytes := cfg.maxRepoSizeBytes()
	maxFiles := cfg.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 5000
	}
	maxFile := cfg.maxFileSizeBytes()
	var total int64
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := validateWorkspacePath(root, path); err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		count++
		if count > maxFiles {
			return fmt.Errorf("repository exceeds max file count (%d)", maxFiles)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxFile {
			return fmt.Errorf("file exceeds max size (%d MB)", cfg.SandboxMaxFileSizeMB)
		}
		total += info.Size()
		if total > maxBytes {
			return fmt.Errorf("repository exceeds max size (%d MB)", cfg.MaxRepoSizeMB)
		}
		return nil
	})
	return total, count, err
}
