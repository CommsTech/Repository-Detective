package scanners_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/scanners"
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

const gitleaksRawSecretNeverExpected = "AKIAIOSFODNN7EXAMPLE"

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

const gitleaksFoundOneLine = `[{"RuleID":"generic-api-key","Description":"Generic API Key","StartLine":10,"EndLine":10,"Match":"api_key=REDACTED_FOR_TEST_ONLY","Secret":"REDACTED_FOR_TEST_ONLY","File":"config/env.py","Commit":"0000000000000000","Entropy":4.5,"Fingerprint":"test-fingerprint-001"}]`

func TestGitleaksNonzeroExitWithFindingsIsFound(t *testing.T) {
	logger := logrus.New()
	dir := t.TempDir()
	cfg := scanners.DefaultConfig()
	cfg.EnableGitleaks = true

	// PowerShell exits 1 after writing JSON to stdout — mirrors gitleaks leak exit code.
	result := scanners.RunGitleaksWithCommandForTest(
		context.Background(),
		logger,
		dir,
		cfg,
		"powershell",
		"-NoProfile",
		"-Command",
		"[Console]::Out.Write('"+gitleaksFoundOneLine+"'); exit 1",
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
	result := scanners.RunGitleaksWithCommandForTest(
		context.Background(),
		logger,
		t.TempDir(),
		scanners.DefaultConfig(),
		"powershell",
		"-NoProfile",
		"-Command",
		"[Console]::Out.Write('not-json')",
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

	result := scanners.RunGitleaksWithCommandForTest(
		ctx,
		logger,
		t.TempDir(),
		cfg,
		"powershell",
		"-NoProfile",
		"-Command",
		"Start-Sleep -Seconds 5",
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

func assertNoRawSecret(t *testing.T, value string) {
	t.Helper()
	if strings.Contains(value, gitleaksRawSecretNeverExpected) {
		t.Fatalf("normalized output must not contain raw secret, got %q", value)
	}
}
