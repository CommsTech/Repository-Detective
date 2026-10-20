package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

// Concurrent createOrUpdate for the same fingerprint must not open duplicates
// inside one Manager process (serialized lookup→create).
func TestCreateOrUpdateIssueSerializesInProcess(t *testing.T) {
	var creates atomic.Int32
	var mu sync.Mutex
	var issues []gitea.Issue

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode(issues)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
			creates.Add(1)
			n := len(issues) + 1
			iss := gitea.Issue{
				Number:  n,
				HTMLURL: fmt.Sprintf("https://git.example.com/owner/repo/issues/%d", n),
				Body:    "## Tracking\n\n- Repository Detective fingerprint: rd-race\n",
				Title:   "race",
			}
			issues = append(issues, iss)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(iss)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	manager := NewManager(client, nil, GetDefaultConfig(), logrus.New())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issue := ai.CodeIssue{
				Title: "Race finding", Description: "test", Severity: "high",
				Category: "secret", Source: "gitleaks", RuleID: "generic-api-key",
				File: "a.env", LineNumber: 1, Confidence: 0.95,
			}
			EnrichIssue("owner/repo", &issue, "scan-race")
			issue.Fingerprint = "rd-race"
			_, _ = manager.CreateIssuesFromAnalysis(context.Background(), &IssueCreationRequest{
				Owner: "owner", Repository: "repo", ScanID: "scan-race",
				AnalysisResult: &ai.CodeAnalysisResult{Issues: []ai.CodeIssue{issue}},
			})
		}()
	}
	wg.Wait()
	if got := creates.Load(); got != 1 {
		t.Fatalf("expected exactly 1 forge create, got %d", got)
	}
}
