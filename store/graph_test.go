package store_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestScanGraphSaveGet(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "graph-scan-test01"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual})

	g := graph.Graph{ScanID: scanID, Nodes: []graph.Node{{ID: "n1", Type: "file", Label: "a.go"}}, Edges: []graph.Edge{}}
	raw, _ := json.Marshal(g)
	if err := s.SaveScanGraph(ctx, store.ScanGraphRecord{
		ScanID: scanID, RepositoryID: repo.ID, GraphJSON: raw, NodeCount: 1, GeneratedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.GetScanGraph(ctx, scanID)
	if err != nil || got.NodeCount != 1 {
		t.Fatalf("get: %v count=%d", err, got.NodeCount)
	}
}

// Regression: modernc SQLite can return TEXT columns as []byte; converting
// graph_json to json.RawMessage must survive that and still unmarshal.
func TestScanGraphHandlesRawMessageFromDriver(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "graph-driver-rm-01"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual})

	if err := s.SaveScanGraph(ctx, store.ScanGraphRecord{
		ScanID: scanID, RepositoryID: repo.ID,
		GraphJSON: json.RawMessage(`{"scan_id":"` + scanID + `","nodes":[]}`),
		NodeCount: 0, GeneratedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.GetScanGraph(ctx, scanID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.GraphJSON) == 0 {
		t.Fatal("graph_json empty after read")
	}
	var back map[string]any
	if err := json.Unmarshal(got.GraphJSON, &back); err != nil {
		t.Fatalf("unmarshal GraphJSON: %v (raw=%q type=%T)", err, string(got.GraphJSON), got.GraphJSON)
	}
	if _, ok := back["nodes"]; !ok {
		t.Fatalf("graph payload lost: %q", string(got.GraphJSON))
	}
}

func TestFinishScanPersistsGraphStateSmoke(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "graph-finish-smoke1"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual})

	g := graph.Graph{
		ScanID:  scanID,
		Nodes:   []graph.Node{{ID: "n1", Type: "file", Label: "main.go"}},
		Edges:   []graph.Edge{{ID: "e1", From: "n1", To: "n1", Type: "contains"}},
		Metrics: graph.GraphMetrics{NodeCount: 1, EdgeCount: 1},
	}
	raw, _ := json.Marshal(g)
	rec := store.NewRecorder(s, nil)
	if err := rec.FinishScan(ctx, scanID, &store.ScanCompletion{
		GraphEnabled:   true,
		GraphState:     store.GraphStateAvailable,
		GraphNodeCount: 1,
		GraphEdgeCount: 1,
		GraphJSON:      raw,
	}, nil); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	gotGraph, err := s.GetScanGraph(ctx, scanID)
	if err != nil || gotGraph.NodeCount != 1 {
		t.Fatalf("graph not persisted: %v nodes=%d", err, gotGraph.NodeCount)
	}
	scan, err := s.GetScan(ctx, scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if !strings.Contains(string(scan.SummaryJSON), `"graph_state":"available"`) {
		t.Fatalf("summary missing graph_state: %s", scan.SummaryJSON)
	}

	sqlite, ok := s.(*store.SQLiteStore)
	if !ok {
		t.Fatal("expected sqlite store")
	}
	_ = sqlite.UpdateScanPipelineState(ctx, scanID, store.ScanStatusCompleted, nil)
	st, err := sqlite.GraphStatusForScan(ctx, scanID, store.DefaultGlobalSettings())
	if err != nil {
		t.Fatalf("graph status: %v", err)
	}
	if st.State != store.GraphStateAvailable {
		t.Fatalf("expected available graph state, got %q", st.State)
	}
}
