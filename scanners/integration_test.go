package scanners_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

func TestRunTrivyOnSampleWorkspace(t *testing.T) {
	if _, err := exec.LookPath("trivy"); err != nil {
		t.Skip("trivy not installed")
	}

	dir, cleanup, err := scanners.CreateWorkspace([]scanners.FileEntry{
		{
			Path:    "Dockerfile",
			Content: "FROM alpine:3.10\nRUN apk add --no-cache curl\n",
		},
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result := scanners.RunTrivy(ctx, logger, dir, scanners.DefaultConfig())
	t.Logf("trivy status=%s findings=%d", result.Status, len(result.Findings))
}

func TestRunLintersOnGoFile(t *testing.T) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not installed")
	}

	dir, cleanup, err := scanners.CreateWorkspace([]scanners.FileEntry{
		{
			Path: "main.go",
			Content: `package main

import "fmt"

func main() {
	fmt.Println("unused")
	unused := 1
	_ = unused
}
`,
		},
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	findings := scanners.RunLinters(ctx, logger, dir, []scanners.FileEntry{
		{Path: "main.go"},
	}, true, true, scanners.DefaultConfig())
	var total int
	for _, result := range findings {
		total += len(result.Findings)
	}
	t.Logf("linter findings: %d", total)
}

func TestRunAllReturnsCandidates(t *testing.T) {
	dir, err := os.MkdirTemp("", "rd-all-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := scanners.Config{
		EnableTrivy:    false,
		EnableGrype:    false,
		EnableGitleaks: false,
		EnableSemgrep:  false,
		EnableLinters:  false,
	}
	entries := []scanners.FileEntry{{Path: "go.mod", Content: "module example.com/test\n\ngo 1.21\n"}}
	summary := scanners.RunAll(context.Background(), logger, dir, entries, cfg, true, true)
	if len(summary.Candidates()) != 0 {
		t.Fatalf("expected no candidates with scanners disabled, got %d", len(summary.Candidates()))
	}
	for _, result := range summary.Results {
		if result.Status != scanners.StatusDisabled {
			t.Fatalf("expected disabled status for %s, got %s", result.Scanner, result.Status)
		}
	}
}
