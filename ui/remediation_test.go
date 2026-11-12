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

func TestFindingDetailShowsRemediationSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui-rem.db")})
	defer s.Close()
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	finding, _ := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "fp", Title: "Finding", Severity: "high", Confidence: 0.9})
	r, h := testUI(t, s)
	h.SetRemediationBackend(true, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/findings/"+strconv.FormatInt(finding.ID, 10), nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "Remediation plan") {
		t.Fatalf("expected remediation section: %d", w.Code)
	}
	if !strings.Contains(body, "Planning only") {
		t.Fatal("expected planning-only warning")
	}
	if !strings.Contains(body, "Generate / regenerate plan") {
		t.Fatal("expected generate button")
	}
}
