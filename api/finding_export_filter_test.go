package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestExportFindingsRespectsFiltersAndAllPages(t *testing.T) {
	s := testStore(t)
	repo := seedRepo(t, s)
	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 220; i++ {
		severity, source, status := "medium", "ruff", "open"
		if i < 212 {
			severity, source = "high", "trivy"
		}
		if i == 211 {
			status = "suppressed"
		}
		title := fmt.Sprintf("CVE-2026-TEST-%03d", i)
		if i == 0 {
			title = "=1+2"
		}
		if _, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID, Fingerprint: fmt.Sprintf("export-%d", i),
			Title: title, Severity: severity, Status: status, Source: source,
			FirstSeenAt: now, LastSeenAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	r := testRouter(t, s)
	request := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}
	// Export must ignore UI pagination options and return the whole *matching*
	// set: 211 open high/trivy rows spread over two DB pages.
	w := request("/api/v1/findings/export?format=json&focus=1&source=trivy&limit=1&offset=200&status=resolved")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Count    int `json:"count"`
		Findings []struct {
			Severity string `json:"severity"`
			Status   string `json:"status"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 211 || len(result.Findings) != 211 {
		t.Fatalf("export returned %d instead of 211", result.Count)
	}
	for _, f := range result.Findings {
		if f.Severity != "high" || f.Status != "open" {
			t.Fatalf("filter ignored: %+v", f)
		}
	}
	csv := request("/api/v1/findings/export?format=csv&q=%3D1%2B2")
	if csv.Code != http.StatusOK || !strings.Contains(csv.Body.String(), "'=1+2") {
		t.Fatalf("CSV escaping/search failed: %d %s", csv.Code, csv.Body.String())
	}
	if got := request("/api/v1/findings/export?format=json&status=suppressed"); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "CVE-2026-TEST-211") {
		t.Fatal("explicit suppressed-status export missing")
	}
	if got := request("/api/v1/findings/export?format=json&owner=mine"); got.Code != http.StatusBadRequest {
		t.Fatal("API owner filter must be rejected")
	}
	if got := request("/api/v1/findings/export?format=xml"); got.Code != http.StatusBadRequest {
		t.Fatal("unsupported export must be rejected")
	}
}
