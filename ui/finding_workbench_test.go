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

// The workbench must preserve existing review actions and explicit evidence
// boundaries while making the explanation and plan navigable without scripting.
func TestFindingWorkbenchRendersReviewableFlow(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "finding-workbench.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	}()
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "ops", Name: "long-project", FullName: "ops/long-project"})
	if err != nil {
		t.Fatal(err)
	}
	finding, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "finding-workbench", Title: strings.Repeat("a-long-path-", 15), Severity: "high", Confidence: 0.92, Category: "security", Source: "gosec", Status: "open"})
	if err != nil {
		t.Fatal(err)
	}
	router, handler := testUI(t, s)
	handler.SetRemediationBackend(true, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ui/findings/"+strconv.FormatInt(finding.ID, 10), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("finding workbench status: %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		`class="rd-card rd-finding-brief rd-case-page"`,
		`id="rd-finding-headline"`,
		`aria-label="Detection assessment"`,
		`<meter min="0" max="100" value="92"`,
		`href="#rd-remediation"`,
		`id="rd-remediation"`,
		`id="rd-evidence"`,
		`class="rd-card rd-evidence-expander rd-case-evidence"`,
		`remediation/generate`,
		`name="csrf_token"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected element or safety boundary %q", expected)
		}
	}
	brief := strings.Index(body, `class="rd-card rd-finding-brief rd-case-page"`)
	plan := strings.Index(body, `id="rd-remediation"`)
	evidence := strings.Index(body, `id="rd-evidence"`)
	if brief < 0 || plan <= brief || evidence <= plan {
		t.Fatalf("wrong investigation → plan → evidence order: %d %d %d", brief, plan, evidence)
	}
}
