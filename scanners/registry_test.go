package scanners_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

func TestDefaultRegistryOrdering(t *testing.T) {
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

func TestRegistryRunAllDisabledStatuses(t *testing.T) {
	logger := logrus.New()
	reg := scanners.DefaultScannerRegistry()

	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:         logger,
		Workspace:      t.TempDir(),
		Config:         scanners.Config{EnableTrivy: false, EnableGrype: false, EnableGitleaks: false, EnableSemgrep: false, EnableGovulncheck: false, EnableGosec: false, EnableStaticcheck: false, EnableHadolint: false, EnableCheckov: false, EnableLinters: false},
		EnableSecurity: true,
		EnableQuality:  true,
	})

	if len(summary.Results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(summary.Results))
	}
	for _, result := range summary.Results {
		if result.Status != scanners.StatusDisabled {
			t.Fatalf("scanner %s: expected disabled, got %s", result.Scanner, result.Status)
		}
	}
}

func TestRegistryRunAllSecurityDisabledDetail(t *testing.T) {
	logger := logrus.New()
	reg := scanners.DefaultScannerRegistry()

	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:    logger,
		Workspace: t.TempDir(),
		Config: scanners.Config{
			EnableTrivy:       true,
			EnableGrype:       true,
			EnableGitleaks:    true,
			EnableSemgrep:     true,
			EnableGovulncheck: true,
			EnableGosec:       true,
			EnableHadolint:    true,
			EnableCheckov:     true,
			EnableLinters:     true,
		},
		EnableSecurity: false,
		EnableQuality:  true,
	})

	for _, result := range summary.Results {
		if result.Scanner != "trivy" && result.Scanner != "grype" && result.Scanner != "gitleaks" && result.Scanner != "semgrep" && result.Scanner != "govulncheck" && result.Scanner != "gosec" && result.Scanner != "hadolint" && result.Scanner != "checkov" {
			continue
		}
		if result.Status != scanners.StatusDisabled {
			t.Fatalf("%s: expected disabled, got %s", result.Scanner, result.Status)
		}
		if result.Detail != "security analysis disabled" {
			t.Fatalf("%s: unexpected detail %q", result.Scanner, result.Detail)
		}
	}
}

func TestRegistryRunAllPreservesAllResults(t *testing.T) {
	logger := logrus.New()
	reg := scanners.DefaultScannerRegistry()

	summary := reg.RunAll(context.Background(), scanners.RunRequest{
		Logger:    logger,
		Workspace: t.TempDir(),
		Entries:   []scanners.FileEntry{{Path: "main.go", Content: "package main\n"}},
		Config: scanners.Config{
			EnableTrivy:       false,
			EnableGrype:       false,
			EnableGitleaks:    false,
			EnableSemgrep:     false,
			EnableLinters:     true,
			LinterMinSeverity: "warning",
			TimeoutSeconds:    120,
		},
		EnableSecurity: true,
		EnableQuality:  true,
	})

	if len(summary.Results) < 2 {
		t.Fatalf("expected disabled trivy/grype plus linter results, got %d", len(summary.Results))
	}
	if summary.Results[0].Scanner != "trivy" || summary.Results[0].Status != scanners.StatusDisabled {
		t.Fatalf("unexpected first result: %+v", summary.Results[0])
	}
	if summary.Results[1].Scanner != "grype" || summary.Results[1].Status != scanners.StatusDisabled {
		t.Fatalf("unexpected second result: %+v", summary.Results[1])
	}
	if summary.Results[2].Scanner != "gitleaks" || summary.Results[2].Status != scanners.StatusDisabled {
		t.Fatalf("unexpected third result: %+v", summary.Results[2])
	}
	if summary.Results[3].Scanner != "semgrep" || summary.Results[3].Status != scanners.StatusDisabled {
		t.Fatalf("unexpected fourth result: %+v", summary.Results[3])
	}
	if summary.Results[4].Scanner != "govulncheck" || summary.Results[4].Status != scanners.StatusDisabled {
		t.Fatalf("unexpected fifth result: %+v", summary.Results[4])
	}
}

func TestDeterministicSourcesRecognized(t *testing.T) {
	for _, name := range []string{"trivy", "grype", "gitleaks", "semgrep", "govulncheck", "gosec", "staticcheck", "hadolint", "checkov", "golangci-lint", "ruff", "shellcheck"} {
		if !scanners.IsDeterministicSource(name) {
			t.Fatalf("expected %q to be deterministic", name)
		}
	}
}

func TestParseFailureStillReportedViaRegistry(t *testing.T) {
	_, err := scanners.ParseTrivyOutputForTest([]byte(`not-json`), t.TempDir())
	if err == nil {
		t.Fatal("expected parse failure from trivy parser")
	}
}
