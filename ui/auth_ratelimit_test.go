package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func TestLoginRateLimited(t *testing.T) {
	h, s := newAuthTestHandler(t)
	defer s.Close()
	h.loginLimiter = auth.NewLoginLimiter(0.01, 1, 64) // burst 1, then deny

	hash, _ := auth.HashPassword("abcdefghijkl1")
	if _, err := s.CreateUser(context.Background(), store.User{
		Email: "admin@example.com", DisplayName: "Admin", PasswordHash: hash, Role: store.RoleAdmin, Enabled: true,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	r := gin.New()
	g := r.Group("/ui")
	h.RegisterAuthRoutes(g)

	csrf := security.SessionCSRFToken(h.auth.SessionSecret, "bootstrap", 0)
	submit := func() *httptest.ResponseRecorder {
		form := url.Values{}
		form.Set("csrf_token", csrf)
		form.Set("email", "admin@example.com")
		form.Set("password", "wrong-password-1")
		req := httptest.NewRequest(http.MethodPost, "/ui/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	first := submit()
	if first.Code == http.StatusTooManyRequests {
		t.Fatalf("first attempt should not be rate limited, got %d", first.Code)
	}
	second := submit()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second attempt, got %d body=%s", second.Code, second.Body.String())
	}
}
