package graph_test

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/graph"
)

func TestOrphanFileDescriptionIncludesPathAndCounts(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "lonely.go", Content: "package lonely\n", Language: "go"},
		},
	}, testCfg(), nil)
	var orphan *graph.GraphFinding
	for i := range findings {
		if findings[i].RuleID == "GRAPH-ORPHAN-FILE" && findings[i].File == "lonely.go" {
			orphan = &findings[i]
			break
		}
	}
	if orphan == nil {
		t.Fatal("expected orphan file finding")
	}
	if !strings.Contains(orphan.Title, "lonely.go") {
		t.Fatalf("title should include path: %s", orphan.Title)
	}
	if !strings.Contains(orphan.Description, "Inbound import edges") {
		t.Fatalf("description missing edge counts: %s", orphan.Description)
	}
	if !strings.Contains(orphan.Description, "Troubleshooting:") {
		t.Fatal("expected troubleshooting section")
	}
	if orphan.Detail.RuleID != "GRAPH-ORPHAN-FILE" {
		t.Fatalf("detail rule: %+v", orphan.Detail)
	}
	if strings.Contains(strings.ToLower(orphan.Description), "password") {
		t.Fatal("graph detail must not include secret-like placeholders")
	}
}

func TestGraphFindingDetailJSONSafe(t *testing.T) {
	_, findings := graph.Build(context.Background(), graph.BuildInput{
		Files: []graph.FileInput{
			{Path: "main.go", Content: "package main\nfunc main() {}\n", Language: "go"},
			{Path: "orphan.go", Content: "package orphan\n", Language: "go"},
		},
	}, testCfg(), nil)
	for _, f := range findings {
		if f.RuleID != "GRAPH-ORPHAN-FILE" {
			continue
		}
		raw := f.Detail.JSON()
		if !strings.Contains(raw, `"rule_id":"GRAPH-ORPHAN-FILE"`) {
			t.Fatalf("expected json detail: %s", raw)
		}
		return
	}
	t.Fatal("no graph orphan finding")
}
