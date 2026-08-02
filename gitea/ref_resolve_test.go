package gitea_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

func TestResolveRefUsesDefaultBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/o/r":
			_ = json.NewEncoder(w).Encode(map[string]any{"default_branch": "develop"})
		case r.URL.Path == "/api/v1/repos/o/r/branches":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "develop"}, {"name": "feature"}})
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/o/r/git/refs/heads/"):
			ref := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/o/r/git/refs/heads/")
			if ref == "develop" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"ref": "refs/heads/develop"}})
				return
			}
			http.NotFound(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/o/r/contents"):
			if r.URL.Query().Get("ref") == "main" {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "README.md", "path": "README.md", "type": "file"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	got, err := client.ResolveRef(context.Background(), "o", "r", "main")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if got != "develop" {
		t.Fatalf("ref = %q, want develop", got)
	}
}

func TestResolveRefEmptyRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/empty" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"default_branch": "main",
			"empty":          true,
		})
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	got, err := client.ResolveRef(context.Background(), "o", "empty", "")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if got != "main" {
		t.Fatalf("ref = %q, want main", got)
	}
}

func TestResolveRefForgeUnavailableNotInvalidRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/o/r":
			_ = json.NewEncoder(w).Encode(map[string]any{"default_branch": "main", "empty": false})
		default:
			http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
		}
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	_, err := client.ResolveRef(context.Background(), "o", "r", "main")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unable to verify refs") {
		t.Fatalf("want forge-unavailable style error, got %v", err)
	}
	if strings.Contains(err.Error(), "no valid ref found") {
		t.Fatalf("must not misclassify outage as missing ref: %v", err)
	}
}
