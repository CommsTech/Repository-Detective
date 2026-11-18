package graph_test

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func testCfg() graph.Config {
	cfg := graph.DefaultConfig()
	cfg.MaxNodes = 500
	cfg.MaxEdges = 2000
	return cfg
}

func TestRepoTreeNodes(t *testing.T) {
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{Path: "pkg/foo.go", Content: "package pkg\n", Language: "go"}},
	}, testCfg(), nil)
	types := map[string]bool{}
	for _, n := range g.Nodes {
		types[n.Type] = true
	}
	if !types["repo"] || !types["directory"] || !types["file"] {
		t.Fatalf("expected repo/directory/file nodes, got %v", types)
	}
}

func TestGoImports(t *testing.T) {
	content := "package main\nimport \"fmt\"\nimport \"./local\"\nfunc main() {}\n"
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: content, Language: "go"},
			{Path: "local.go", Content: "package main\n", Language: "go"},
		},
	}, testCfg(), nil)
	hasImport := false
	for _, e := range g.Edges {
		if e.Type == "imports" || e.Type == "depends_on" {
			hasImport = true
		}
	}
	if !hasImport {
		t.Fatal("expected import edges")
	}
}

func TestOrphanFileDetected(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "orphan.go", Content: "package orphan\nfunc unused() {}\n", Language: "go"},
		},
	}, testCfg(), nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected orphan file finding")
	}
}

func TestTestFileNotOrphan(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "foo.go", Content: "package foo\n", Language: "go"},
			{Path: "foo_test.go", Content: "package foo\n", Language: "go"},
		},
	}, testCfg(), nil)
	for _, f := range findings {
		if strings.Contains(f.File, "_test.go") {
			t.Fatalf("test file should not be flagged orphan/disconnected: %s %s", f.RuleID, f.File)
		}
	}
}

func TestMainNotOrphan(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
		},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID == "GRAPH-ORPHAN-FILE" && f.File == "main.go" {
			t.Fatal("main should not be orphan")
		}
	}
}

func TestFindingOverlay(t *testing.T) {
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{Path: "a.go", Content: "package a\n", Language: "go"}},
		Findings: []graph.FindingOverlay{{
			ID: "f1", File: "a.go", Severity: "high", Category: "security", Title: "Issue",
		}},
	}, testCfg(), nil)
	found := false
	for _, n := range g.Nodes {
		if n.Type == "finding" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected finding node")
	}
}

func TestSkipVendor(t *testing.T) {
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{Path: "vendor/lib/x.go", Content: "package lib\n", Language: "go"}},
	}, testCfg(), nil)
	if len(g.Nodes) > 2 {
		t.Fatalf("vendor should be skipped, got %d nodes", len(g.Nodes))
	}
}

func TestGraphDisabled(t *testing.T) {
	cfg := testCfg()
	cfg.Enabled = false
	g, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{Path: "a.go", Content: "package a\n", Language: "go"}},
	}, cfg, nil)
	if len(g.Nodes) != 0 || len(findings) != 0 {
		t.Fatal("disabled graph should return empty")
	}
}

func TestDeterministicSourceRegistered(t *testing.T) {
	if !scanners.IsDeterministicSource("graph") {
		t.Fatal("graph should be deterministic source")
	}
}

func TestOrphanWording(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "lonely.go", Content: "package lonely\n", Language: "go"},
		},
	}, testCfg(), nil)
	for _, f := range findings {
		if strings.Contains(strings.ToLower(f.Title), "ai-written") {
			t.Fatal("must not claim AI authorship")
		}
		if f.RuleID == "GRAPH-ORPHAN-FILE" && !strings.Contains(strings.ToLower(f.Title), "possible") {
			t.Fatal("title should use cautious wording")
		}
	}
}

func TestJSImports(t *testing.T) {
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{
			Path: "index.js", Content: "import fs from 'fs'\nimport './util.js'\n", Language: "javascript",
		}},
	}, testCfg(), nil)
	if len(g.Edges) == 0 {
		t.Fatal("expected JS import edges")
	}
}

func TestPythonImports(t *testing.T) {
	g, _ := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{{
			Path: "app.py", Content: "import os\nfrom utils import helper\n", Language: "python",
		}},
	}, testCfg(), nil)
	hasExt := false
	for _, e := range g.Edges {
		if e.Type == "depends_on" {
			hasExt = true
		}
	}
	if !hasExt {
		t.Fatal("expected python external dependency edge")
	}
}
