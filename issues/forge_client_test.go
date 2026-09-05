package issues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/github"
	"github.com/sirupsen/logrus"
)

func TestGitHubForgeCreateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/labels"):
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode(github.Issue{Number: 5, HTMLURL: "https://github.com/o/r/issues/5"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	forge := &GitHubForge{Client: github.NewClient(server.URL, "token", logrus.New())}
	issue, err := forge.CreateIssue(context.Background(), "o", "r", "title", "body", []string{"repository-detective"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if issue.Number != 5 {
		t.Fatalf("unexpected: %+v", issue)
	}
}
