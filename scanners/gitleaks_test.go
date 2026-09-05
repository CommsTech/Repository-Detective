package scanners_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

const gitleaksCleanJSON = `[]`

const gitleaksFoundJSON = `[
  {
    "RuleID": "generic-api-key",
    "Description": "Generic API Key",
    "StartLine": 10,
    "EndLine": 10,
    "Match": "api_key=REDACTED_FOR_TEST_ONLY",
    "Secret": "REDACTED_FOR_TEST_ONLY",
    "File": "config/env.py",
    "Commit": "0000000000000000",
    "Entropy": 4.5,
    "Fingerprint": "test-fingerprint-001"
  }
]`

const gitleaksRawSecretNeverExpected = "AKI" + "AIOSFODNN7EXAMPLE"

func TestGitleaksDisabledStatus(t *testing.T) {
	logger := logrus.New()
	reg := scanners.DefaultScannerRegistry()

	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logger,
		Workspace:      t.TempDir(),
		Config:         scanners.Config{EnableGitleaks: false},
		EnableSecurity: true,
		EnableQuality:  true,
	})

	var gitleaksResult *scanners.RunResult
	for i := range summary.Results {
		if summary.Results[i].Scanner == "gitleaks" {
			gitleaksResult = &summary.Results[i]
			break
		}
	}
	if gitleaksResult == nil {
		t.Fatal("expected gitleaks result in summary")
	}
	if gitleaksResult.Status != scanners.StatusDisabled {
		t.Fatalf("expected disabled, got %s", gitleaksResult.Status)
	}
}

func TestGitleaksBinaryMissing(t *testing.T) {
	if scanners.CommandAvailableForTest("gitleaks") {
		t.Skip("gitleaks is installed — skipping binary-missing test")
	}

	logger := logrus.New()
	result := scanners.RunGitleaks(context.Background(), logger, t.TempDir(), scanners.DefaultConfig())
	if result.Status != scanners.StatusBinaryMissing {
		t.Fatalf("expected binary_missing, got %s", result.Status)
	}
}

func TestParseGitleaksOutputClean(t *testing.T) {
	findings, err := scanners.ParseGitleaksOutputForTest([]byte(gitleaksCleanJSON), t.TempDir())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected clean scan, got %d findings", len(findings))
	}
}

func TestParseGitleaksOutputFound(t *testing.T) {
	dir := t.TempDir()
	findings, err := scanners.ParseGitleaksOutputForTest([]byte(gitleaksFoundJSON), dir)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]
	if finding.Source != "gitleaks" {
		t.Fatalf("expected source gitleaks, got %q", finding.Source)
	}
	if finding.Category != "secret" {
		t.Fatalf("expected category secret, got %q", finding.Category)
	}
	if finding.Severity != "high" {
		t.Fatalf("expected severity high, got %q", finding.Severity)
	}
	if finding.Confidence != 0.95 {
		t.Fatalf("expected confidence 0.95, got %v", finding.Confidence)
	}
	if finding.File != "config/env.py" {
		t.Fatalf("unexpected file %q", finding.File)
	}
	if finding.Line != 10 {
		t.Fatalf("expected line 10, got %d", finding.Line)
	}
	wantID := "GITLEAKS-generic-api-key:config/env.py:10"
	if finding.ID != wantID {
		t.Fatalf("expected stable id %q, got %q", wantID, finding.ID)
	}
	if strings.Contains(finding.ID, "/tmp/") {
		t.Fatalf("id must not embed temp paths: %q", finding.ID)
	}
	if !strings.Contains(finding.Title, "generic-api-key") {
		t.Fatalf("unexpected title %q", finding.Title)
	}
	assertNoRawSecret(t, finding.Code)
	assertNoRawSecret(t, finding.Description)

	candidate := finding.ToCandidateFinding()
	if candidate.AuditorType != "gitleaks" {
		t.Fatalf("expected auditor type gitleaks, got %q", candidate.AuditorType)
	}
	assertNoRawSecret(t, candidate.Evidence.Code)
}

