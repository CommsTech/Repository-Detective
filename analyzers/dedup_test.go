package analyzers

import "testing"

func TestDedupKeepsHighestConfidence(t *testing.T) {
	engine := &Engine{}

	validated := []ValidatedFinding{
		{
			CandidateFinding: CandidateFinding{
				ID:         "a",
				Hypothesis: "SQL injection",
				File:       "db.go",
				Line:       42,
				Severity:   "high",
				Confidence: 0.7,
			},
		},
		{
			CandidateFinding: CandidateFinding{
				ID:         "b",
				Hypothesis: "SQL injection duplicate",
				File:       "db.go",
				Line:       44,
				Severity:   "high",
				Confidence: 0.95,
			},
		},
	}

	deduped := engine.Dedup(validated)
	if len(deduped) != 1 {
		t.Fatalf("expected 1 deduped finding, got %d", len(deduped))
	}
	if deduped[0].Confidence != 0.95 {
		t.Fatalf("expected confidence 0.95, got %f", deduped[0].Confidence)
	}
	if len(deduped[0].Files) != 2 {
		t.Fatalf("expected 2 files in group, got %d", len(deduped[0].Files))
	}
}

func TestDedupEmpty(t *testing.T) {
	engine := &Engine{}
	if got := engine.Dedup(nil); len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}
