package issuelink_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/issuelink"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

type stubForge struct {
	issues     []issues.ForgeIssue
	issueByNum map[int]issues.ForgeIssue
	getErr     error
}

func (s *stubForge) ListOpenLabeledIssues(_ context.Context, _, _ string, _ []string, limit, page int) ([]issues.ForgeIssue, error) {
	if page > 1 {
		return nil, nil
	}
	if limit <= 0 || limit >= len(s.issues) {
		return s.issues, nil
	}
	return s.issues[:limit], nil
}
func (s *stubForge) ListOpenIssues(_ context.Context, _, _ string, limit, page int) ([]issues.ForgeIssue, error) {
	return s.ListOpenLabeledIssues(context.Background(), "", "", nil, limit, page)
}
func (s *stubForge) CreateIssue(context.Context, string, string, string, string, []string) (*issues.ForgeIssue, error) {
	return &issues.ForgeIssue{Number: 999}, nil
}
func (s *stubForge) CreateIssueComment(context.Context, string, string, int, string) error {
	return nil
}
func (s *stubForge) EditIssueBody(context.Context, string, string, int, string) error    { return nil }
func (s *stubForge) AddIssueLabels(context.Context, string, string, int, []string) error { return nil }
func (s *stubForge) GetIssue(_ context.Context, _, _ string, issueNumber int) (*issues.ForgeIssue, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.issueByNum != nil {
		if issue, ok := s.issueByNum[issueNumber]; ok {
			cp := issue
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("gitea API returned status 404: not found")
}

func TestRefreshExternalIssueStatesMarksClosed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "refresh.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	f, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "rd-refresh1", Title: "x", Severity: "high",
		Source: "trivy", Status: store.FindingStatusOpen, FirstSeenAt: now, LastSeenAt: now,
	})
	_, _ = s.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: f.ID, ForgeType: "gitea", IssueNumber: 214,
		IssueURL: "http://git/o/r/issues/214", State: "open",
	})

	forge := &stubForge{issueByNum: map[int]issues.ForgeIssue{
		214: {Number: 214, State: "closed", HTMLURL: "http://git/o/r/issues/214"},
	}}
	result, err := issuelink.RefreshExternalIssueStates(ctx, &issuelink.Store{Query: s}, forge, "o", "r", repo.ID, "gitea", "scan-rf", logrus.New())
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if result.Updated != 1 {
		t.Fatalf("updated=%d want 1 examined=%d errors=%v", result.Updated, result.Examined, result.Errors)
	}
	ext, err := s.GetExternalIssueByIssueNumber(ctx, repo.ID, "gitea", 214)
	if err != nil || ext.State != "closed" {
		t.Fatalf("expected closed mapping, got %+v err=%v", ext, err)
	}
	// Finding stays open until reconcile/evidence closure.
	detail, err := s.GetFindingDetail(ctx, f.ID)
	if err != nil || detail.Status != store.FindingStatusOpen {
		t.Fatalf("finding status should remain open, got %+v err=%v", detail, err)
	}
}

func TestBackfillExternalIssueMapping(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "backfill.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	f, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "rd-backfill1", Title: "x", Severity: "medium",
		Source: "health", RuleID: "HEALTH-TEST", FirstSeenAt: now, LastSeenAt: now,
	})

	forge := &stubForge{issues: []issues.ForgeIssue{{
		Number: 55, HTMLURL: "http://git/o/r/issues/55",
		Body: "## Tracking\n\n- Repository Detective fingerprint: rd-backfill1\n",
	}}}

	result, err := issuelink.BackfillExternalIssueMappings(ctx, &issuelink.Store{Query: s}, forge, "o", "r", repo.ID, "gitea", "scan-bf", logrus.New())
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.Backfilled != 1 {
		t.Fatalf("backfilled=%d want 1", result.Backfilled)
	}
	ext, err := s.GetExternalIssueByFingerprint(ctx, repo.ID, "gitea", "rd-backfill1")
	if err != nil || ext.IssueNumber != 55 {
		t.Fatalf("mapping missing: %+v err=%v", ext, err)
	}
	events, _ := s.ListLifecycleEventsByFinding(ctx, f.ID)
	found := false
	for _, ev := range events {
		if ev.EventType == store.LifecycleEventExternalIssueMappingBackfilled {
			found = true
		}
	}
	if !found {
		t.Fatal("expected backfill lifecycle event")
	}
}

func TestMappedIssueReturnsLocalLink(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "map.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	f, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "rd-idem1", Title: "x", Severity: "high",
		Source: "health", FirstSeenAt: now, LastSeenAt: now,
	})
	_, _ = s.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID: f.ID, ForgeType: "gitea", IssueNumber: 12, IssueURL: "http://git/o/r/issues/12", State: "open",
	})

	num, url, ok := issuelink.MappedIssue(ctx, &issuelink.Store{Query: s}, repo.ID, "gitea", "rd-idem1")
	if !ok || num != 12 || url == "" {
		t.Fatalf("mapped=%d url=%q ok=%v", num, url, ok)
	}
}
