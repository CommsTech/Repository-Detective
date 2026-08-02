package store_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func graphGlobal() store.GlobalSettingsSnapshot {
	g := store.DefaultGlobalSettings()
	g.EnableCodeGraph = true
	g.GraphMaxNodes = 5000
	g.GraphMaxEdges = 15000
	g.GraphTimeoutSeconds = 120
	return g
}

func TestMigrationAddsGraphColumns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "g", FullName: "o/g"})
	disabled := false
	maxNodes := 2000
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID: repo.ID, EnableCodeGraph: &disabled, GraphMaxNodes: &maxNodes,
	}); err != nil {
		t.Fatalf("save graph settings: %v", err)
	}
	got, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got.EnableCodeGraph == nil || *got.EnableCodeGraph {
		t.Fatal("expected stored enable_code_graph=false")
	}
	if got.GraphMaxNodes == nil || *got.GraphMaxNodes != 2000 {
		t.Fatalf("expected graph_max_nodes=2000, got %v", got.GraphMaxNodes)
	}
}

func TestGraphSettingsInheritGlobal(t *testing.T) {
	global := graphGlobal()
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if !effective.EnableCodeGraph {
		t.Fatal("expected global graph enabled")
	}
	if effective.GraphMaxNodes != 5000 {
		t.Fatalf("expected max nodes 5000, got %d", effective.GraphMaxNodes)
	}
}

func TestGraphSettingsOverrideGlobal(t *testing.T) {
	global := graphGlobal()
	graphOff := false
	nodes := 1000
	edges := 3000
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{
		EnableCodeGraph: &graphOff,
		GraphMaxNodes:   &nodes,
		GraphMaxEdges:   &edges,
	})
	if effective.EnableCodeGraph {
		t.Fatal("repo should disable code graph")
	}
	if effective.GraphMaxNodes != 1000 || effective.GraphMaxEdges != 3000 {
		t.Fatalf("unexpected overrides: nodes=%d edges=%d", effective.GraphMaxNodes, effective.GraphMaxEdges)
	}
}

func TestInvalidGraphThresholdRejected(t *testing.T) {
	bad := 50
	err := store.ValidateSettingsUpdate(store.SettingsUpdate{GraphMaxNodes: &bad})
	if err == nil {
		t.Fatal("expected validation error for graph_max_nodes below minimum")
	}
}

func TestSanitizeInvalidStoredGraphThreshold(t *testing.T) {
	global := graphGlobal()
	bad := 10
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{GraphMaxNodes: &bad})
	if effective.GraphMaxNodes != global.GraphMaxNodes {
		t.Fatalf("invalid stored threshold should fall back to global, got %d", effective.GraphMaxNodes)
	}
}
