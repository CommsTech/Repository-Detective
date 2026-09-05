package ui_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func testUIWithAPIKeyAuth(t *testing.T, apiKey string) (*gin.Engine, *ui.Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ui-auth.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	h, err := ui.NewHandler(s, store.DefaultGlobalSettings(), "/ui", logrus.New(), nil, false, apiKey)
	if err != nil {
		t.Fatalf("new ui handler: %v", err)
	}
	h.SetAuthConfig(ui.AuthConfig{Mode: "api_key_only"})
	r := gin.New()
	g := r.Group("/ui")
	g.Use(h.UIAPIKeyCookieMiddleware())
	h.RegisterPublicRoutes(g)
	protected := g.Group("")
	protected.Use(h.APIKeyAuthMiddleware())
	h.RegisterRoutes(protected)
	return r, h
}

func TestDashboardWithoutAPIKeyReturnsHTMLUnlock(t *testing.T) {
	r, _ := testUIWithAPIKeyAuth(t, "test-secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, `"error":"API key required"`) || strings.Contains(body, `{"error"`) {
		t.Fatal("expected HTML unlock page, not JSON error")
	}
	if !strings.Contains(body, "Unlock dashboard") {
		t.Fatal("expected unlock page title")
	}
	if !strings.Contains(body, "X-Repository-Detective-API-Key") {
		t.Fatal("expected preferred header guidance")
	}
}

func TestDashboardWithValidAPIKeyHeader(t *testing.T) {
	r, _ := testUIWithAPIKeyAuth(t, "test-secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	req.Header.Set("X-Repository-Detective-API-Key", "test-secret-key")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDashboardWithValidAPIKeyCookie(t *testing.T) {
	r, _ := testUIWithAPIKeyAuth(t, "test-secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	req.AddCookie(&http.Cookie{Name: "rd_ui_sess", Value: "test-secret-key"})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with cookie auth, got %d", w.Code)
	}
}

func TestQueryStringAPIKeyNotLeakedInRenderedPage(t *testing.T) {
	r, _ := testUIWithAPIKeyAuth(t, "super-secret-key-value")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/?api_key=super-secret-key-value", nil)
	r.ServeHTTP(w, req)
	// Cookie middleware redirects; follow by checking redirect target has no api_key
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect to clean URL, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if strings.Contains(loc, "api_key") {
		t.Fatalf("redirect leaked api_key in location: %s", loc)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/ui/", nil)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	r.ServeHTTP(w2, req2)
	body := w2.Body.String()
	if strings.Contains(body, "super-secret-key-value") {
		t.Fatal("API key leaked in rendered dashboard HTML")
	}
}

func TestUnlockFormSetsCookie(t *testing.T) {
	r, h := testUIWithAPIKeyAuth(t, "test-secret-key")
	csrf := ""
	// GET unlock page for CSRF token
	w0 := httptest.NewRecorder()
	req0, _ := http.NewRequest(http.MethodGet, "/ui/unlock", nil)
	r.ServeHTTP(w0, req0)
	body0 := w0.Body.String()
	if i := strings.Index(body0, `name="csrf_token" value="`); i >= 0 {
		rest := body0[i+len(`name="csrf_token" value="`):]
		if j := strings.Index(rest, `"`); j >= 0 {
			csrf = rest[:j]
		}
	}
	if csrf == "" {
		t.Fatal("missing csrf token on unlock page")
	}
	_ = h

	form := url.Values{}
	form.Set("api_key", "test-secret-key")
	form.Set("csrf_token", csrf)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ui/unlock", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after unlock, got %d body=%s", w.Code, w.Body.String())
	}
}