func TestGitleaksIDStableAcrossTempWorkspaces(t *testing.T) {
	payload := `[{
    "RuleID": "aws-access-token",
    "Description": "AWS",
    "StartLine": 13,
    "EndLine": 13,
    "Match": "REDACTED",
    "Secret": "REDACTED",
    "File": "redact/secrets_test.go",
    "Fingerprint": "/tmp/rd-archive-12345/redact/secrets_test.go:aws-access-token:13"
  }]`
	a, err := scanners.ParseGitleaksOutputForTest([]byte(payload), "/tmp/rd-archive-111")
	if err != nil {
		t.Fatal(err)
	}
	b, err := scanners.ParseGitleaksOutputForTest([]byte(payload), "/tmp/rd-archive-222")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("expected 1 finding each")
	}
	if a[0].ID != b[0].ID {
		t.Fatalf("unstable ids across workspaces: %q vs %q", a[0].ID, b[0].ID)
	}
	if strings.Contains(a[0].ID, "rd-archive") {
		t.Fatalf("id still has archive path: %q", a[0].ID)
	}
}

const gitleaksFoundOneLine = `[{"RuleID":"generic-api-key","Description":"Generic API Key","StartLine":10,"EndLine":10,"Match":"api_key=REDACTED_FOR_TEST_ONLY","Secret":"REDACTED_FOR_TEST_ONLY","File":"config/env.py","Commit":"0000000000000000","Entropy":4.5,"Fingerprint":"test-fingerprint-001"}]`

func TestGitleaksNonzeroExitWithFindingsIsFound(t *testing.T) {
	logger := logrus.New()
	dir := t.TempDir()
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true
	cmd, args := testWriteScript(gitleaksFoundOneLine, 1)

	result := scanners.RunGitleaksWithCommandForTest(
		context.Background(),
		logger,
		dir,
		cfg,
		cmd,
		args...,
	)
	if result.Status != scanners.StatusFound {
		t.Fatalf("expected found, got %s detail=%q", result.Status, result.Detail)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestGitleaksParseFailure(t *testing.T) {
	logger := logrus.New()
	cmd, args := testWriteScript("not-json", 0)
	result := scanners.RunGitleaksWithCommandForTest(
		context.Background(),
		logger,
		t.TempDir(),
		scanners.DefaultConfig(),
		cmd,
		args...,
	)
	if result.Status != scanners.StatusParseFailed {
		t.Fatalf("expected parse_failed, got %s", result.Status)
	}
}

func TestGitleaksTimeout(t *testing.T) {
	logger := logrus.New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := scanners.DefaultConfig()
	cfg.GitleaksTimeoutSeconds = 1
	cmd, args := testSleepScript(5)

	result := scanners.RunGitleaksWithCommandForTest(
		ctx,
		logger,
		t.TempDir(),
		cfg,
		cmd,
		args...,
	)
	if result.Status != scanners.StatusTimedOut {
		t.Fatalf("expected timed_out, got %s detail=%q", result.Status, result.Detail)
	}
}

func TestGitleaksIsDeterministicSource(t *testing.T) {
	if !scanners.IsDeterministicSource("gitleaks") {
		t.Fatal("expected gitleaks to be registered as deterministic")
	}
}

func TestParseGitleaksScanOutputPrefersReportFile(t *testing.T) {
	dir := t.TempDir()
	stderrLogs := []byte("\x1b[90m1:37PM\x1b[0m \x1b[32mINF\x1b[0m scan completed\n\x1b[31mWRN\x1b[0m leaks found: 10\n")
	findings, err := scanners.ParseGitleaksScanOutputForTest([]byte(gitleaksFoundJSON), stderrLogs, dir)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding from report file, got %d", len(findings))
	}
}

func TestParseGitleaksStderrANSIIgnoredWhenReportFileValid(t *testing.T) {
	dir := t.TempDir()
	// stderr-only ANSI brackets must not be parsed when report file is present.
	findings, err := scanners.ParseGitleaksScanOutputForTest([]byte(gitleaksCleanJSON), []byte("\x1b[90mINF\x1b[0m no leaks\n"), dir)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected clean report file, got %d", len(findings))
	}
}

func TestParseGitleaksOutputInvalidJSON(t *testing.T) {
	_, err := scanners.ParseGitleaksOutputForTest([]byte("not-json"), t.TempDir())
	if err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func assertNoRawSecret(t *testing.T, value string) {
	t.Helper()
	if strings.Contains(value, gitleaksRawSecretNeverExpected) {
		t.Fatalf("normalized output must not contain raw secret, got %q", value)
	}
}
