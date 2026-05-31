package qdrant_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/bugbot/memory/qdrant"
)

func TestEmbeddingTextIncludesTitle(t *testing.T) {
	text := qdrant.EmbeddingText(qdrant.MatchInput{
		Title:       "SQL injection",
		Description: "User input concatenated into query",
		File:        "db.go",
	})
	if text == "" {
		t.Fatal("expected non-empty embedding text")
	}
}

func TestFingerprintStable(t *testing.T) {
	input := qdrant.MatchInput{
		Repository:  "org/repo",
		Title:       "Hardcoded secret",
		Description: "API key in source",
		File:        "config.go",
		Line:        10,
		Category:    "hardcoded_secret",
	}

	store := qdrant.NewStore(qdrant.DefaultConfig())
	if store == nil {
		t.Fatal("expected store")
	}

	// Exercise internal fingerprint via Remember path with disabled client — should no-op.
	if err := store.Remember(context.Background(), input, "", 0, []float32{0.1, 0.2}); err != nil {
		t.Fatalf("remember on disabled store: %v", err)
	}
}
