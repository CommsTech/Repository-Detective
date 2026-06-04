package qdrant

import (
	"strings"
	"testing"
)

func TestRedactSummaryRemovesSecrets(t *testing.T) {
	raw := `AWS_SECRET_ACCESS_KEY=actual_secret_here in config.env.template`
	got := RedactSummary("Hardcoded secret", raw, raw)
	if strings.Contains(got, "actual_secret_here") {
		t.Fatalf("secret leaked: %q", got)
	}
}

func TestBuildCAHPayloadNoRawCode(t *testing.T) {
	payload := BuildCAHPayload(MatchInput{
		Repository:  "o/r",
		Title:       "SQL injection",
		Description: "query := \"SELECT * FROM users WHERE id=\" + userInput",
		File:        "app/db.go",
		Line:        10,
		Severity:    "high",
		Category:    "security",
		Source:      "semgrep",
		RuleID:      "SEC-SQL-001",
		Confidence:  0.9,
	}, "", 0, "scan-1")
	if PayloadContainsRawSecrets(payload) {
		t.Fatalf("payload looks sensitive: %+v", payload)
	}
	if payload.FindingID == "" || payload.ID != payload.FindingID {
		t.Fatalf("expected stable point id, got %+v", payload)
	}
	if payload.Severity != "High" {
		t.Fatalf("severity mapping: %q", payload.Severity)
	}
}

func TestEmbeddingTextUsesNormalizedFields(t *testing.T) {
	text := EmbeddingText(MatchInput{
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

func TestPayloadContainsRawSecretsDetectsLeak(t *testing.T) {
	payload := CAHFindingPayload{Description: `password = "supersecret12345"`}
	if !PayloadContainsRawSecrets(payload) {
		t.Fatal("expected secret detection")
	}
}
