package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

func TestReconciliationSummaryReportOnly(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	sqlite := s.(*store.SQLiteStore)

	repo, _ := sqlite.UpsertRepository(ctx, store.Repository{Owner: "amber", Name: "app", FullName: "amber/app", ConnectedRepo: true})
	f1, _ := sqlite.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-secret", Title: "hardcoded secret",
		Severity: "high", Source: "gitleaks", Status: store.FindingStatusOpen,
	})
	_, _ = sqlite.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-orphan", Title: "orphan file",
		Severity: "low", Source: "graph", Status: store.FindingStatusOpen,
	})
	_, _ = sqlite.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: f1.ID, ForgeType: "gitea", IssueNumber: 205,
		IssueURL: "https://git.example.com/amber/app/issues/205", State: "open",
	})

	scanID := "scan-report-only-1"
	summary := mustJSON(map[string]any{
		"issues_found": 2, "persistence_status": store.PersistenceStatusComplete,
		"issue_sync_status": store.IssueSyncStatusSkipped, "dry_run_report_only": true,
		"persistence_persisted_count": 2,
	})
	_, _ = sqlite.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Ref: "main", Status: store.ScanStatusCompleted, SummaryJSON: summary,
	})

	rec := store.NewRecorder(s, logrus.New())
	_, _ = rec.RecordFindings(ctx, repo.ID, scanID, []ai.CodeIssue{
		{Fingerprint: "fp-secret", Title: "hardcoded secret", Severity: "high", Source: "gitleaks"},
		{Fingerprint: "fp-orphan", Title: "orphan file", Severity: "low", Source: "graph"},
	})

	sum, err := sqlite.ReconciliationSummaryForScan(ctx, repo.ID, scanID, false)
	if err != nil {
		t.Fatal(err)
	}
	if sum.ScanFindingsTotal != 2 {
		t.Fatalf("scan findings want 2 got %d", sum.ScanFindingsTotal)
	}
	if sum.MappedOpenIssues != 1 {
		t.Fatalf("mapped open want 1 got %d", sum.MappedOpenIssues)
	}
	if sum.FindingsWithoutIssue != 1 {
		t.Fatalf("findings without issue want 1 got %d", sum.FindingsWithoutIssue)
	}
	if !sum.DryRunReportOnly {
		t.Fatal("expected dry_run_report_only")
	}
	if !sum.CountsDifferExpected {
		t.Fatal("expected counts differ to be normal")
	}
	if sum.MismatchWarning != "" {
		t.Fatalf("unexpected mismatch warning: %q", sum.MismatchWarning)
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
