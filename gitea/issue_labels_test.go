package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAddIssueLabelsUsesWrappedBody(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/org/repo/issues/1/labels" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[]`))
		case http.MethodPost:
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			received = string(buf)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1080,"name":"security","color":"d73a4a"}]`))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
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

func TestAddIssueLabelsSkipsAlreadyPresent(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/org/repo/issues/1/labels" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":1,"name":"repository-detective/open"}]`))
		case http.MethodPost:
			posts++
			t.Fatal("should not POST when label already present")
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	_, err := client.AddIssueLabels(context.Background(), "org", "repo", 1, []any{"repository-detective/open"})
	if err != nil {
		t.Fatalf("AddIssueLabels: %v", err)
	}
	if posts != 0 {
		t.Fatalf("expected no POST, got %d", posts)
	}
}

// TestAddIssueLabelsSilentNoOpIsError reproduces the live Beta 5 defect: Gitea
// returns HTTP 200 for the attach call but never applies the unknown label name
// (e.g. when the lifecycle label does not exist on the repository). The client
// must ensure the label exists, retry, and only report success when the label is
// actually observed on the issue.
func TestAddIssueLabelsSilentNoOpIsError(t *testing.T) {
	var repoLabels atomic.Value // []Label
	repoLabels.Store([]Label{})
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case strings.HasSuffix(r.URL.Path, "/issues/1/labels"):
			switch r.Method {
			case http.MethodGet:
				writeJSON(t, w, repoLabels.Load().([]Label))
			case http.MethodPost:
				// Silently drop the unknown name but return 200 with empty body,
				// exactly the live behavior that caused reconcile to "succeed"
				// while Gitea labels never changed.
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected method %s on %s", r.Method, r.URL.Path)
			}
		case r.URL.Path == "/api/v1/repos/org/repo/labels":
			switch r.Method {
			case http.MethodGet:
				writeJSON(t, w, repoLabels.Load().([]Label))
			case http.MethodPost:
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				name, _ := body["name"].(string)
				created := Label{ID: 9000 + int64(len(name)), Name: name, Color: "6f42c1", Exclusive: true}
				repoLabels.Store(append(repoLabels.Load().([]Label), created))
				writeJSON(t, w, created)
			default:
				t.Fatalf("unexpected method %s on %s", r.Method, r.URL.Path)
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	labels, err := client.AddIssueLabels(context.Background(), "org", "repo", 1, []any{"repository-detective/duplicate"})
	if err != nil {
		t.Fatalf("AddIssueLabels: %v", err)
	}
	found := false
	for _, l := range labels {
		if l.Name == "repository-detective/duplicate" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected observed label on issue after ensure+retry, got %#v", labels)
	}
}

// TestAddIssueLabelsUnresolvableNameErrors verifies fail-closed behavior when
// Gitea drops the label AND it cannot be ensured (e.g. permission denied on the
// repository label endpoints). The write must return an error, never success.
func TestAddIssueLabelsUnresolvableNameErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/issues/1/labels"):
			if r.Method == http.MethodGet {
				writeJSON(t, w, []Label{})
				return
			}
			if r.Method == http.MethodPost {
				_, _ = w.Write([]byte(`[]`)) // 200 silent drop
				return
			}
			t.Fatalf("unexpected method %s", r.Method)
		case strings.HasSuffix(r.URL.Path, "/repo/labels"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"forbidden"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", logrus.New())
	_, err := client.AddIssueLabels(context.Background(), "org", "repo", 1, []any{"repository-detective/duplicate"})
	if err == nil {
		t.Fatal("expected error when label cannot be applied or ensured")
	}
	if !strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "not applied") {
		t.Fatalf("error should name the failure mode, got: %v", err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}
