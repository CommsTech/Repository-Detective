package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/graph"
	"git.commsnet.org/commstech/bugbot/store"
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
