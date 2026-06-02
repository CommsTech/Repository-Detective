package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestListIssuesSkipsPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issues := []Issue{
			{Number: 1, HTMLURL: "https://github.com/o/r/issues/1", Body: "issue"},
			{Number: 2, HTMLURL: "https://github.com/o/r/pull/2", Body: "pr", PullRequest: &struct {
				URL string `json:"url"`
			}{URL: "https://api.github.com/repos/o/r/pulls/2"}},
		}
		_ = json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	list, err := client.ListIssues(context.Background(), "o", "r", ListIssuesOptions{State: "open"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(list) != 1 || list[0].Number != 1 {
		t.Fatalf("expected only issue #1, got %+v", list)
	}
}

func TestCreateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/issues") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Issue{Number: 42, HTMLURL: "https://github.com/o/r/issues/42"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	issue, err := client.CreateIssue(context.Background(), "o", "r", &CreateIssueRequest{
		Title:  "test",
		Body:   "body",
		Labels: []string{"repository-detective"},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if issue.Number != 42 {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}
