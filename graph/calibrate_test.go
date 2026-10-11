package graph_test

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/graph"
)

func TestMakefileReferencedScriptNotOrphan(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "Makefile", Content: "run:\n\t./scripts/deploy.sh\n", Language: "generic"},
			{Path: "scripts/deploy.sh", Content: "#!/bin/sh\necho hi\n", Language: "shell"},
			{Path: "lib/util.py", Content: "def helper(): pass\n", Language: "python"},
		},
		Repo: graph.RepoContext{FileCount: 3, HomelabInfra: true},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && strings.Contains(f.File, "deploy.sh") {
			t.Fatalf("Makefile-referenced script should not be orphan: %+v", f)
		}
	}
}

func TestDockerComposeReferencedFileNotOrphan(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "docker-compose.yml", Content: "services:\n  app:\n    command: ./entry.sh\n", Language: "yaml"},
			{Path: "entry.sh", Content: "#!/bin/sh\n", Language: "shell"},
		},
		Repo: graph.RepoContext{FileCount: 2, HomelabInfra: true},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && f.File == "entry.sh" {
			t.Fatalf("compose-referenced entry should not be orphan")
		}
	}
}

func TestDocsExampleLowerActionability(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "docs/example.go", Content: "package example\n", Language: "go"},
		},
		Repo: graph.RepoContext{FileCount: 2},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && strings.Contains(f.File, "docs/example") {
			t.Fatalf("docs/example should be excluded from orphan findings")
		}
	}
}

func TestSmallRepoGraphNoiseDowngraded(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "lonely.go", Content: "package lonely\n", Language: "go"},
		},
		Repo: graph.RepoContext{FileCount: 2, HomelabInfra: true},
	}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && f.File == "lonely.go" {
			found = true
			if f.Severity != "info" {
				t.Fatalf("expected info severity, got %s", f.Severity)
			}
			if f.Detail.CalibrationNote == "" {
				t.Fatal("expected calibration note")
			}
		}
	}
	if !found {
		t.Fatal("expected downgraded orphan finding for real orphan in small repo")
	}
}

func TestRealOrphanStillDetected(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "orphan.go", Content: "package orphan\nfunc unused() {}\n", Language: "go"},
		},
		Repo: graph.RepoContext{FileCount: 200},
	}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && f.File == "orphan.go" {
			found = true
			if f.Severity == "info" {
				t.Fatal("large repo orphan should remain low severity, not auto-downgraded to info")
			}
		}
	}
	if !found {
		t.Fatal("expected orphan file finding")
	}
}
