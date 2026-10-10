package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func testStore(t *testing.T) store.QueryStore {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "api.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func seedRepo(t *testing.T, s store.QueryStore) store.Repository {
	t.Helper()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "acme", Name: "demo", FullName: "acme/demo", ConnectedRepo: true})
	if err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	return repo
}

func testRouter(t *testing.T, s store.QueryStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewHandler(s, store.DefaultGlobalSettings(), logrus.New())
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)
	return r
}

func TestListRepositories(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/repos", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var resp map[string][]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["repositories"]) != 1 {
		t.Fatalf("expected 1 repo, got %v", resp["repositories"])
	}
}

func TestGetRepoSettings(t *testing.T) {
	s := testStore(t)
	repo := seedRepo(t, s)
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/repos/1/settings", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("notice")) {
		t.Fatal("expected settings notice in response")
	}
	_ = repo
}

func TestUpdateRepoSettingsValid(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"policy_level":"gate_pr","workspace_mode":"archive","analysis_depth":2}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestUpdateRepoSettingsInvalid(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"policy_level":"invalid_policy"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateRepoSettingsHealthFields(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"enable_health_checks":false,"enable_reliability_checks":false,"health_large_file_lines":500}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/repos/1/settings", nil)
	r.ServeHTTP(w2, req2)
	if !bytes.Contains(w2.Body.Bytes(), []byte(`"enable_health_checks":false`)) {
		t.Fatalf("expected health fields in effective settings: %s", w2.Body.String())
	}
}

func TestUpdateRepoSettingsInvalidHealthThreshold(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"health_large_file_lines":999999}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid threshold, got %d", w.Code)
	}
}

func TestDatabaseDisabledResponse(t *testing.T) {
	r := testRouter(t, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/repos", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestNoSecretsInAPIResponse(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	for _, secret := range []string{"gitea_token", "ai_api_key", "database_dsn", "super-secret"} {
		if bytes.Contains([]byte(body), []byte(secret)) {
			t.Fatalf("response leaked %s", secret)
		}
	}
}

func TestScanDetailAndScannerResults(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	scanID := "abc123scan0001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	_ = s.AddScannerResults(ctx, []store.ScannerResultRecord{{ScanID: scanID, ScannerName: "trivy", Status: "clean"}})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scans/"+scanID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("scan detail status %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/scans/"+scanID+"/scanner-results", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("scanner results status %d", w2.Code)
	}
}

func TestFindingsAndLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	now := time.Now().UTC()
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "rd-api-test", Title: "Test", Severity: "high",
		FirstSeenAt: now, LastSeenAt: now,
	})
	fid := finding.ID
	_ = s.AddLifecycleEvent(ctx, store.LifecycleEvent{FindingID: &fid, EventType: "open", Message: "seen"})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/findings?severity=high", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("findings list %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/findings/1/lifecycle", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("lifecycle %d", w2.Code)
	}
}

func TestDashboardSummary(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard %d", w.Code)
	}
}

func TestUpdateRepoSettingsGraphFields(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"enable_code_graph":false,"graph_max_nodes":2000,"graph_max_edges":5000}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/repos/1/settings", nil)
	r.ServeHTTP(w2, req2)
	if !bytes.Contains(w2.Body.Bytes(), []byte(`"enable_code_graph":false`)) {
		t.Fatalf("expected graph fields in effective settings: %s", w2.Body.String())
	}
}

func TestUpdateRepoSettingsInvalidGraphThreshold(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"graph_max_nodes":50}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid graph threshold, got %d", w.Code)
	}
}

func TestExportScanGraph(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	scanID := "exportgraph0001"
	_, _ = s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual, Status: store.ScanStatusCompleted})
	payload := []byte(`{"nodes":[{"id":"n1","type":"file","label":"a.go"}],"edges":[]}`)
	_ = s.SaveScanGraph(ctx, store.ScanGraphRecord{
		ScanID: scanID, RepositoryID: repo.ID, GraphJSON: payload, NodeCount: 1, GeneratedAt: time.Now().UTC(),
	})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scans/"+scanID+"/graph/export", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("export status %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"nodes"`)) {
		t.Fatal("expected graph JSON body")
	}
	if disp := w.Header().Get("Content-Disposition"); disp == "" || !bytes.Contains([]byte(disp), []byte("attachment")) {
		t.Fatalf("expected attachment disposition, got %q", disp)
	}
}

func TestDisableRepoScanningAPI(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	repo := seedRepo(t, s)
	on := true
	_ = s.SaveRepoSettings(ctx, store.RepoSettings{RepositoryID: repo.ID, Enabled: &on})
	r := testRouter(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/repos/1/disable-scanning", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	settings, _ := s.GetRepoSettings(ctx, repo.ID)
	g := store.DefaultGlobalSettings()
	g.IssuePolicy = store.IssuePolicyOff
	if store.ResolveEffectiveSettings(g, settings).Enabled {
		t.Fatal("expected disabled")
	}
}
