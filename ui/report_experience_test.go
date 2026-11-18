package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFleetReportSectionsAndEvidenceLabels(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "fleet-experience.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/reports", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, expected := range []string{
		`id="fleet-overview"`, `id="fleet-decision"`, `id="fleet-metrics"`,
		`id="fleet-repositories"`, `id="fleet-scans"`,
		`Snapshot `, `UTC`, `Not an exploitability ranking`,
		`data-rd-quicknav-open`, `id="rd-quicknav"`, `icons.svg`,
		`Insufficient scan evidence`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("fleet report missing %q", expected)
		}
	}
}

func TestRepositoryAuditReportsScopeAndNoEvidence(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "repo-experience.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	repo, err := s.UpsertRepository(context.Background(), store.Repository{Owner: "maintainer", Name: "service", FullName: "maintainer/service"})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := testUI(t, s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/repos/"+strconv.FormatInt(repo.ID, 10)+"/report", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, expected := range []string{
		`id="repo-overview"`, `id="repo-decision"`, `id="repo-metrics"`,
		`id="repo-coverage"`, `id="repo-inventory"`, `id="repo-findings"`,
		`id="repo-evidence"`, `id="repo-scan-outcomes"`, "enabled in policy", "only this page",
		"No scan evidence", "Technical findings", "Snapshot ",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("repository report missing %q", expected)
		}
	}
	if strings.Contains(body, "rd-report-chart-data") || strings.Contains(body, "repo-report-charts.js") {
		t.Error("repository report should use the text-first chart-free presentation")
	}
	if strings.Contains(body, "Do not proceed — release blocked") {
		t.Error("missing scan evidence should not be conflated with a verified blocking vulnerability")
	}
}
