package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func newAuthTestHandler(t *testing.T) (*Handler, *store.SQLiteStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	path := filepath.Join(t.TempDir(), "ui-auth.db")
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	h, err := NewHandler(s, store.GlobalSettingsSnapshot{}, "/ui", logrus.New(), nil, false, "api-key-secret")
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	h.SetAuthConfig(AuthConfig{
		Mode:                       "local",
		SessionSecret:              "test-session-secret-32chars-min",
		SessionCookieName:          "rd_session",
		SessionTTLHours:            12,
		CSRFEnabled:                true,
		LocalAdminBootstrapEnabled: true,
		PublicURL:                  "http://127.0.0.1:8081",
	})
	return h, s
}

func TestBootstrapBlockedAfterFirstUser(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()
	ctx := context.Background()

	hash, _ := auth.HashPassword("abcdefghijkl1")
	if _, err := s.CreateUser(ctx, store.User{
		Email: "owner@example.com", DisplayName: "Owner", PasswordHash: hash, Role: store.RoleOwner, Enabled: true,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	r := gin.New()
	g := r.Group("/ui")
	h.RegisterAuthRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/ui/bootstrap", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect from bootstrap, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "/ui/login") {
		t.Fatalf("unexpected location %q", w.Header().Get("Location"))
	}
}

func TestBootstrapCreatesOwnerAndSession(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()

	r := gin.New()
	g := r.Group("/ui")
	h.RegisterAuthRoutes(g)

	csrf := security.SessionCSRFToken(h.auth.SessionSecret, "bootstrap", 0)
	form := url.Values{}
	form.Set("csrf_token", csrf)
	form.Set("display_name", "Owner")
	form.Set("email", "owner@example.com")
	form.Set("password", "abcdefghijkl1")
	form.Set("password_confirm", "abcdefghijkl1")

	req := httptest.NewRequest(http.MethodPost, "/ui/bootstrap", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect after bootstrap, got %d body=%s", w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookie {
		if c.Name == "rd_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie after bootstrap")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected HttpOnly session cookie")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}

	count, _ := s.CountUsers(context.Background())
	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}
}

func TestLoginFailureGenericRedirect(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()
	ctx := context.Background()
	hash, _ := auth.HashPassword("abcdefghijkl1")
	_, _ = s.CreateUser(ctx, store.User{
		Email: "admin@example.com", DisplayName: "Admin", PasswordHash: hash, Role: store.RoleAdmin, Enabled: true,
	})

	r := gin.New()
	g := r.Group("/ui")
	h.RegisterAuthRoutes(g)

	csrf := security.SessionCSRFToken(h.auth.SessionSecret, "bootstrap", 0)
	form := url.Values{}
	form.Set("csrf_token", csrf)
	form.Set("email", "admin@example.com")
	form.Set("password", "wrong-password-1")

	req := httptest.NewRequest(http.MethodPost, "/ui/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "error=invalid") {
		t.Fatalf("expected generic invalid redirect, got %q", w.Header().Get("Location"))
	}
}

func TestUIRequiresSessionInLocalMode(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()

	r := gin.New()
	g := r.Group("/ui")
	g.Use(h.SessionAuthMiddleware())
	g.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect to login, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "/ui/login") {
		t.Fatalf("unexpected location %q", w.Header().Get("Location"))
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()
	ctx := context.Background()

	hash, _ := auth.HashPassword("abcdefghijkl1")
	user, _ := s.CreateUser(ctx, store.User{
		Email: "admin@example.com", DisplayName: "Admin", PasswordHash: hash, Role: store.RoleAdmin, Enabled: true,
	})
	sessionID, _ := auth.NewSessionID()
	_ = s.CreateSession(ctx, store.Session{ID: sessionID, UserID: user.ID, ExpiresAt: auth.SessionExpiresAt(12)})

	signed, _ := auth.SignSessionCookie(h.auth.SessionSecret, sessionID)
	csrf := security.SessionCSRFToken(h.auth.SessionSecret, sessionID, user.ID)

	r := gin.New()
	g := r.Group("/ui")
	h.RegisterAuthRoutes(g)

	form := url.Values{}
	form.Set("csrf_token", csrf)
	req := httptest.NewRequest(http.MethodPost, "/ui/logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "rd_session", Value: signed, Path: "/ui/"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect after logout, got %d", w.Code)
	}
	if _, err := s.GetSession(ctx, sessionID); err == nil {
		t.Fatal("expected session deleted after logout")
	}
}
