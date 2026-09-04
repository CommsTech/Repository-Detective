package issues

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

type fakePRComments struct {
	mu       sync.Mutex
	nextID   int64
	byPR     map[int][]CommentRef
	listErr  error
	createErr error
	editErr  error
	deleteErr error
	creates  int
	edits    int
}

func newFakePRComments() *fakePRComments {
	return &fakePRComments{nextID: 1, byPR: map[int][]CommentRef{}}
}

func (f *fakePRComments) ListIssueComments(_ context.Context, _, _ string, issueNumber int) ([]CommentRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := append([]CommentRef(nil), f.byPR[issueNumber]...)
	return out, nil
}

func (f *fakePRComments) CreateIssueComment(_ context.Context, _, _ string, issueNumber int, body string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.creates++
	id := f.nextID
	f.nextID++
	f.byPR[issueNumber] = append(f.byPR[issueNumber], CommentRef{ID: id, Body: body})
	return id, nil
}

func (f *fakePRComments) EditIssueComment(_ context.Context, _, _ string, commentID int64, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.editErr != nil {
		return f.editErr
	}
	f.edits++
	for pr, list := range f.byPR {
		for i := range list {
			if list[i].ID == commentID {
				list[i].Body = body
				f.byPR[pr] = list
				return nil
			}
		}
	}
	return errors.New("comment not found")
}

func (f *fakePRComments) DeleteIssueComment(_ context.Context, _, _ string, commentID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for pr, list := range f.byPR {
		kept := list[:0]
		for _, c := range list {
			if c.ID != commentID {
				kept = append(kept, c)
			}
		}
		f.byPR[pr] = kept
	}
	return nil
}

func TestUpsertPRPolicySummaryCreateThenUpdate(t *testing.T) {
	api := newFakePRComments()
	body1 := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET", ScanID: "s1", CommitSHA: "aaa"})
	r1 := UpsertPRPolicySummary(context.Background(), api, "o", "r", 7, body1)
	if r1.Action != "created" || r1.Err != nil {
		t.Fatalf("first: %+v", r1)
	}
	body2 := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "ACTION_REQUIRED", ScanID: "s2", CommitSHA: "bbb"})
	r2 := UpsertPRPolicySummary(context.Background(), api, "o", "r", 7, body2)
	if r2.Action != "updated" || r2.Err != nil {
		t.Fatalf("second: %+v", r2)
	}
	if api.creates != 1 || api.edits != 1 {
		t.Fatalf("creates=%d edits=%d", api.creates, api.edits)
	}
	list, _ := api.ListIssueComments(context.Background(), "o", "r", 7)
	if len(list) != 1 {
		t.Fatalf("want 1 comment, got %d", len(list))
	}
	if !strings.Contains(list[0].Body, "ACTION_REQUIRED") || !strings.Contains(list[0].Body, "bbb") {
		t.Fatalf("body not updated: %s", list[0].Body)
	}
}

func TestUpsertLeavesUserAndMalformedMarkersAlone(t *testing.T) {
	api := newFakePRComments()
	api.byPR[3] = []CommentRef{
		{ID: 10, Body: "user said hello"},
		{ID: 11, Body: "<!-- repository-detective-policy-summary-fake --> not ours"},
	}
	body := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET", ScanID: "s1"})
	r := UpsertPRPolicySummary(context.Background(), api, "o", "r", 3, body)
	if r.Action != "created" {
		t.Fatalf("expected create, got %+v", r)
	}
	list, _ := api.ListIssueComments(context.Background(), "o", "r", 3)
	if len(list) != 3 {
		t.Fatalf("got %d comments", len(list))
	}
	userOK, malformedOK := false, false
	for _, c := range list {
		if c.Body == "user said hello" {
			userOK = true
		}
		if strings.Contains(c.Body, "policy-summary-fake") {
			malformedOK = true
		}
	}
	if !userOK || !malformedOK {
		t.Fatal("user/malformed comments were modified")
	}
}

func TestUpsertDedupesLegacyRDSummaries(t *testing.T) {
	api := newFakePRComments()
	old1 := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET", ScanID: "old1"})
	old2 := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET", ScanID: "old2"})
	api.byPR[9] = []CommentRef{{ID: 1, Body: old1}, {ID: 2, Body: old2}}
	body := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "ACTION_REQUIRED", ScanID: "new"})
	r := UpsertPRPolicySummary(context.Background(), api, "o", "r", 9, body)
	if r.Action != "updated" || r.DuplicatesRemoved != 1 {
		t.Fatalf("got %+v", r)
	}
	list, _ := api.ListIssueComments(context.Background(), "o", "r", 9)
	if len(list) != 1 {
		t.Fatalf("want 1 leftover, got %d", len(list))
	}
}

func TestUpsertFailsClosedWhenListFails(t *testing.T) {
	api := newFakePRComments()
	api.listErr = errors.New("boom")
	body := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET"})
	r := UpsertPRPolicySummary(context.Background(), api, "o", "r", 1, body)
	if r.Action != "failed" || api.creates != 0 {
		t.Fatalf("must not create when list fails: %+v creates=%d", r, api.creates)
	}
}

func TestIdenticalRescanDoesNotDuplicate(t *testing.T) {
	api := newFakePRComments()
	body := RenderPRPolicySummary(PRPolicySummaryInput{Outcome: "POLICY_MET", ScanID: "s1", CommitSHA: "abc"})
	_ = UpsertPRPolicySummary(context.Background(), api, "o", "r", 5, body)
	_ = UpsertPRPolicySummary(context.Background(), api, "o", "r", 5, body)
	list, _ := api.ListIssueComments(context.Background(), "o", "r", 5)
	if len(list) != 1 || api.creates != 1 {
		t.Fatalf("duplicated: comments=%d creates=%d", len(list), api.creates)
	}
}
