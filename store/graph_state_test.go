package store_test

import (
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestResolveGraphStatusAvailable(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanID: "s1", RepoID: 1,
		ScanStatus: store.ScanStatusCompleted, GraphEnabled: true, AnalysisDepth: 2,
		GraphJSON: []byte(`{"nodes":[],"edges":[]}`), NodeCount: 1, EdgeCount: 0,
	})
	if st.State != store.GraphStateAvailable {
		t.Fatalf("expected available, got %s", st.State)
	}
	if string(st.Graph) == "null" {
		t.Fatal("expected graph payload")
	}
}

func TestResolveGraphStatusMissing(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanStatus: store.ScanStatusCompleted,
		GraphEnabled: true, AnalysisDepth: 3,
	})
	if st.State != store.GraphStateMissing {
		t.Fatalf("expected missing, got %s", st.State)
	}
	if st.NextAction == "" {
		t.Fatal("expected next action")
	}
}

func TestResolveGraphStatusDisabled(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanStatus: store.ScanStatusCompleted,
		GraphEnabled: false, AnalysisDepth: 3,
	})
	if st.State != store.GraphStateDisabled {
		t.Fatalf("expected disabled, got %s", st.State)
	}
}

func TestResolveGraphStatusFailed(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanStatus: store.ScanStatusCompleted,
		GraphEnabled: true, AnalysisDepth: 2, GraphError: "graph build timed out",
	})
	if st.State != store.GraphStateFailed {
		t.Fatalf("expected failed, got %s", st.State)
	}
}

func TestResolveGraphStatusPending(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanStatus: store.ScanStatusStarted,
		GraphEnabled: true, AnalysisDepth: 2,
	})
	if st.State != store.GraphStatePending {
		t.Fatalf("expected pending, got %s", st.State)
	}
}

func TestResolveGraphStatusTruncated(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{
		ScanFound: true, RepoFound: true, ScanStatus: store.ScanStatusCompleted,
		GraphEnabled: true, AnalysisDepth: 2, Truncated: true,
		GraphJSON: []byte(`{"nodes":[{"id":"n1"}],"edges":[]}`), NodeCount: 1,
	})
	if st.State != store.GraphStateTruncated {
		t.Fatalf("expected truncated, got %s", st.State)
	}
}

func TestResolveGraphStatusScanNotFound(t *testing.T) {
	st := store.ResolveGraphStatus(store.GraphStatusInput{RepoFound: true})
	if st.State != store.GraphStateScanNotFound {
		t.Fatal(st.State)
	}
}

func TestGraphMetaFromSummary(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"graph_enabled":  true,
		"analysis_depth": 2,
		"graph_error":    "timeout",
	})
	enabled, depth, _, errMsg := store.GraphMetaFromSummary(raw)
	if !enabled || depth != 2 || errMsg != "timeout" {
		t.Fatalf("enabled=%v depth=%d err=%q", enabled, depth, errMsg)
	}
}
