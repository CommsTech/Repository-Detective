package scanners_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

const hadolintFoundJSON = `[{"code":"DL3008","message":"Pin versions in apt get install","line":3,"column":1,"file":"Dockerfile","level":"error"}]`

const checkovFoundJSON = `[{"check_type":"terraform","results":{"failed_checks":[{"check_id":"CKV_AWS_20","check_name":"S3 Bucket has public access block","file_path":"main.tf","file_line_range":[10,15],"resource":"aws_s3_bucket.example","guideline":"https://example.com","severity":"HIGH"}]}}]`

func TestDefaultRegistryIncludesIACScanners(t *testing.T) {
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

func TestIACScannersDeterministicSources(t *testing.T) {
	for _, name := range []string{"hadolint", "checkov"} {
		if !scanners.IsDeterministicSource(name) {
			t.Fatalf("expected %q to be deterministic", name)
		}
	}
}

func TestHadolintDisabled(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "Dockerfile", Content: "FROM alpine\n"}},
		Config:         scanners.Config{EnableHadolint: false},
		EnableSecurity: true,
	})
	assertScannerStatusIAC(t, summary, "hadolint", scanners.StatusDisabled)
}

func TestHadolintNoDockerfilesClean(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config:         scanners.Config{EnableHadolint: true},
		EnableSecurity: true,
	})
	assertScannerStatusIAC(t, summary, "hadolint", scanners.StatusClean)
}

func TestParseHadolintFound(t *testing.T) {
	dir := t.TempDir()
	parsed, err := scanners.ParseHadolintOutputForTest([]byte(hadolintFoundJSON), dir, scanners.Config{IACScannerMaxFindings: 100})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
	f := parsed.Findings[0]
	if f.Source != "hadolint" || f.Reference != "DL3008" || f.Severity != "high" {
		t.Fatalf("unexpected finding: %+v", f)
	}
}

func TestHadolintSeverityMapping(t *testing.T) {
	if scanners.MapHadolintSeverityForTest("error") != "high" {
		t.Fatal("error should map to high")
	}
	if scanners.MapHadolintSeverityForTest("style") != "low" {
		t.Fatal("style should map to low")
	}
}

func TestCheckovDisabled(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "main.tf", Content: "resource \"aws_s3_bucket\" \"x\" {}\n"}},
		Config:         scanners.Config{EnableCheckov: false},
		EnableSecurity: true,
	})
	assertScannerStatusIAC(t, summary, "checkov", scanners.StatusDisabled)
}

func TestCheckovNoIaCClean(t *testing.T) {
	reg := scanners.DefaultScannerRegistry()
	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logrus.New(),
		Workspace:      t.TempDir(),
		Entries:        []scanners.FileEntry{{Path: "README.md", Content: "# hi\n"}},
		Config:         scanners.Config{EnableCheckov: true},
		EnableSecurity: true,
	})
	assertScannerStatusIAC(t, summary, "checkov", scanners.StatusClean)
}

func TestParseCheckovFound(t *testing.T) {
	dir := t.TempDir()
	parsed, err := scanners.ParseCheckovOutputForTest([]byte(checkovFoundJSON), dir, scanners.Config{IACScannerMaxFindings: 100})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(parsed.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(parsed.Findings))
	}
	f := parsed.Findings[0]
	if f.Source != "checkov" || f.Reference != "CKV_AWS_20" || f.Severity != "high" {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if f.Category != "iac" {
		t.Fatalf("expected iac category, got %s", f.Category)
	}
}

func TestCheckovSeverityCategoryMapping(t *testing.T) {
	if scanners.MapCheckovSeverityForTest("HIGH") != "high" {
		t.Fatal("HIGH severity mapping failed")
	}
	if scanners.MapCheckovCategoryForTest("CKV_DOCKER_2", "", "") != "container" {
		t.Fatal("docker category mapping failed")
	}
}

func TestIACScannerTruncation(t *testing.T) {
	findings := make([]scanners.Finding, 0, 5)
	for i := 0; i < 5; i++ {
		findings = append(findings, scanners.Finding{ID: "DL3008"})
	}
	capped := scanners.CapFindingsForTest(findings, 2)
	if !capped.Truncated || len(capped.Findings) != 2 {
		t.Fatalf("unexpected cap: %+v", capped)
	}
}

func TestIsDockerfilePath(t *testing.T) {
	cases := map[string]bool{
		"Dockerfile":           true,
		"docker/Dockerfile":    true,
		"svc/app.Dockerfile":   true,
		"main.go":              false,
	}
	for path, want := range cases {
		if scanners.IsDockerfilePath(path) != want {
			t.Fatalf("%s: expected %v", path, want)
		}
	}
}

func TestWorkspaceHasIaC(t *testing.T) {
	entries := []scanners.FileEntry{{Path: "infra/main.tf", Content: `resource "aws_s3_bucket" "x" {}`}}
	if !scanners.WorkspaceHasIaC(entries) {
		t.Fatal("expected terraform file to match")
	}
}

func TestHadolintBinaryMissing(t *testing.T) {
	dir := writeDockerWorkspace(t)
	result := scanners.RunHadolintWithCommandForTest(context.Background(), logrus.New(), dir, []scanners.FileEntry{{Path: "Dockerfile"}}, scanners.Config{}, "hadolint-nonexistent")
	if result.Status != scanners.StatusBinaryMissing {
		t.Fatalf("expected binary_missing, got %s", result.Status)
	}
}

func assertScannerStatusIAC(t *testing.T, summary scanners.RunSummary, scanner string, want scanners.Status) {
	t.Helper()
	for _, res := range summary.Results {
		if res.Scanner == scanner {
			if res.Status != want {
				t.Fatalf("%s: expected %s, got %s", scanner, want, res.Status)
			}
			return
		}
	}
	t.Fatalf("scanner %q not found", scanner)
}

func writeDockerWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\nRUN apk add curl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
