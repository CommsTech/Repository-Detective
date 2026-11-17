package store_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingsQueueOneRowPerFindingWithMultipleForgeIssues(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "same-finding", Title: "CVE issue", Severity: "critical", Status: "open", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		num   int
		state string
	}{{3, "closed"}, {4, "open"}} {
		_, err = s.UpsertExternalIssue(ctx, store.ExternalIssue{FindingID: f.ID, IssueNumber: v.num, IssueURL: "https://example.invalid/issue", State: v.state})
		if err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.ListFindings(ctx, store.FindingFilter{PriorityOnly: true, Status: "open", Limit: 20})
	if err != nil || len(items) != 1 {
		t.Fatalf("duplicate finding for two forge issues: got %d err=%v", len(items), err)
	}
	if items[0].ExternalIssueNumber != 4 {
		t.Fatalf("expected open issue #4, got #%d", items[0].ExternalIssueNumber)
	}
	detail, err := s.GetFindingDetail(ctx, f.ID)
	if err != nil || detail.ExternalIssueNumber != 4 {
		t.Fatalf("detail must match queue's chosen open issue, got #%d err=%v", detail.ExternalIssueNumber, err)
	}
}
