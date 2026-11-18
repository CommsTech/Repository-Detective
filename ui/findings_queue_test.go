package ui_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingsQueuePaginationAndCompleteExport(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "queue.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < 211; i++ {
		title := fmt.Sprintf("Finding %03d", i)
		if i == 210 {
			title = "=HYPERLINK(\"https://malicious.invalid\")"
		}
		_, err = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: fmt.Sprintf("fp-%03d", i), Title: title, Severity: "medium", Status: "open", Category: "quality", Source: "ruff", FirstSeenAt: now.Add(time.Duration(i) * time.Second), LastSeenAt: now.Add(time.Duration(i) * time.Second)})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "old-critical", Title: "Old critical CVE", Severity: "critical", Status: "open", Category: "dependency", Source: "trivy", FirstSeenAt: now.Add(-48 * time.Hour), LastSeenAt: now.Add(-48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := testUI(t, s)
	request := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status %d: %s", path, w.Code, w.Body.String())
		}
		return w
	}
	focus := request("/ui/findings?focus=1&status=resolved") // focus must override stale status query
	if !strings.Contains(focus.Body.String(), "Old critical CVE") || strings.Contains(focus.Body.String(), "Finding 210") {
		t.Fatal("focus queue must show old critical and not medium noise")
	}
	page1 := request("/ui/findings")
	if !strings.Contains(page1.Body.String(), "Showing 1–100 of 212") || !strings.Contains(page1.Body.String(), "page=2") {
		t.Fatal("missing count or next-page link")
	}
	page2 := request("/ui/findings?page=2")
	if !strings.Contains(page2.Body.String(), "Showing 101–200 of 212") {
		t.Fatal("missing page 2 range")
	}
	page3 := request("/ui/findings?page=3")
	if !strings.Contains(page3.Body.String(), "Showing 201–212 of 212") {
		t.Fatal("missing page 3 range")
	}
	csv := request("/ui/findings/export?format=csv")
	lines := strings.Split(strings.TrimSpace(csv.Body.String()), "\n")
	if len(lines) != 213 {
		t.Fatalf("csv rows=%d, want 213 including header", len(lines))
	}
	if !strings.Contains(csv.Body.String(), "'=HYPERLINK") {
		t.Fatal("formula-like CSV cell not neutralized")
	}
	j := request("/ui/findings/export?format=json")
	var exported struct {
		Count    int                     `json:"count"`
		Findings []store.FindingListItem `json:"findings"`
	}
	if err := json.Unmarshal(j.Body.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Count != 212 || len(exported.Findings) != 212 {
		t.Fatalf("json export truncated: %+v", exported.Count)
	}
	if exported.Findings[0].Severity != "critical" {
		t.Fatal("json export must sort critical before medium")
	}
}

func TestFindingsExportRejectsUnknownFormat(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "export.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/findings/export?format=xml", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unexpected format status=%d", w.Code)
	}
}

func TestFindingsSearchUXAndSuppressedStatus(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "search-ui.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "ops", Name: "repo", FullName: "ops/repo"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, f := range []store.Finding{
		{RepositoryID: repo.ID, Fingerprint: "fp-critical", Title: "CVE-2026-4455 urgent", Severity: "critical", Status: "open", Source: "trivy", Category: "dependency", FirstSeenAt: now, LastSeenAt: now},
		{RepositoryID: repo.ID, Fingerprint: "fp-muted", Title: "Noise finding", Severity: "low", Status: "suppressed", Source: "ruff", Category: "quality", FirstSeenAt: now, LastSeenAt: now},
	} {
		if _, err := s.UpsertFinding(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	router, _ := testUI(t, s)
	get := func(path string) string {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	focus := get("/ui/findings?focus=1&q=CVE-2026-4455")
	if !strings.Contains(focus, "CVE-2026-4455 urgent") || strings.Contains(focus, "Noise finding") {
		t.Fatal("focus search not applied before result filtering")
	}
	if !strings.Contains(focus, "Search the findings database") || !strings.Contains(focus, "name=\"q\"") {
		t.Fatal("search control not present")
	}
	searched := get("/ui/findings?q=Noise")
	if strings.Contains(searched, "Noise finding</strong>") {
		t.Fatal("suppressed should be hidden by default")
	}
	suppressed := get("/ui/findings?status=suppressed")
	if !strings.Contains(suppressed, "Noise finding</strong>") {
		t.Fatal("suppressed status must not require second checkbox")
	}
	suppressedOnly := get("/ui/findings?status=open&only_suppressed=1")
	if !strings.Contains(suppressedOnly, "Noise finding</strong>") {
		t.Fatal("only_suppressed must override conflicting status")
	}
	empty := get("/ui/findings?q=willnotmatch")
	if !strings.Contains(empty, "No matching findings") || !strings.Contains(empty, "Check scan coverage") {
		t.Fatal("no-action empty state")
	}
	exported := get("/ui/findings/export?format=json&q=CVE-2026-4455")
	if !strings.Contains(exported, "CVE-2026-4455 urgent") || strings.Contains(exported, "Noise finding") {
		t.Fatal("export should respect search")
	}
}
