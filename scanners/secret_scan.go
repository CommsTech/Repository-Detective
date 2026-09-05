package scanners

import (
	"os"
	"path/filepath"
	"strings"
)

// Secret scan scope labels persisted in finding metadata.
const (
	SecretScopeCurrentTree   = "current_tree"
	SecretScopeGitHistory    = "git_history"
	SecretScopeRecentCommits = "recent_commits"
	SecretScopeChangedFiles  = "changed_files"
)

// SecretScanModes selects which gitleaks scan modes run for a repository scan.
type SecretScanModes struct {
	Tree          bool
	GitHistory    bool
	RecentCommits bool
	ChangedFiles  bool
}

// ResolveSecretScanModes picks secret scan modes from config and scan context.
// scopedScan is true when the scan targets specific changed files (PR/push quick scan).
func ResolveSecretScanModes(cfg Config, scopedScan bool, analysisDepth int) SecretScanModes {
	if !cfg.EnableGitleaks {
		return SecretScanModes{}
	}
	modes := SecretScanModes{Tree: true}
	if scopedScan {
		modes.ChangedFiles = true
		if cfg.SecretScanRecentCommitsMax > 0 {
			modes.RecentCommits = true
		}
		return modes
	}
	if !cfg.SecretScanGitHistoryEnabled {
		return modes
	}
	if analysisDepth < 2 {
		return modes
	}
	if cfg.SecretScanRecentCommitsMax > 0 && cfg.SecretScanHistoryMaxCommits > 0 {
		modes.RecentCommits = true
	} else {
		modes.GitHistory = true
	}
	return modes
}

// HistoryScannerName is the RunResult.Scanner value for git-history secret scans.
const HistoryScannerName = "gitleaks-history"

// TreeScannerName is the RunResult.Scanner value for current-tree secret scans.
const TreeScannerName = "gitleaks"

// RemediationRotateGuidance is appended to historical secret findings.
const RemediationRotateGuidance = "Rotate or revoke this credential even if it was removed from the current tree; it may still be valid or exposed in forks and caches."

func secretScopeLabel(scope string) string {
	switch scope {
	case SecretScopeCurrentTree:
		return "current-tree secret"
	case SecretScopeGitHistory:
		return "historical secret (full Git history)"
	case SecretScopeRecentCommits:
		return "historical secret (recent commits)"
	case SecretScopeChangedFiles:
		return "historical secret (changed files / scoped scan)"
	default:
		return scope
	}
}

func fileExistsInTree(treeRoot, relPath string) bool {
	if treeRoot == "" || relPath == "" {
		return false
	}
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = strings.TrimPrefix(relPath, "\\")
	if relPath == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(treeRoot, relPath))
	return err == nil
}
