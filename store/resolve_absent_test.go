package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
)

// A completed scan that no longer reports a previously open finding must close
// it, while findings it still reports stay open. Absence must never be recorded
// as "resolved_verified"; that status is reserved for evidence-backed closure.
func TestCompletedScanResolvesAbsentFindings(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "absent.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	}()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, id := range []string{"scan-first", "scan-second"} {
		if _, err := s.CreateScan(ctx, store.Scan{
			ID: id, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
			Status: "started", StartedAt: now, Ref: "main",
		}); err != nil {
			t.Fatal(err)
		}
	}
	rec := store.NewRecorder(s, nil)
	first := []ai.CodeIssue{
		{Title: "still present", Category: "security", Severity: "high", Confidence: 0.9, Source: "gosec", Fingerprint: "fp-present"},
		{Title: "will be fixed", Category: "security", Severity: "high", Confidence: 0.9, Source: "gosec", Fingerprint: "fp-fixed"},
	}
	if _, err := rec.RecordFindings(ctx, repo.ID, "scan-first", first); err != nil {
		t.Fatal(err)
	}
	second := []ai.CodeIssue{first[0]}
	if _, err := rec.RecordFindings(ctx, repo.ID, "scan-second", second); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListFindings(ctx, store.FindingFilter{RepositoryID: repo.ID, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, item := range items {
		status[item.Fingerprint] = item.Status
	}
	if status["fp-present"] != "open" {
		t.Errorf("still-present finding changed status: %q", status["fp-present"])
	}
	if status["fp-fixed"] != "resolved_absent" {
		t.Errorf("absent finding not closed: %q", status["fp-fixed"])
	}
	for _, item := range items {
		if item.Status == "resolved_verified" {
			t.Errorf("absence must not be recorded as verified closure: %+v", item)
		}
	}
}
