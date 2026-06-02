package ui_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/bugbot/graph"
	"git.commsnet.org/commstech/bugbot/notify"
	"git.commsnet.org/commstech/bugbot/store"
	"git.commsnet.org/commstech/bugbot/ui"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func testUI(t *testing.T, s store.QueryStore) (*gin.Engine, *ui.Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h, err := ui.NewHandler(s, store.DefaultGlobalSettings(), "/ui", logrus.New(), nil, false, "test-secret")
	if err != nil {
		t.Fatalf("new ui handler: %v", err)
	}
	r := gin.New()
	g := r.Group("/ui")
	h.RegisterRoutes(g)
	return r, h
}

func TestDashboardRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Repository Detective") {
		t.Fatal("expected page title branding")
	}
	if !strings.Contains(body, "theme.css") {
		t.Fatal("expected branded theme stylesheet")
	}
}

func TestScanDetailRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "34a0c0d5698a5da1"
	summary, _ := json.Marshal(map[string]any{
		"issues_found":     804,
		"files_analyzed":   362,
		"analysis_time_ms": 234000,
		"effective_settings": map[string]any{"scan_profile": "standard"},
	})
	finished := time.Now().UTC()
	_, _ = s.CreateScan(ctx, store.Scan{
		ID:           scanID,
		RepositoryID: repo.ID,
		TriggerType:  store.TriggerManual,
		Ref:          "main",
		Status:       store.ScanStatusCompleted,
		StartedAt:    finished.Add(-4 * time.Minute),
		FinishedAt:   &finished,
		SummaryJSON:  summary,
	})
	_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{{
		ScanID: scanID, ScannerName: "trivy", Status: "found", FindingsCount: 3,
	}})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/scans/"+scanID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("scan detail status %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "804") {
		t.Fatal("expected issues count from summary")
	}
	if !strings.Contains(body, "completed") {
		t.Fatal("expected completed scan status")
	}
	if !strings.Contains(body, "o/r") {
		t.Fatal("expected repository name")
	}
	if strings.Contains(body, "can't evaluate field") {
		t.Fatal("template render error leaked into page")
	}
}

func TestReposPageRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	_, _ = s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "o/r") {
		t.Fatalf("repos page failed: %d %s", w.Code, w.Body.String())
	}
}

func TestFindingDetailEscapesHTML(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "bugbot-x", Title: "<script>alert(1)</script>",
		FirstSeenAt: now, LastSeenAt: now,
	})
	_ = s.AddFindingInstance(ctx, store.FindingInstance{
		FindingID: finding.ID, ScanID: "s1", EvidenceRedacted: "<b>safe</b>",
	})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/findings/1", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatal("HTML was not escaped in finding title")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("expected escaped script in output")
	}
}

func TestRepoSettingsRendersProfileSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/settings", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Scan profile") {
		t.Fatalf("settings page missing profile section: %d", w.Code)
	}
	if !strings.Contains(body, "Advanced settings") || !strings.Contains(body, "deterministic-first") {
		t.Fatal("expected profile UX elements")
	}
}

func TestRepoSettingsRendersHealthSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/settings", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Health checks") {
		t.Fatalf("settings page missing health section: %d", w.Code)
	}
}

func TestRepoSettingsRendersGraphSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/settings", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Repository map / code graph") {
		t.Fatalf("settings page missing graph section: %d", w.Code)
	}
}

func TestRepoSettingsRendersNotificationsSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	r, h := testUI(t, s)
	cfg := notify.DefaultConfig()
	cfg.Enabled = true
	h.SetNotificationGlobal(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/settings", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Notifications") {
		t.Fatalf("settings page missing notifications section: %d", w.Code)
	}
	if strings.Contains(body, "webhook_url") {
		t.Fatal("webhook URL field must not appear in settings HTML")
	}
}

func TestGraphPageUsesLocalCytoscape(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui-graph.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "g", FullName: "o/g"})
	scanID := "graphuitest00001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	g := graph.Graph{
		ScanID: scanID,
		Nodes:  []graph.Node{{ID: "n1", Type: "file", Label: "main.go"}},
		Edges:  []graph.Edge{},
	}
	raw, _ := json.Marshal(g)
	if err := s.SaveScanGraph(ctx, store.ScanGraphRecord{
		ScanID: scanID, RepositoryID: repo.ID, GraphJSON: raw, NodeCount: 1, GeneratedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("save graph: %v", err)
	}
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/scans/"+scanID+"/graph", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("graph page status %d", w.Code)
	}
	if strings.Contains(body, "jsdelivr") || strings.Contains(body, "cdn.jsdelivr") {
		t.Fatal("graph page must not reference CDN")
	}
	if !strings.Contains(body, "/ui/static/cytoscape.min.js") {
		t.Fatal("expected local cytoscape asset")
	}
	if !strings.Contains(body, "/ui/static/graph.js") {
		t.Fatal("expected local graph.js asset")
	}
	if !strings.Contains(body, "graph-legend") {
		t.Fatal("expected graph legend markup")
	}
}

func TestGraphStaticAssetServed(t *testing.T) {
	r, _ := testUI(t, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/static/cytoscape.min.js", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("static cytoscape status %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
}

func TestUIDisabledStoreReturns503(t *testing.T) {
	h, err := ui.NewHandler(nil, store.DefaultGlobalSettings(), "/ui", logrus.New(), nil, false, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	g := r.Group("/ui")
	h.RegisterRoutes(g)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestScansPageRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "scans-ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/scans", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Scan history") {
		t.Fatalf("scans page failed: %d %s", w.Code, body[:min(200, len(body))])
	}
	if !strings.Contains(body, "logo.png") {
		t.Fatal("expected branded logo.png in layout")
	}
}

func TestHealthPageRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "health-ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "System Health") {
		t.Fatalf("health page failed: %d", w.Code)
	}
}

func TestReportsPageRenders(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "reports-ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/reports", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Executive summary") {
		t.Fatalf("reports page failed: %d", w.Code)
	}
	if !strings.Contains(body, "app.js") {
		t.Fatal("expected app.js in layout")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
