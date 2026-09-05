package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestGetScanGraphReturnsStructuredState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	scanID := "graphstate00001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	g := graph.Graph{ScanID: scanID, Nodes: []graph.Node{{ID: "n1", Type: "file", Label: "main.go"}}, Edges: []graph.Edge{}}
	raw, _ := json.Marshal(g)
	_ = s.SaveScanGraph(ctx, store.ScanGraphRecord{
		ScanID: scanID, RepositoryID: repo.ID, GraphJSON: raw, NodeCount: 1, GeneratedAt: time.Now().UTC(),
	})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scans/"+scanID+"/graph", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var body store.GraphStatus
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.State != store.GraphStateAvailable {
		t.Fatalf("expected available, got %s", body.State)
	}
	if len(body.Graph) == 0 {
		t.Fatal("expected graph payload")
	}
}

func TestGetScanGraphMissingState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	scanID := "graphstate00002"
	summary, _ := json.Marshal(map[string]any{
		"graph_enabled":  true,
		"analysis_depth": 2,
		"effective_settings": map[string]any{
			"enable_code_graph": true,
			"analysis_depth":    2,
		},
	})
	_, _ = s.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, SummaryJSON: summary,
	})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scans/"+scanID+"/graph", nil)
	r.ServeHTTP(w, req)
	var body store.GraphStatus
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.State != store.GraphStateMissing {
		t.Fatalf("expected missing, got %s", body.State)
	}
}
