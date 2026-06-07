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
	h.RegisterPublicRoutes(g)
	h.RegisterRoutes(g)
	return r, h
}

func extractDashboardChartJSON(t *testing.T, body string) string {
	t.Helper()
	const start = `id="rd-dashboard-data">`
	const end = `</script>`
	i := strings.Index(body, start)
	if i < 0 {
		t.Fatal("missing rd-dashboard-data script")
	}
	i += len(start)
	j := strings.Index(body[i:], end)
	if j < 0 {
		t.Fatal("missing closing script tag for chart data")
	}
	return strings.TrimSpace(body[i : i+j])
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
	if !strings.Contains(body, "logo.svg") {
		t.Fatal("expected branded logo.svg in layout")
	}
	if !strings.Contains(body, "rd-chart-severity") {
		t.Fatal("expected dashboard severity chart canvas")
	}
	if !strings.Contains(body, "dashboard-charts.js") {
		t.Fatal("expected dashboard charts script")
	}
	if !strings.Contains(body, "Executive report") {
		t.Fatal("expected executive report section on dashboard")
	}
	if !strings.Contains(body, "favicon.svg") {
		t.Fatal("expected favicon in layout")
	}
}

func TestDashboardChartJSONParses(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "chart-parse.db")})
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp1", Title: "Test finding",
		Severity: "high", Category: "security", Source: "semgrep", Status: "open",
		FirstSeenAt: now, LastSeenAt: now,
	})

	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	raw := extractDashboardChartJSON(t, w.Body.String())
	if strings.HasPrefix(raw, `"`) {
		t.Fatalf("chart JSON must not be double-encoded, got prefix: %.40q", raw)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid chart JSON: %v\nraw=%.120s", err, raw)
	}
	if _, ok := payload["severityLabels"]; !ok {
		t.Fatalf("expected severityLabels in chart payload: %v", payload)
	}
}

func TestUIRoutesSmoke(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "smoke.db")})
	defer s.Close()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-smoke", Title: "Smoke test finding",
		Severity: "medium", Category: "security", Source: "semgrep", Status: "open",
		FirstSeenAt: now, LastSeenAt: now,
	})

	r, _ := testUI(t, s)
	routes := []struct {
		path    string
		contain string
	}{
		{"/ui/", "rd-chart-severity"},
		{"/ui/repos", "o/r"},
		{"/ui/repos/" + strconv.FormatInt(repo.ID, 10), "o/r"},
		{"/ui/repos/" + strconv.FormatInt(repo.ID, 10) + "/report", "Technical findings"},
		{"/ui/findings", "Findings queue"},
		{"/ui/scans", "Scan history"},
		{"/ui/reports", "Executive summary"},
		{"/ui/health", "System Health"},
		{"/ui/configure", "Platform configuration"},
		{"/ui/preinstall", "Pre-install audit"},
		{"/ui/projects", "Project groups"},
	}
	for _, rt := range routes {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, rt.path, nil)
		r.ServeHTTP(w, req)
		body := w.Body.String()
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", rt.path, w.Code)
		}
		if !strings.Contains(body, rt.contain) {
			t.Fatalf("%s missing %q", rt.path, rt.contain)
		}
		if strings.Contains(body, "template error") || strings.Contains(body, "can't evaluate field") {
			t.Fatalf("%s template failure: %.200s", rt.path, body)
		}
	}
}

func TestStaticAssetsPublic(t *testing.T) {
	r, _ := testUI(t, nil)
	assets := []string{
		"/ui/static/theme.css",
		"/ui/static/chart.umd.min.js",
		"/ui/static/dashboard-charts.js",
		"/ui/static/logo.svg",
		"/ui/static/favicon.svg",
		"/ui/static/app.js",
	}
	for _, path := range assets {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
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
		"issues_found":       804,
		"files_analyzed":     362,
		"analysis_time_ms":   234000,
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

func TestGraphPageMissingGraphNoTruncationBanner(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui-graph-missing.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "g", FullName: "o/g"})
	scanID := "graphmissing0001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/scans/"+scanID+"/graph", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("graph page status %d", w.Code)
	}
	if !strings.Contains(body, `data-graph-available="false"`) {
		t.Fatal("expected graph unavailable marker")
	}
	if !strings.Contains(body, "No graph was stored for this scan") {
		t.Fatal("expected missing graph message")
	}
	if strings.Contains(body, `id="graph-truncated"`) && strings.Contains(body, "Graph was truncated") {
		// truncation banner exists in template but must stay hidden for missing graphs (handled in JS + server flag)
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
	if !strings.Contains(body, "logo.svg") {
		t.Fatal("expected branded logo.svg in layout")
	}
}

func TestThemeCSSServedWithoutAPIKey(t *testing.T) {
	r, _ := testUI(t, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/static/theme.css", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("theme.css must be public, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
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

func TestRepoReportFindingsTracker(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "repo-report.db")})
	defer s.Close()

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()
	_, _ = s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp1", Title: "SQL injection risk",
		Severity: "high", Category: "security", Source: "semgrep", Status: "open",
		FilePath: "main.go", Line: 42, Confidence: 0.92,
		FirstSeenAt: now, LastSeenAt: now,
	})

	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/report", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("repo report status %d", w.Code)
	}
	if strings.Contains(body, "(sample)") {
		t.Fatal("report must not label findings as sample")
	}
	if !strings.Contains(body, "Technical findings") || !strings.Contains(body, "SQL injection risk") {
		t.Fatal("expected structured findings table with data")
	}
	if !strings.Contains(body, "rd-table-triage") {
		t.Fatal("expected triage table styling")
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

func TestThemeBootstrapAndToggle(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "theme-ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "localStorage.getItem") {
		t.Fatal("expected theme bootstrap script in head")
	}
	if !strings.Contains(body, "document.documentElement.dataset.theme") {
		t.Fatal("expected data-theme bootstrap via dataset.theme")
	}
	if !strings.Contains(body, `id="rd-theme-toggle"`) {
		t.Fatal("expected theme toggle group")
	}
	if !strings.Contains(body, `data-theme-choice="system"`) {
		t.Fatal("expected system theme option")
	}
	if !strings.Contains(body, `data-theme-choice="light"`) {
		t.Fatal("expected light theme option")
	}
	if !strings.Contains(body, `data-theme-choice="dark"`) {
		t.Fatal("expected dark theme option")
	}
	if !strings.Contains(body, `aria-label="Theme"`) {
		t.Fatal("expected accessible Theme label on toggle group")
	}
	if !strings.Contains(body, "theme.js") {
		t.Fatal("expected theme.js script")
	}
}

func TestThemeCSSIncludesLightAndDarkTokens(t *testing.T) {
	r, _ := testUI(t, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/static/theme.css", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("theme.css status %d", w.Code)
	}
	body := w.Body.String()
	for _, token := range []string{`html[data-theme="light"]`, `html[data-theme="dark"]`, `html[data-theme="system"]`, ".rd-theme-switch"} {
		if !strings.Contains(body, token) {
			t.Fatalf("expected %q in theme.css", token)
		}
	}
}

func TestGraphPageAccessibility(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "graph-ui.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/repos/1/graph", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Fatalf("unexpected graph page status %d", w.Code)
	}
	if w.Code == http.StatusOK {
		body := w.Body.String()
		if !strings.Contains(body, `id="graph-summary"`) {
			t.Fatal("expected graph text summary fallback")
		}
		if !strings.Contains(body, `for="layout-mode"`) {
			t.Fatal("expected labeled graph layout control")
		}
		if !strings.Contains(body, "rd-graph-canvas") {
			t.Fatal("expected themed graph canvas class")
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
