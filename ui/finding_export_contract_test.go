package ui_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingsExportDownloadHeadersAndClientHooks(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "exports.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "docs", Name: "app", FullName: "docs/app"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "export-demo", Title: "Example", Status: "open", Severity: "medium", Source: "static", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := testUI(t, s)
	page := httptest.NewRecorder()
	r.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/ui/findings", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("findings page: %d %s", page.Code, page.Body.String())
	}
	for _, token := range []string{"data-findings-export", "data-finding-export-format=\"json\"", "data-finding-export-status"} {
		if !strings.Contains(page.Body.String(), token) {
			t.Fatalf("missing export hook %s", token)
		}
	}
	for _, format := range []string{"csv", "json"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/findings/export?format="+format, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s export: %d %s", format, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("Content-Disposition"), "."+format+"\"") {
			t.Fatalf("bad attachment disposition for %s: %s", format, w.Header().Get("Content-Disposition"))
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s response should not be cached", format)
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s response must set nosniff", format)
		}
		if format == "json" {
			var payload struct {
				Count int `json:"count"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil || payload.Count != 1 {
				t.Fatalf("bad JSON export: %+v %v", payload, err)
			}
		}
	}
}

func TestFindingAssignRequiresAuthenticatedLocalAccount(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "assign-ui.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/ui/findings/1/assign", strings.NewReader("assignee_user_id=1")))
	if w.Code != http.StatusForbidden {
		t.Fatalf("API-key mode must not accept assignment: %d", w.Code)
	}
}
