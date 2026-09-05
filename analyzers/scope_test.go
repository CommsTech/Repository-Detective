package analyzers

import (
	"context"
	"testing"
)

func TestShouldAnalyzeFileSkipsVendorAndBinaries(t *testing.T) {
	engine := &Engine{config: &Config{
		SkipPatterns: []string{"coverage"},
	}}

	cases := []struct {
		path    string
		analyze bool
	}{
		{"main.go", true},
		{"vendor/lib.go", false},
		{"node_modules/pkg/index.js", false},
		{"assets/logo.png", false},
		{"coverage/out.txt", false},
	}

	for _, tc := range cases {
		got := engine.shouldAnalyzeFile(tc.path)
		if got != tc.analyze {
			t.Fatalf("path %q: expected analyze=%v, got %v", tc.path, tc.analyze, got)
		}
	}
}

func TestResolveAnalyzableFilesScoped(t *testing.T) {
	engine := &Engine{config: &Config{}}

	files, err := engine.resolveAnalyzableFiles(context.Background(), "owner", "repo", "main", []string{
		"main.go",
		"vendor/x.go",
		"readme.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "main.go" {
		t.Fatalf("expected only main.go, got %v", files)
	}
}
