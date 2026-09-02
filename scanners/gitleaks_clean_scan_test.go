package scanners

import (
	"strings"
	"testing"
)

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

func TestSanitizeHistoryGitErrorFallsBackWhenOutputEmpty(t *testing.T) {
	if got := sanitizeHistoryGitError(nil).Error(); got != "git operation failed" {
		t.Fatalf("unexpected fallback message: %q", got)
	}
}
