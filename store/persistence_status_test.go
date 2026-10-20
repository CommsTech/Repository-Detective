package store

import "testing"

func TestPipelineStateFromSummary(t *testing.T) {
	raw := []byte(`{"issues_found":42,"persistence_status":"complete","persistence_expected_count":42,"persistence_persisted_count":42}`)
	state := PipelineStateFromSummary(raw)
	if state.IssuesFound != 42 || !state.IsPersistenceComplete() {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestMergeSummaryPipelineFields(t *testing.T) {
	raw := []byte(`{"issues_found":10}`)
	merged, err := MergeSummaryPipelineFields(raw, map[string]any{
		"persistence_status": PersistenceStatusComplete,
	})
	if err != nil {
		t.Fatal(err)
	}
	state := PipelineStateFromSummary(merged)
	if state.IssuesFound != 10 || state.PersistenceStatus != PersistenceStatusComplete {
		t.Fatalf("merge failed: %+v", state)
	}
}
