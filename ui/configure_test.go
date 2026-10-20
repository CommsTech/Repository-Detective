package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func testUIWithReadiness(t *testing.T, s store.QueryStore) (*gin.Engine, *ui.Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h, err := ui.NewHandler(s, store.DefaultGlobalSettings(), "/ui", logrus.New(), nil, false, "test-secret")
	if err != nil {
		t.Fatalf("new ui handler: %v", err)
	}
	h.SetPlatformContext(ui.PlatformContext{RemediationPRMaxFiles: 3, RemediationPRMaxDiffLines: 100})
	h.SetReadinessFn(func() operator.Readiness {
		return operator.Readiness{
			Features: operator.FeatureFlags{
				DatabaseEnabled: true, RemediationPREnabled: false, RemediationPlannerEnabled: true,
			},
		}
	})
	h.SetNotificationGlobal(notify.DefaultConfig())
	r := gin.New()
	g := r.Group("/ui")
	h.RegisterPublicRoutes(g)
	h.RegisterRoutes(g)
	return r, h
}

func TestConfigurePageRemediationPRSection(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "cfg.db")})
	defer s.Close()
	r, _ := testUIWithReadiness(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/configure", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"id=\"remediation-pr\"", "remediation_pr_enabled", "Beta default", "gitea_token",
		"id=\"edit-settings\"", "Save settings", "scan_profile",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in configure page", want)
		}
	}
}

func TestConfigureSavePersistsSettings(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "cfg-save.db")})
	defer s.Close()
	r, h := testUIWithReadiness(t, s)
	h.SetPlatformSettingsApplier(func(settings store.PlatformSettings) error { return nil })

	csrf := security.CSRFToken("test-secret", "configure-test-key")
	form := url.Values{}
	form.Set("csrf_token", csrf)
	form.Set("scan_profile", store.ScanProfileStandard)
	form.Set("severity_gate", "high")
	form.Set("confidence_gate", "0.85")
	form.Set("analysis_depth", "2")
	form.Set("auto_create_issues", "false")
	form.Set("issue_policy", "off")
	form.Set("remediation_policy", "suggest")
	form.Set("enable_gitleaks", "true")
	form.Set("enable_trivy", "true")
	form.Set("scheduler_enabled", "true")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ui/configure", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Repository-Detective-API-Key", "configure-test-key")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	saved, err := s.GetPlatformSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if saved.ScanProfile != store.ScanProfileStandard {
		t.Fatalf("saved profile=%q", saved.ScanProfile)
	}
	if saved.SchedulerEnabled == nil || !*saved.SchedulerEnabled {
		t.Fatal("expected scheduler enabled saved")
	}
}

func TestHealthCapabilityConfigureLinks(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "health.db")})
	defer s.Close()
	r, h := testUIWithReadiness(t, s)
	h.SetReadinessFn(func() operator.Readiness {
		return operator.Readiness{Features: operator.FeatureFlags{RemediationPREnabled: false, RemediationPlannerEnabled: true}}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/health", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "/ui/configure#remediation-pr") {
		t.Fatalf("health page missing remediation configure link: %.200s", body)
	}
}

func TestPreinstallDisabledNot404(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "pre.db")})
	defer s.Close()
	r, _ := testUI(t, s)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/preinstall", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Pre-install audit disabled") {
		t.Fatal("expected disabled banner")
	}
}
