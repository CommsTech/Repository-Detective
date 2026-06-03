package reconcile_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/calibration"
	"git.commsnet.org/commstech/bugbot/reconcile"
	"git.commsnet.org/commstech/bugbot/store"
)

type fakeForge struct {
	comments []string
	labels   [][]string
	closed   []int
}

func (f *fakeForge) AddIssueLabels(_ context.Context, _, _ string, _ int, labels []string) error {
	f.labels = append(f.labels, labels)
	return nil
}
func (f *fakeForge) CreateIssueComment(_ context.Context, _, _ string, _ int, body string) error {
	f.comments = append(f.comments, body)
	return nil
}
func (f *fakeForge) CloseIssue(_ context.Context, _, _ string, issueNumber int) error {
	f.closed = append(f.closed, issueNumber)
	return nil
}
func (f *fakeForge) AnnotateCalibration(_ context.Context, _, _, _ string, _ int, _ bool, _ string) error {
	return nil
}

func seedRepoIssue(t *testing.T, s store.QueryStore) (store.Repository, store.Finding, store.ExternalIssue) {
	t.Helper()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	now := time.Now().UTC()
	f, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-rec-1", Title: "test", Severity: "medium",
		Source: "graph", RuleID: "GRAPH-ORPHAN-FILE", Status: store.FindingStatusOpen,
		FirstSeenAt: now, LastSeenAt: now,
	})
	scanID := "scan-rec-1"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, Status: store.ScanStatusCompleted, StartedAt: now, TriggerType: store.TriggerManual})
	_ = s.AddFindingInstance(ctx, store.FindingInstance{FindingID: f.ID, ScanID: scanID})
	ei, _ := s.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: f.ID, ForgeType: "gitea", IssueNumber: 42, IssueURL: "http://x/42", State: "open",
	})
	return repo, f, ei
}

func TestPreviewStillPresent(t *testing.T) {
	s := openTestStore(t)
	repo, _, _ := seedRepoIssue(t, s)
	forge := &fakeForge{}
	eng := reconcile.NewEngine(s, calibration.NewMatcher(s), forge, reconcile.Config{Comment: true, PublicBasePath: "/ui"})
	result, err := eng.Preview(context.Background(), repo.ID)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Status != reconcile.StatusStillPresent && result.Items[0].Status != reconcile.StatusStaleRule {
		t.Fatalf("unexpected status %s", result.Items[0].Status)
	}
	if len(forge.comments) != 0 {
		t.Fatal("preview must not comment on forge")
	}
}

func TestApplyDoesNotCloseWithoutVerifiedPolicy(t *testing.T) {
	s := openTestStore(t)
	repo, f, _ := seedRepoIssue(t, s)
	ctx := context.Background()
	// second scan without finding instance => absent
	now := time.Now().UTC()
	scan2 := "scan-rec-2"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scan2, RepositoryID: repo.ID, Status: store.ScanStatusCompleted, StartedAt: now.Add(time.Minute), TriggerType: store.TriggerManual})
	_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{{ScanID: scan2, ScannerName: "graph", Status: "completed"}})
	forge := &fakeForge{}
	eng := reconcile.NewEngine(s, calibration.NewMatcher(s), forge, reconcile.Config{
		Comment: true, CloseVerified: false, PublicBasePath: "/ui",
	})
	_, _ = f, forge
	result, err := eng.Apply(ctx, repo.ID)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("errors: %v", result.Errors)
	}
	if len(forge.closed) != 0 {
		t.Fatal("must not close without close_verified policy")
	}
}

func TestPreviewManyIssuesNoForgeCalls(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "bulk", FullName: "o/bulk", ConnectedRepo: true})
	now := time.Now().UTC()
	scanID := "scan-bulk-1"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, Status: store.ScanStatusCompleted, StartedAt: now, TriggerType: store.TriggerManual})
	_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{{ScanID: scanID, ScannerName: "graph", Status: "completed"}})
	forge := &fakeForge{}
	eng := reconcile.NewEngine(s, calibration.NewMatcher(s), forge, reconcile.Config{Comment: true, PublicBasePath: "/ui"})
	const n = 100
	for i := 0; i < n; i++ {
		fp := fmt.Sprintf("fp-bulk-%d", i)
		f, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID, Fingerprint: fp, Title: "t", Severity: "low",
			Source: "graph", RuleID: "GRAPH-ORPHAN-FILE", Status: store.FindingStatusOpen,
			FirstSeenAt: now, LastSeenAt: now,
		})
		if err != nil {
			t.Fatalf("finding %d: %v", i, err)
		}
		_ = s.AddFindingInstance(ctx, store.FindingInstance{FindingID: f.ID, ScanID: scanID})
		_, _ = s.UpsertExternalIssue(ctx, store.ExternalIssue{
			FindingID: f.ID, ForgeType: "gitea", IssueNumber: 1000 + i,
			IssueURL: fmt.Sprintf("http://x/%d", 1000+i), State: "open",
		})
	}
	start := time.Now()
	result, err := eng.Preview(ctx, repo.ID)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(result.Items) != n {
		t.Fatalf("expected %d items, got %d", n, len(result.Items))
	}
	if len(forge.comments) != 0 || len(forge.labels) != 0 || len(forge.closed) != 0 {
		t.Fatal("preview must not call forge")
	}
	if elapsed := time.Since(start); elapsed > 60*time.Second {
		t.Fatalf("preview took %s, want under 60s", elapsed)
	}
}

func openTestStore(t *testing.T) store.QueryStore {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: dir + "/t.db"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
