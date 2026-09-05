package store_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingQualityMetricsWindows(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, finding := seedRepoAndFinding(t, s)

	now := time.Now().UTC()
	old := now.Add(-40 * 24 * time.Hour)
	_, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-old-window", Category: "secret", Severity: "medium",
		Source: "gitleaks", RuleID: "generic", Title: "old", Status: store.FindingStatusFalsePositive,
		FirstSeenScanID: "s0", LastSeenScanID: "s0", FirstSeenAt: old, LastSeenAt: old,
	})
	if err != nil {
		t.Fatalf("old finding: %v", err)
	}
	_ = finding

	all, err := s.FindingQualityMetrics(ctx, store.FindingQualityWindowAll)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if all.FindingsOpened < 2 {
		t.Fatalf("expected >=2 opened all-time, got %d", all.FindingsOpened)
	}
	if all.FalsePositiveDispositions < 1 {
		t.Fatalf("expected FP disposition, got %d", all.FalsePositiveDispositions)
	}
	if all.Definitions["false_positive_disposition_rate"] == "" {
		t.Fatal("missing metric definitions")
	}

	w7, err := s.FindingQualityMetrics(ctx, store.FindingQualityWindow7d)
	if err != nil {
		t.Fatalf("7d: %v", err)
	}
	if w7.FindingsOpened < 1 {
		t.Fatalf("expected recent finding in 7d window, got %d", w7.FindingsOpened)
	}
	if w7.FindingsOpened != 1 {
		t.Fatalf("expected 1 opened in 7d (seed only), got %d all=%d", w7.FindingsOpened, all.FindingsOpened)
	}
}
