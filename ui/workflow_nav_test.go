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

func TestOperatorWorkflowOrderAndMobileNavigation(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "workflow.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	router, _ := testUI(t, s)
	check := func(path string) string {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s %d", path, w.Code)
		}
		return w.Body.String()
	}
	dashboard := check("/ui/")
	if !strings.Contains(dashboard, `data-mobile-nav-toggle`) || !strings.Contains(dashboard, `id="rd-primary-nav"`) {
		t.Fatal("accessible mobile navigation missing")
	}
	// The dashboard redesign replaced the first-run checklist with the
	// "Start here" onboarding block; assert the operator path, not old copy.
	if !strings.Contains(dashboard, `rd-dash-onboarding`) || !strings.Contains(dashboard, "Set up your first trustworthy scan") {
		t.Fatal("zero-repository dashboard needs useful onboarding")
	}
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "ops", Name: "r", FullName: "ops/r"})
	if err != nil {
		t.Fatal(err)
	}
	finding, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "test-finding", Title: "Dependency alert", Severity: "high", Status: "open", Category: "dependency", Source: "trivy"})
	if err != nil {
		t.Fatal(err)
	}
	detail := check("/ui/findings/" + strconv.FormatInt(finding.ID, 10))
	brief := strings.Index(detail, `class="rd-card rd-finding-brief rd-case-page"`)
	plan := strings.Index(detail, `id="rd-remediation"`)
	evidence := strings.Index(detail, `id="rd-evidence"`)
	if brief < 0 || plan <= brief || evidence <= plan {
		t.Fatalf("finding review order should be brief, plan, evidence; offsets: %d, %d, %d", brief, plan, evidence)
	}
}
