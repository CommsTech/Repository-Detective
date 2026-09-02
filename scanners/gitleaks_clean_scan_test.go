package scanners

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gitleaks runs with the scan workspace as its working directory, so a relative
// config path resolved inside the repository under scan. gitleaks then aborted
// with "unable to load gitleaks config" and wrote no report, which silently
// disabled secret scanning for every repo lacking that file.
func TestGitleaksArgsResolveConfigToAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "gitleaks.toml")
	if err := os.WriteFile(configPath, []byte("title = \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workdir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	args := gitleaksArgs("/scan/workspace", Config{GitleaksConfig: "gitleaks.toml"}, "/tmp/report.json")

	idx := indexOfArg(args, "--config")
	if idx < 0 {
		t.Fatal("expected --config to be passed when the file exists")
	}
	if got := args[idx+1]; !filepath.IsAbs(got) {
		t.Fatalf("config path must be absolute, got %q", got)
	}
}

func TestGitleaksArgsDropUnusableConfig(t *testing.T) {
	args := gitleaksArgs("/scan/workspace", Config{GitleaksConfig: "does/not/exist.toml"}, "/tmp/report.json")

	if indexOfArg(args, "--config") >= 0 {
		t.Fatal("a missing config must be dropped so the scan falls back to default rules")
	}
}

func indexOfArg(args []string, want string) int {
	for i, a := range args {
		if a == want {
			return i
		}
	}
	return -1
}

// gitleaks writes no report and logs only to stderr when a scan is clean. That used
// to surface as parse_failed and flooded the learning pipeline with scanner_failed
// events for scans that actually succeeded.
func TestParseGitleaksScanOutputTreatsLogOnlyOutputAsClean(t *testing.T) {
	logOnly := []byte("2:44AM INF scan completed in 6.6ms\n2:44AM INF no leaks found\n")

	findings, err := parseGitleaksScanOutput(nil, logOnly, "/tmp/workspace")
	if err != nil {
		t.Fatalf("clean scan should not be a parse failure, got: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected zero findings, got %d", len(findings))
	}
}

func TestParseGitleaksScanOutputStillReportsMalformedJSON(t *testing.T) {
	malformed := []byte(`[{"RuleID": "generic-api-key"`)

	if _, err := parseGitleaksScanOutput(malformed, nil, "/tmp/workspace"); err == nil {
		t.Fatal("expected a parse error for truncated JSON report")
	}
}

func TestParseGitleaksScanOutputPrefersReportFile(t *testing.T) {
	report := []byte(`[{"RuleID":"generic-api-key","File":"/tmp/workspace/app.py","StartLine":7}]`)
	logOnly := []byte("2:44AM INF scan completed\n")

	findings, err := parseGitleaksScanOutput(report, logOnly, "/tmp/workspace")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].File != "app.py" {
		t.Fatalf("expected workspace-relative path, got %q", findings[0].File)
	}
}

func TestSanitizeHistoryGitErrorKeepsDiagnosisAndRedactsSecrets(t *testing.T) {
	raw := []byte("Cloning into '/tmp/rd-gitleaks-history-93417/repo'...\n" +
		"fatal: could not read Username for 'https://tokenvalue@git.example.org': No such device or address")

	msg := sanitizeHistoryGitError(raw).Error()

	if !strings.Contains(msg, "could not read Username") {
		t.Fatalf("expected git's own diagnosis to survive, got %q", msg)
	}
	if strings.Contains(msg, "tokenvalue") {
		t.Fatalf("credentials leaked into error: %q", msg)
	}
	if strings.Contains(msg, "rd-gitleaks-history-93417") {
		t.Fatalf("scratch workspace path leaked into error: %q", msg)
	}
}

func TestAuthenticatedCloneURLEmbedsToken(t *testing.T) {
	got, err := authenticatedCloneURL("https://git.example.org/owner/repo.git", "s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "oauth2:s3cr3t@git.example.org") {
		t.Fatalf("token not embedded: %q", got)
	}
}

func TestAuthenticatedCloneURLLeavesURLAloneWithoutToken(t *testing.T) {
	const raw = "https://git.example.org/owner/repo.git"
	got, err := authenticatedCloneURL(raw, "  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Fatalf("expected URL unchanged, got %q", got)
	}
}

func TestAuthenticatedCloneURLSkipsNonHTTPSchemes(t *testing.T) {
	const raw = "git@git.example.org:owner/repo.git"
	got, err := authenticatedCloneURL(raw, "s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Fatalf("ssh URL must not be rewritten, got %q", got)
	}
}

// A tokenized URL must never reach logs or findings if the clone fails.
func TestSanitizeHistoryGitErrorRedactsTokenizedCloneURL(t *testing.T) {
	authURL, err := authenticatedCloneURL("https://git.example.org/owner/repo.git", "s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	msg := sanitizeHistoryGitError([]byte("fatal: could not read from " + authURL)).Error()
	if strings.Contains(msg, "s3cr3t") {
		t.Fatalf("token leaked through sanitizer: %q", msg)
	}
}

func TestSanitizeHistoryGitErrorFallsBackWhenOutputEmpty(t *testing.T) {
	if got := sanitizeHistoryGitError(nil).Error(); got != "git operation failed" {
		t.Fatalf("unexpected fallback message: %q", got)
	}
}
