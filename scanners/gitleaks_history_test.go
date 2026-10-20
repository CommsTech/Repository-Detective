package scanners_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

const historyFoundJSON = `[
  {
    "RuleID": "generic-api-key",
    "Description": "Generic API Key",
    "StartLine": 3,
    "EndLine": 3,
    "Match": "api_key=REDACTED_HIST",
    "Secret": "REDACTED_HIST",
    "File": "old/config.env",
    "Commit": "abc123def4567890abcdef1234567890abcdef12",
    "Entropy": 4.2,
    "Fingerprint": "hist-fp-001"
  }
]`

func TestAnnotateHistoryFindingRedactedAndLabeled(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}

	findings, err := scanners.ParseGitleaksOutputForTest([]byte(historyFoundJSON), dir)
	if err != nil {
		t.Fatal(err)
	}
	annotated := scanners.AnnotateHistoryFindingsForTest(findings, scanners.SecretScopeGitHistory, dir, true)
	if len(annotated) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(annotated))
	}
	f := annotated[0]
	if f.Source != scanners.HistoryScannerName {
		t.Fatalf("expected %s source, got %q", scanners.HistoryScannerName, f.Source)
	}
	if !strings.Contains(f.Description, "historical secret") {
		t.Fatalf("expected historical label in description: %q", f.Description)
	}
	if !strings.Contains(f.Description, "not in current tree") {
		t.Fatalf("expected not-in-tree note: %q", f.Description)
	}
	if strings.Contains(f.Description, "REDACTED_HIST") {
		t.Fatal("raw secret must not appear in description")
	}
	if !strings.Contains(f.Description, scanners.RemediationRotateGuidance) {
		t.Fatal("expected remediation guidance")
	}
}

func TestGitleaksHistoryArgsUseDetectMode(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.SecretScanRedact = true
	args := scanners.GitleaksHistoryArgsForTest("/tmp/repo", cfg, scanners.SecretScopeGitHistory, "/tmp/out.json")
	if args[0] != "detect" {
		t.Fatalf("expected detect subcommand, got %v", args)
	}
	if !containsArg(args, "--source", "/tmp/repo") {
		t.Fatalf("expected --source /tmp/repo in %v", args)
	}
	if !containsArg(args, "--redact") {
		t.Fatal("expected --redact when SecretScanRedact is true")
	}
}

func TestGitleaksHistoryRecentCommitsLogOpts(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.SecretScanRecentCommitsMax = 25
	args := scanners.GitleaksHistoryArgsForTest("/tmp/repo", cfg, scanners.SecretScopeRecentCommits, "/tmp/out.json")
	if !containsArgPair(args, "--log-opts", "-n 25") {
		t.Fatalf("expected recent commits log-opts in %v", args)
	}
}

func TestChangedFileModeDoesNotClaimFullHistory(t *testing.T) {
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cfg.SecretScanGitHistoryEnabled = true
	cfg.SecretScanRecentCommitsMax = 5
	modes := scanners.ResolveSecretScanModes(cfg, true, 3)
	if modes.GitHistory {
		t.Fatal("changed-file scoped scan must not enable full history mode")
	}
}

func TestCurrentTreeSecretHigherPriorityWhenPresent(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "old", "config.env")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, []byte("api_key=x"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := scanners.ParseGitleaksOutputForTest([]byte(historyFoundJSON), dir)
	if err != nil {
		t.Fatal(err)
	}
	annotated := scanners.AnnotateHistoryFindingsForTest(findings, scanners.SecretScopeGitHistory, dir, true)
	if annotated[0].Severity != "critical" {
		t.Fatalf("expected critical when also in current tree, got %q", annotated[0].Severity)
	}
	if !strings.Contains(annotated[0].Description, "also present in current tree") {
		t.Fatal("expected in-current-tree note")
	}
}

func containsArg(args []string, want ...string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == want[0] {
			if len(want) == 1 {
				return true
			}
			if i+1 < len(args) && args[i+1] == want[1] {
				return true
			}
		}
	}
	return false
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
