package issues

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

func TestFindIssueByFingerprint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		issues := []gitea.Issue{{
			Number:  7,
			HTMLURL: "https://git.example.com/owner/repo/issues/7",
			Body:    "## Tracking\n\n- Repository Detective fingerprint: rd-matchme\n",
		}}
		_ = json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	match, err := FindIssueByFingerprint(context.Background(), &GiteaForge{Client: client}, "owner", "repo", "rd-matchme")
	if err != nil {
		t.Fatalf("FindIssueByFingerprint: %v", err)
	}
	if match == nil || match.IssueNumber != 7 {
		t.Fatalf("unexpected match: %+v", match)
	}
}

func TestCreateIssuesUpdatesExistingFingerprint(t *testing.T) {
	var comments int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			issues := []gitea.Issue{{
				Number:  9,
				HTMLURL: "https://git.example.com/owner/repo/issues/9",
				Body:    "## Tracking\n\n- Repository Detective fingerprint: rd-existing\n",
			}}
			_ = json.NewEncoder(w).Encode(issues)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			comments++
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusOK)
			_, _ = io.ReadAll(r.Body)
			w.Write([]byte("[]"))
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	manager := NewManager(client, nil, GetDefaultConfig(), logrus.New())

	issue := ai.CodeIssue{
		Title:       "Secret finding",
		Description: "test",
		Severity:    "high",
		Category:    "secret",
		Source:      "gitleaks",
		RuleID:      "generic-api-key",
		File:        "config/env.py",
		LineNumber:  10,
		Confidence:  0.95,
		CodeSnippet: "api_key=REDACTED",
	}

	EnrichIssue("owner/repo", &issue, "scan-xyz")
	issue.Fingerprint = "rd-existing"

	result, err := manager.CreateIssuesFromAnalysis(context.Background(), &IssueCreationRequest{
		Owner:      "owner",
		Repository: "repo",
		ScanID:     "scan-xyz",
		AnalysisResult: &ai.CodeAnalysisResult{
			Issues: []ai.CodeIssue{issue},
		},
	})
	if err != nil {
		t.Fatalf("CreateIssuesFromAnalysis: %v", err)
	}
	if result.IssuesCreated != 0 {
		t.Fatalf("expected no new issues, got %d", result.IssuesCreated)
	}
	if result.IssuesUpdated != 1 {
		t.Fatalf("expected 1 updated issue, got %d", result.IssuesUpdated)
	}
	if comments != 1 {
		t.Fatalf("expected one still-present comment, got %d", comments)
	}
}
