package ui_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
)

func TestThemeJSUsesRDThemeStorageKey(t *testing.T) {
	body := readStaticFile(t, "static/theme.js")
	for _, token := range []string{
		`STORAGE_KEY = "rd-theme"`,
		"readPreference",
		"writePreference",
		"dataset.theme",
		"rd-theme-change",
		"window.RDTheme",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("expected theme.js to contain %q", token)
		}
	}
	if strings.Contains(body, `localStorage.setItem(STORAGE_KEY, "system")`) {
		t.Fatal("theme.js must not force system preference on page load")
	}
}

func TestThemeJSInitializesFromStoredPreference(t *testing.T) {
	body := readStaticFile(t, "static/theme.js")
	if !strings.Contains(body, "applyTheme(readPreference()") {
		t.Fatal("expected theme.js to apply stored preference on init")
	}
}

func TestGraphJSListensForThemeChange(t *testing.T) {
	body := readStaticFile(t, "static/graph.js")
	for _, token := range []string{"rd-theme-change", "refreshGraphStyles"} {
		if !strings.Contains(body, token) {
			t.Fatalf("expected graph.js to contain %q", token)
		}
	}
}

func readStaticFile(t *testing.T, name string) string {
	t.Helper()
	data, err := fs.ReadFile(ui.StaticFS(), name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func renderDashboardHTML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "dash-theme.db")})
	defer s.Close()
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard status %d", w.Code)
	}
	return w.Body.String()
}

func renderPageHTML(t *testing.T, path string) string {
	t.Helper()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "page-theme.db")})
	defer s.Close()
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("%s status %d", path, w.Code)
	}
	return w.Body.String()
}

func TestThemeBootstrapBeforeCSS(t *testing.T) {
	body := renderDashboardHTML(t)
	bootstrap := strings.Index(body, `document.documentElement.dataset.theme`)
	css := strings.Index(body, "theme.css")
	if bootstrap < 0 {
		t.Fatal("expected theme bootstrap script in layout head")
	}
	if css < 0 {
		t.Fatal("expected theme.css link in layout head")
	}
	if bootstrap > css {
		t.Fatal("theme bootstrap must appear before theme.css in head")
	}
	if !strings.Contains(body, `localStorage.getItem(key)`) {
		t.Fatal("expected bootstrap to read rd-theme from localStorage")
	}
}

func TestLayoutDoesNotHardcodeSystemTogglePressed(t *testing.T) {
	body := renderDashboardHTML(t)
	if strings.Contains(body, `data-theme-choice="system" aria-pressed="true"`) {
		t.Fatal("layout must not hardcode System as the active theme toggle")
	}
}

func TestFindingsPageIncludesSharedThemeAssets(t *testing.T) {
	body := renderPageHTML(t, "/ui/findings")
	assertSharedThemeAssets(t, body)
}

func TestScansPageIncludesSharedThemeAssets(t *testing.T) {
	body := renderPageHTML(t, "/ui/scans")
	assertSharedThemeAssets(t, body)
}

func TestGraphPageIncludesSharedThemeAssets(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "graph-theme.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{
		Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true,
	})
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/graph", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graph page status %d", w.Code)
	}
	assertSharedThemeAssets(t, w.Body.String())
	if !strings.Contains(w.Body.String(), "graph.js") {
		t.Fatal("expected graph.js on graph page")
	}
}

func assertSharedThemeAssets(t *testing.T, body string) {
	t.Helper()
	for _, token := range []string{"theme.css", "theme.js", `id="rd-theme-toggle"`, "dataset.theme"} {
		if !strings.Contains(body, token) {
			t.Fatalf("expected shared theme asset %q in page", token)
		}
	}
}
