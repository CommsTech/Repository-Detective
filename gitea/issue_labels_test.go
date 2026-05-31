package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAddIssueLabelsUsesWrappedBody(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/org/repo/issues/1/labels" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		received = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1080,"name":"security","color":"d73a4a"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	labels, err := client.AddIssueLabels(context.Background(), "org", "repo", 1, []any{int64(1080), "security"})
	if err != nil {
		t.Fatalf("AddIssueLabels: %v", err)
	}
	if len(labels) != 1 || labels[0].Name != "security" {
		t.Fatalf("unexpected labels: %#v", labels)
	}

	var body map[string][]any
	if err := json.Unmarshal([]byte(received), &body); err != nil {
		t.Fatalf("invalid json body: %s", received)
	}
	if len(body["labels"]) != 2 {
		t.Fatalf("expected labels array in body, got %s", received)
	}
}

func TestAddIssueLabelsEmptyResponseFallsBackToGet(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":9,"name":"bugbot"}]`))
			return
		}
		t.Fatalf("unexpected method %s", r.Method)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	labels, err := client.AddIssueLabels(context.Background(), "org", "repo", 1, []any{"bugbot"})
	if err != nil {
		t.Fatalf("AddIssueLabels: %v", err)
	}
	if len(labels) != 1 || labels[0].Name != "bugbot" {
		t.Fatalf("unexpected labels: %#v", labels)
	}
	if calls != 2 {
		t.Fatalf("expected POST then GET fallback, got %d calls", calls)
	}
}
