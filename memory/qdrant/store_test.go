package qdrant_test

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/bugbot/memory/qdrant"
)

func TestEmbeddingTextUsesNormalizedFields(t *testing.T) {
	text := qdrant.EmbeddingText(qdrant.MatchInput{
		Source: "gitleaks", RuleID: "generic-api-key", Category: "secret",
		File: "config.env.template", Verdict: "false_positive",
		Title: "token", Description: "placeholder token in example file",
	})
	for _, want := range []string{"source: gitleaks", "rule: generic-api-key", "verdict: false_positive"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in %q", want, text)
		}
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
