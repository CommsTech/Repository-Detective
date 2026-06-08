package qdrant

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultConfigUsesCAHFindings(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Collection != "cah_findings" {
		t.Fatalf("collection=%q want cah_findings", cfg.Collection)
	}
	if cfg.Enabled {
		t.Fatal("qdrant should be disabled by default")
	}
}

func TestStoreDisabledNoWrites(t *testing.T) {
	store := NewStore(DefaultConfig())
	input := MatchInput{Repository: "o/r", Title: "test"}
	if err := store.Remember(context.Background(), input, "", 0, []float32{0.1}); err != nil {
		t.Fatalf("disabled remember should no-op: %v", err)
	}
	match, err := store.FindSimilar(context.Background(), input, []float32{0.1})
	if err != nil || match != nil {
		t.Fatalf("disabled find should no-op, match=%v err=%v", match, err)
	}
}

func TestValidateVectorLenBlocksMismatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	client := NewClient(cfg)
	if err := client.ValidateVectorLen(512); !errors.Is(err, ErrCollectionMismatch) {
		t.Fatalf("expected collection mismatch for wrong vector len, got %v", err)
	}
}

func TestParseHitPayloadCAHFields(t *testing.T) {
	parsed := ParseHitPayload(map[string]any{
		"description": "Hardcoded credential-like value detected",
		"gitea_issue": float64(42),
		"issue_url":   "https://git.example/o/r/issues/42",
		"cluster_id":  "cluster-001",
	})
	if parsed.IssueNumber != 42 || parsed.IssueURL == "" || parsed.Title == "" {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
}
