package issuelink_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/issuelink"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/sirupsen/logrus"
)

type stubForge struct {
	issues []issues.ForgeIssue
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
func (s *stubForge) AddIssueLabels(context.Context, string, string, int, []string) error { return nil }

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
