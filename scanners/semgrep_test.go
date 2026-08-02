package scanners_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

const semgrepCleanJSON = `{"version":"1.0.0","results":[]}`

const semgrepFoundJSON = `{
  "version": "1.0.0",
  "results": [
    {
      "check_id": "python.lang.security.audit.eval-detected.eval-detected",
      "path": "src/app.py",
      "start": {"line": 42, "col": 5, "offset": 100},
      "end": {"line": 42, "col": 20, "offset": 115},
      "extra": {
        "message": "Detected use of eval",
        "metadata": {
          "category": "security",
          "cwe": ["CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code"],
          "owasp": ["A03:2021 - Injection"]
        },
        "severity": "ERROR",
        "fingerprint": "fp-semgrep-001",
        "lines": "result = eval(user_input)"
      }
    }
  ]
}`

const semgrepFoundOneLine = `{"version":"1.0.0","results":[{"check_id":"python.lang.security.audit.eval-detected.eval-detected","path":"src/app.py","start":{"line":42,"col":5,"offset":100},"end":{"line":42,"col":20,"offset":115},"extra":{"message":"Detected use of eval","metadata":{"category":"security","cwe":["CWE-95"],"owasp":["A03:2021 - Injection"]},"severity":"ERROR","fingerprint":"fp-semgrep-001","lines":"result = eval(user_input)"}}]}`

func TestSemgrepDisabledStatus(t *testing.T) {
	logger := logrus.New()
	reg := scanners.DefaultScannerRegistry()

	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logger,
		Workspace:      t.TempDir(),
		Config:         scanners.Config{EnableSemgrep: false},
		EnableSecurity: true,
		EnableQuality:  true,
	})

	var semgrepResult *scanners.RunResult
	for i := range summary.Results {
		if summary.Results[i].Scanner == "semgrep" {
			semgrepResult = &summary.Results[i]
			break
		}
	}
	if semgrepResult == nil {
		t.Fatal("expected semgrep result in summary")
	}
	if semgrepResult.Status != scanners.StatusDisabled {
		t.Fatalf("expected disabled, got %s", semgrepResult.Status)
	}
}

func TestSemgrepBinaryMissing(t *testing.T) {
	if scanners.CommandAvailableForTest("semgrep") {
		t.Skip("semgrep is installed — skipping binary-missing test")
	}

	logger := logrus.New()
	result := scanners.RunSemgrep(context.Background(), logger, t.TempDir(), scanners.DefaultConfig())
	if result.Status != scanners.StatusBinaryMissing {
		t.Fatalf("expected binary_missing, got %s", result.Status)
	}
}

func TestParseSemgrepOutputClean(t *testing.T) {
	parsed, err := scanners.ParseSemgrepOutputForTest([]byte(semgrepCleanJSON), t.TempDir(), scanners.DefaultConfig())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 0 {
		t.Fatalf("expected clean scan, got %d findings", len(parsed.Findings))
	}
}

func TestParseSemgrepOutputFound(t *testing.T) {
	dir := t.TempDir()
	cfg := scanners.DefaultConfig()
	parsed, err := scanners.ParseSemgrepOutputForTest([]byte(semgrepFoundJSON), dir, cfg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}

	finding := parsed.Findings[0]
	if finding.Source != "semgrep" {
		t.Fatalf("expected source semgrep, got %q", finding.Source)
	}
	if finding.Category != "security" {
		t.Fatalf("expected category security, got %q", finding.Category)
	}
	if finding.Severity != "high" {
		t.Fatalf("expected severity high, got %q", finding.Severity)
	}
	if finding.Confidence != 0.90 {
		t.Fatalf("expected confidence 0.90, got %v", finding.Confidence)
	}
	if finding.File != "src/app.py" {
		t.Fatalf("unexpected file %q", finding.File)
	}
	if finding.Line != 42 {
		t.Fatalf("expected line 42, got %d", finding.Line)
	}
	if !strings.Contains(finding.Title, "eval-detected") {
		t.Fatalf("unexpected title %q", finding.Title)
	}
	if !strings.Contains(finding.Description, "CWE-95") {
		t.Fatalf("expected CWE in description, got %q", finding.Description)
	}
	if !strings.Contains(finding.Description, "A03:2021") {
		t.Fatalf("expected OWASP in description, got %q", finding.Description)
	}
	if finding.Code != "result = eval(user_input)" {
		t.Fatalf("unexpected code snippet %q", finding.Code)
	}

	candidate := finding.ToCandidateFinding()
	if candidate.AuditorType != "semgrep" {
		t.Fatalf("expected auditor type semgrep, got %q", candidate.AuditorType)
	}
}

