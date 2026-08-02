package scanners_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

func TestRunAllDisabledScanners(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	summary := scanners.RunAll(context.Background(), logger, t.TempDir(), nil, scanners.Config{
		EnableTrivy:         false,
		EnableGrype:         false,
		EnableGitleaks:      false,
		EnableSemgrep:       false,
		EnableGovulncheck:   false,
		EnableGosec:         false,
		EnableStaticcheck:   false,
		EnableHadolint:      false,
		EnableCheckov:       false,
		EnableLinters:       false,
	}, true, true)

	if len(summary.Results) != 10 {
		t.Fatalf("expected 10 scanner results, got %d", len(summary.Results))
	}
	for _, result := range summary.Results {
		if result.Status != scanners.StatusDisabled {
			t.Fatalf("scanner %s: expected disabled, got %s", result.Scanner, result.Status)
		}
	}
	if len(summary.Candidates()) != 0 {
		t.Fatal("expected no findings")
	}
}

func TestParseTrivyOutputCleanScan(t *testing.T) {
	findings, err := scanners.ParseTrivyOutputForTest([]byte(`{"Results":[]}`), "/tmp/workspace")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected clean scan, got %d findings", len(findings))
	}
}

func TestParseTrivyOutputParseFailure(t *testing.T) {
	_, err := scanners.ParseTrivyOutputForTest([]byte(`not-json`), "/tmp/workspace")
	if err == nil {
		t.Fatal("expected parse failure")
	}
}

func TestRunTrivyBinaryMissing(t *testing.T) {
	if scanners.CommandAvailableForTest("trivy") {
		t.Skip("trivy is installed — skipping binary-missing test")
	}

	logger := logrus.New()
	result := scanners.RunTrivy(context.Background(), logger, t.TempDir(), scanners.DefaultConfig())
	if result.Status != scanners.StatusBinaryMissing {
		t.Fatalf("expected binary_missing, got %s", result.Status)
	}
}

func TestRunTrivyTimeout(t *testing.T) {
	logger := logrus.New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := scanners.DefaultConfig()
	cfg.TimeoutSeconds = 1
	cmd, args := testSleepScript(5)

	// Use a built-in long-running command name substitute via test hook.
	result := scanners.RunTrivyWithCommandForTest(ctx, logger, t.TempDir(), cfg, cmd, args...)
	if result.Status != scanners.StatusTimedOut {
		t.Fatalf("expected timed_out, got %s detail=%q", result.Status, result.Detail)
	}
}

func TestRunTrivyFoundIssues(t *testing.T) {
	payload := `{
		"Results": [{
			"Target": "go.mod",
			"Vulnerabilities": [{
				"VulnerabilityID": "CVE-2024-1234",
				"PkgName": "example.com/lib",
				"InstalledVersion": "1.0.0",
				"FixedVersion": "1.0.1",
				"Severity": "HIGH",
				"Title": "Example CVE"
			}]
		}]
	}`

	findings, err := scanners.ParseTrivyOutputForTest([]byte(payload), "/tmp/workspace")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}
