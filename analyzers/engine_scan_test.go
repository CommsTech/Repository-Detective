package analyzers

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/scanners"
)

func TestIsDeterministicAuditor(t *testing.T) {
	cases := map[string]bool{
		"static":        true,
		"trivy":         true,
		"grype":         true,
		"gitleaks":      true,
		"semgrep":       true,
		"golangci-lint": true,
		"ruff":          true,
		"shellcheck":    true,
		"sql":           false,
		"xss":           false,
		"":              false,
	}
	for auditor, want := range cases {
		if got := isDeterministicAuditor(auditor); got != want {
			t.Fatalf("auditor %q: got %v want %v", auditor, got, want)
		}
	}
}

func TestIsDeterministicAuditorUsesScannerRegistry(t *testing.T) {
	scanners.RegisterDeterministicSource("future-scanner")
	if !isDeterministicAuditor("future-scanner") {
		t.Fatal("expected registered scanner source to be deterministic")
	}
	if isDeterministicAuditor("not-a-scanner") {
		t.Fatal("unexpected deterministic classification")
	}
}

func TestSelectLLMTargetFilesUsesDeterministicFlags(t *testing.T) {
	engine := &Engine{logger: nil, config: &Config{}}
	allFiles := []FileContent{
		{Path: "main.go", Content: "package main"},
		{Path: "readme.md", Content: "# hi"},
	}
	candidates := []CandidateFinding{
		{File: "main.go", AuditorType: "trivy"},
	}

	targets := engine.selectLLMTargetFiles(allFiles, candidates)
	if len(targets) != 1 || targets[0].Path != "main.go" {
		t.Fatalf("expected only main.go targeted, got %#v", targets)
	}
}

func TestSelectLLMTargetFilesAllWhenNoCandidates(t *testing.T) {
	engine := &Engine{config: &Config{}}
	allFiles := []FileContent{
		{Path: "a.go"},
		{Path: "b.go"},
	}
	targets := engine.selectLLMTargetFiles(allFiles, nil)
	if len(targets) != 2 {
		t.Fatalf("expected all files when no candidates, got %d", len(targets))
	}
}

func TestMergeFileContentsDedupesPaths(t *testing.T) {
	primary := []FileContent{{Path: "main.go", Content: "a"}}
	extra := []FileContent{
		{Path: "main.go", Content: "duplicate"},
		{Path: "go.mod", Content: "module x"},
	}
	merged := mergeFileContents(primary, extra)
	if len(merged) != 2 {
		t.Fatalf("expected 2 files, got %d", len(merged))
	}
}

func TestFirstNonEmptyCategory(t *testing.T) {
	if firstNonEmptyCategory("", "lint") != "lint" {
		t.Fatal("expected lint")
	}
	if firstNonEmptyCategory("", "") != "security" {
		t.Fatal("expected default security")
	}
}