func TestSemgrepNonzeroExitWithFindingsIsFound(t *testing.T) {
	logger := logrus.New()
	dir := t.TempDir()
	cfg := scanners.DefaultConfig()
	cfg.EnableSemgrep = true
	cmd, args := testWriteScript(semgrepFoundOneLine, 1)

	result := scanners.RunSemgrepWithCommandForTest(
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

func TestSemgrepParseFailure(t *testing.T) {
	logger := logrus.New()
	cmd, args := testWriteScript("not-json", 0)
	result := scanners.RunSemgrepWithCommandForTest(
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

func TestSemgrepTimeout(t *testing.T) {
	logger := logrus.New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := scanners.DefaultConfig()
	cfg.SemgrepTimeoutSeconds = 1
	cmd, args := testSleepScript(5)

	result := scanners.RunSemgrepWithCommandForTest(
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

func TestSemgrepMaxFindingsTruncation(t *testing.T) {
	var results strings.Builder
	results.WriteString(`{"version":"1.0.0","results":[`)
	for i := 0; i < 5; i++ {
		if i > 0 {
			results.WriteString(",")
		}
		fmt.Fprintf(&results, `{"check_id":"rule-%d","path":"file-%d.py","start":{"line":%d,"col":1,"offset":0},"end":{"line":%d,"col":1,"offset":0},"extra":{"message":"finding %d","metadata":{},"severity":"ERROR","fingerprint":"fp-%d","lines":"x = %d"}}`, i, i, i+1, i+1, i, i, i)
	}
	results.WriteString("]}")

	cfg := scanners.Config{SemgrepMaxFindings: 2, SemgrepSeverityThreshold: "INFO"}
	parsed, err := scanners.ParseSemgrepOutputForTest([]byte(results.String()), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 2 {
		t.Fatalf("expected 2 findings after cap, got %d", len(parsed.Findings))
	}
	if !parsed.Truncated {
		t.Fatal("expected truncated flag")
	}
	if parsed.Total != 5 {
		t.Fatalf("expected total 5, got %d", parsed.Total)
	}

	logger := logrus.New()
	cmd, args := testWriteScript(results.String(), 0)
	result := scanners.RunSemgrepWithCommandForTest(
		context.Background(),
		logger,
		t.TempDir(),
		cfg,
		cmd,
		args...,
	)
	if result.Status != scanners.StatusFound {
		t.Fatalf("expected found, got %s", result.Status)
	}
	if !strings.Contains(result.Detail, "truncated to 2 findings (5 total)") {
		t.Fatalf("unexpected detail %q", result.Detail)
	}
}

func TestSemgrepSeverityMapping(t *testing.T) {
	cases := map[string]string{
		"ERROR":   "high",
		"WARNING": "medium",
		"INFO":    "low",
		"":        "medium",
		"weird":   "medium",
	}
	for input, want := range cases {
		if got := scanners.MapSemgrepSeverityForTest(input); got != want {
			t.Fatalf("severity %q: got %q want %q", input, got, want)
		}
	}
}

func TestSemgrepSeverityThresholdFiltersInfo(t *testing.T) {
	payload := `{"version":"1.0.0","results":[
		{"check_id":"info-rule","path":"a.py","start":{"line":1,"col":1,"offset":0},"end":{"line":1,"col":1,"offset":0},"extra":{"message":"info","metadata":{},"severity":"INFO","fingerprint":"fp-info","lines":"x=1"}},
		{"check_id":"error-rule","path":"b.py","start":{"line":2,"col":1,"offset":0},"end":{"line":2,"col":1,"offset":0},"extra":{"message":"error","metadata":{},"severity":"ERROR","fingerprint":"fp-error","lines":"y=2"}}
	]}`

	cfg := scanners.Config{SemgrepSeverityThreshold: "WARNING"}
	parsed, err := scanners.ParseSemgrepOutputForTest([]byte(payload), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding after threshold filter, got %d", len(parsed.Findings))
	}
	if parsed.Findings[0].Reference != "error-rule" {
		t.Fatalf("expected error-rule, got %q", parsed.Findings[0].Reference)
	}
}

func TestSemgrepIsDeterministicSource(t *testing.T) {
	if !scanners.IsDeterministicSource("semgrep") {
		t.Fatal("expected semgrep to be registered as deterministic")
	}
}

func TestDefaultRegistryIncludesSemgrepOrdering(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	names := reg.Names()
	want := []string{"trivy", "grype", "gitleaks", "semgrep", "govulncheck", "gosec", "staticcheck", "hadolint", "checkov", "linters"}
	if len(names) != len(want) {
		t.Fatalf("expected %d registry entries, got %v", len(want), names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("index %d: expected %q, got %q", i, name, names[i])
		}
	}
}
