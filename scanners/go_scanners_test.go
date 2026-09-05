package scanners_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

const govulncheckFindingJSON = `{"finding":{"osv":"GO-2024-0001","module":"example.com/mod","symbol":"Vulnerable","trace":[{"function":"main.main","position":{"filename":"main.go","line":10}}]}}`

const gosecFoundJSON = `{"Issues":[{"severity":"HIGH","confidence":"HIGH","rule_id":"G204","details":"Subprocess launched with variable","file":"main.go","line":"12","code":"exec.Command(userInput)"}]}`

const staticcheckFoundJSON = `{"code":"SA1019","message":"deprecated usage","location":{"file":"main.go","line":5}}`

func TestDefaultRegistryIncludesGoScanners(t *testing.T) {
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

func TestGoScannersDeterministicSources(t *testing.T) {
	for _, name := range []string{"govulncheck", "gosec", "staticcheck"} {
		if !scanners.IsDeterministicSource(name) {
			t.Fatalf("expected %q to be deterministic", name)
		}
	}
}

func TestGovulncheckDisabled(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config:         scanners.Config{EnableGovulncheck: false},
		EnableSecurity: true,
	})
	assertScannerStatus(t, summary, "govulncheck", scanners.StatusDisabled)
}

func TestGovulncheckBinaryMissing(t *testing.T) {
	if scanners.CommandAvailableForTest("govulncheck-nonexistent") {
		t.Skip("unexpected binary")
	}
	dir := writeGoWorkspace(t)
	result := scanners.RunGovulncheckWithCommandForTest(context.Background(), logrus.New(), dir, scanners.Config{EnableGovulncheck: true}, "govulncheck-nonexistent")
	if result.Status != scanners.StatusBinaryMissing {
		t.Fatalf("expected binary_missing, got %s", result.Status)
	}
}

func TestParseGovulncheckFinding(t *testing.T) {
	dir := t.TempDir()
	cfg := scanners.Config{GoScannerMaxFindings: 100}
	parsed, err := scanners.ParseGovulncheckOutputForTest([]byte(govulncheckFindingJSON), dir, cfg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
	f := parsed.Findings[0]
	if f.Source != "govulncheck" || f.Reference != "GO-2024-0001" {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if f.Severity != "high" || f.Confidence != 0.95 {
		t.Fatalf("unexpected severity/confidence: %s %v", f.Severity, f.Confidence)
	}
}

func TestParseGovulncheckParseFailure(t *testing.T) {
	_, err := scanners.ParseGovulncheckOutputForTest([]byte("not json at all"), t.TempDir(), scanners.Config{})
	if err == nil {
		t.Fatal("expected parse failure")
	}
}

func TestGovulncheckTruncation(t *testing.T) {
	findings := make([]scanners.Finding, 0, 5)
	for i := 0; i < 5; i++ {
		findings = append(findings, scanners.Finding{ID: fmt.Sprintf("GO-%d", i)})
	}
	capped := scanners.CapFindingsForTest(findings, 2)
	if !capped.Truncated || len(capped.Findings) != 2 || capped.Total != 5 {
		t.Fatalf("unexpected cap result: %+v", capped)
	}
}

func TestGosecDisabled(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config:         scanners.Config{EnableGosec: false},
		EnableSecurity: true,
	})
	assertScannerStatus(t, summary, "gosec", scanners.StatusDisabled)
}

func TestParseGosecFound(t *testing.T) {
	dir := t.TempDir()
	parsed, err := scanners.ParseGosecOutputForTest([]byte(gosecFoundJSON), dir, scanners.Config{GoScannerMaxFindings: 100})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
	f := parsed.Findings[0]
	if f.Source != "gosec" || f.Reference != "G204" {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if f.Severity != "high" {
		t.Fatalf("expected high severity, got %s", f.Severity)
	}
}

func TestGosecSeverityConfidenceMapping(t *testing.T) {
	if scanners.MapGosecSeverityForTest("HIGH") != "high" {
		t.Fatal("HIGH severity mapping failed")
	}
	if scanners.MapGosecSeverityForTest("LOW") != "low" {
		t.Fatal("LOW severity mapping failed")
	}
	if scanners.MapGosecConfidenceForTest("HIGH") != 0.90 {
		t.Fatal("HIGH confidence mapping failed")
	}
	if scanners.MapGosecConfidenceForTest("LOW") != 0.65 {
		t.Fatal("LOW confidence mapping failed")
	}
}

func TestParseGosecParseFailure(t *testing.T) {
	_, err := scanners.ParseGosecOutputForTest([]byte("{"), t.TempDir(), scanners.Config{})
	if err == nil {
		t.Fatal("expected parse failure")
	}
}

func TestStaticcheckDisabled(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:        logrus.New(),
		Workspace:     t.TempDir(),
		Entries:       []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config:        scanners.Config{EnableStaticcheck: false},
		EnableQuality: true,
	})
	assertScannerStatus(t, summary, "staticcheck", scanners.StatusDisabled)
}

func TestParseStaticcheckNDJSON(t *testing.T) {
	dir := t.TempDir()
	parsed, err := scanners.ParseStaticcheckOutputForTest([]byte(staticcheckFoundJSON), dir, scanners.Config{GoScannerMaxFindings: 100})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
	f := parsed.Findings[0]
	if f.Reference != "SA1019" || f.Source != "staticcheck" {
		t.Fatalf("unexpected finding: %+v", f)
	}
	category, severity, confidence := scanners.MapStaticcheckCodeForTest("SA1019")
	if category != "reliability" || severity != "medium" || confidence != 0.90 {
		t.Fatalf("unexpected SA mapping: %s %s %v", category, severity, confidence)
	}
}

func TestParseStaticcheckSkipsNonJSONLines(t *testing.T) {
	dir := t.TempDir()
	raw := "err: go command required\n" + staticcheckFoundJSON + "\nwarning: ignored\n"
	parsed, err := scanners.ParseStaticcheckOutputForTest([]byte(raw), dir, scanners.Config{GoScannerMaxFindings: 100})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
}

func TestStaticcheckCodeMappings(t *testing.T) {
	cases := map[string]string{
		"ST1000": "maintainability",
		"QF1001": "code_quality",
		"S1001":  "code_quality",
	}
	for code, wantCategory := range cases {
		category, _, _ := scanners.MapStaticcheckCodeForTest(code)
		if category != wantCategory {
			t.Fatalf("%s: expected category %s, got %s", code, wantCategory, category)
		}
	}
}

func TestRegistryRunAllDisabledGoScanners(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config:         scanners.Config{EnableTrivy: false, EnableGrype: false, EnableGitleaks: false, EnableSemgrep: false, EnableGovulncheck: false, EnableGosec: false, EnableStaticcheck: false, EnableHadolint: false, EnableCheckov: false, EnableLinters: false},
		EnableSecurity: true,
		EnableQuality:  true,
	})
	if len(summary.Results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(summary.Results))
	}
}

func assertScannerStatus(t *testing.T, summary scanners.RunSummary, scanner string, want scanners.Status) {
	t.Helper()
	for _, res := range summary.Results {
		if res.Scanner == scanner {
			if res.Status != want {
				t.Fatalf("%s: expected %s, got %s", scanner, want, res.Status)
			}
			return
		}
	}
	t.Fatalf("scanner %q not found in summary", scanner)
}

func writeGoWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
