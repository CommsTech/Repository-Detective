package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetIssueReturnsState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/issues/42") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Issue{Number: 42, State: "closed", HTMLURL: "http://git/o/r/issues/42", Body: "done"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	issue, err := client.GetIssue(context.Background(), "owner", "repo", 42)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Number != 42 || issue.State != "closed" {
		t.Fatalf("unexpected issue %+v", issue)
	}
}

func TestListIssuesUsesLabelsFilter(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]Issue{{Number: 1, Body: "test"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	issues, err := client.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{
		State:  "open",
		Labels: []string{"repository-detective"},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if gotQuery == "" || !strings.Contains(gotQuery, "labels=repository-detective") || !strings.Contains(gotQuery, "state=open") {
		t.Fatalf("unexpected query %q", gotQuery)
	}
}
