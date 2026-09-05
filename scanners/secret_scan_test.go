package scanners_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestResolveSecretScanModesTreeOnlyWhenHistoryDisabled(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cfg.SecretScanGitHistoryEnabled = false
	modes := scanners.ResolveSecretScanModes(cfg, false, 3)
	if !modes.Tree || modes.GitHistory || modes.RecentCommits || modes.ChangedFiles {
		t.Fatalf("expected tree only, got %+v", modes)
	}
}

func TestResolveSecretScanModesFullHistoryOnDeepScan(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cfg.SecretScanGitHistoryEnabled = true
	modes := scanners.ResolveSecretScanModes(cfg, false, 2)
	if !modes.Tree || !modes.GitHistory || modes.ChangedFiles {
		t.Fatalf("expected tree + git history, got %+v", modes)
	}
}

func TestResolveSecretScanModesScopedUsesChangedFiles(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cfg.SecretScanGitHistoryEnabled = true
	cfg.SecretScanRecentCommitsMax = 10
	modes := scanners.ResolveSecretScanModes(cfg, true, 2)
	if !modes.Tree || !modes.ChangedFiles || !modes.RecentCommits {
		t.Fatalf("expected scoped modes, got %+v", modes)
	}
	if modes.GitHistory {
		t.Fatal("scoped scan must not claim full git history")
	}
}

func TestResolveSecretScanModesShallowDepthSkipsHistory(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cfg.SecretScanGitHistoryEnabled = true
	modes := scanners.ResolveSecretScanModes(cfg, false, 1)
	if modes.GitHistory || modes.RecentCommits {
		t.Fatalf("depth 1 should not run history, got %+v", modes)
	}
}
