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
