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
	if deduped[0].ClusterID != "cluster-000" {
		t.Fatalf("expected cluster-000, got %s", deduped[0].ClusterID)
	}
	if len(deduped[0].Files) != 2 {
		t.Fatalf("expected 2 files in group, got %d", len(deduped[0].Files))
	}
}

func TestDedupMergesDifferentCategoriesSameBlock(t *testing.T) {
	// Known risk (Phase 7): spatial dedup merges by file + 10-line block, not category/rule.
	// Different vulnerability classes on the same line block are collapsed intentionally today.
	engine := &Engine{}

	validated := []ValidatedFinding{
		{
			CandidateFinding: CandidateFinding{
				ID:          "sql",
				Hypothesis:  "SQL injection via concatenation",
				File:        "handler.go",
				Line:        42,
				Category:    "sql_injection",
				AuditorType: "static",
				Severity:    "high",
				Confidence:  0.92,
			},
		},
		{
			CandidateFinding: CandidateFinding{
				ID:          "secret",
				Hypothesis:  "Hardcoded API key",
				File:        "handler.go",
				Line:        42,
				Category:    "hardcoded_secret",
				AuditorType: "static",
				Severity:    "high",
				Confidence:  0.90,
			},
		},
	}

	deduped := engine.Dedup(validated)
	if len(deduped) != 1 {
		t.Fatalf("documented behavior: expected 1 merged finding, got %d", len(deduped))
	}
	if deduped[0].Category != "sql_injection" {
		t.Fatalf("expected highest-confidence category to win, got %s", deduped[0].Category)
	}
	if len(deduped[0].Related) == 0 {
		t.Fatal("expected merged hypothesis lineage in Related/description")
	}
}

func TestDedupEmpty(t *testing.T) {
	engine := &Engine{}
	if got := engine.Dedup(nil); len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}
