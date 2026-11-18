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

	"git.commsnet.org/commstech/repository-detective/store"
)

// Scan report evidence must precede optional AI interpretations. Settings
// must preserve the existing POST/CSRF contract and custom-rule authoring.
func TestScanAndSettingsWorkbenchContracts(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "workbench.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	}()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "qa", Name: "example", FullName: "qa/example"})
	if err != nil {
		t.Fatal(err)
	}
	scanID := "workbenchscan000001"
	summary, err := json.Marshal(map[string]any{"issues_found": 3, "files_analyzed": 42})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := s.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, StartedAt: now.Add(-time.Minute),
		FinishedAt: &now, SummaryJSON: summary,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddScannerResults(ctx, []store.ScannerResultRecord{
		{ScanID: scanID, ScannerName: "trivy", Status: "found", FindingsCount: 3},
		{ScanID: scanID, ScannerName: "gosec", Status: "failed", Error: "scanner exited"},
	}); err != nil {
		t.Fatal(err)
	}
	router, _ := testUI(t, s)
	scanResponse := httptest.NewRecorder()
	router.ServeHTTP(scanResponse, httptest.NewRequest(http.MethodGet, "/ui/scans/"+scanID, nil))
	if scanResponse.Code != http.StatusOK {
		t.Fatalf("scan detail returned HTTP %d", scanResponse.Code)
	}
	scanPage := scanResponse.Body.String()
	for _, expected := range []string{
		`id="rd-scan-coverage"`, `data-scanner-coverage`, `data-scanner-status="found"`,
		`data-scanner-status="failed"`, `data-scanner-error="true"`, `id="rd-scan-ai"`,
		`id="rd-scan-diagnostics"`, `Review scanner evidence`,
	} {
		if !strings.Contains(scanPage, expected) {
			t.Errorf("scan page missing %q", expected)
		}
	}
	if a, b := strings.Index(scanPage, `id="rd-scan-coverage"`), strings.Index(scanPage, `id="rd-scan-ai"`); a < 0 || b < 0 || a >= b {
		t.Errorf("scanner evidence must precede AI advice: %d, %d", a, b)
	}
	settingsPath := "/ui/repos/" + strconv.FormatInt(repo.ID, 10) + "/settings"
	settingsResponse := httptest.NewRecorder()
	router.ServeHTTP(settingsResponse, httptest.NewRequest(http.MethodGet, settingsPath, nil))
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("settings returned HTTP %d", settingsResponse.Code)
	}
	settingsPage := settingsResponse.Body.String()
	for _, expected := range []string{
		`id="rd-repo-profile"`, `id="rd-repo-rules"`, `id="rd-repo-alerts"`,
		`id="rd-repo-advanced"`, `name="scan_profile"`, `name="csrf_token"`,
		`data-rule-builder`, `Save repository settings`,
	} {
		if !strings.Contains(settingsPage, expected) {
			t.Errorf("settings missing %q", expected)
		}
	}
	if a, b, c := strings.Index(settingsPage, `id="rd-repo-profile"`), strings.Index(settingsPage, `id="rd-repo-rules"`), strings.Index(settingsPage, `id="rd-repo-alerts"`); !(a >= 0 && a < b && b < c) {
		t.Errorf("settings should show profile → custom rules → notifications")
	}
}
