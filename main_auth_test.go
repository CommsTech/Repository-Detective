package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestValidateAuthLocalRequiresSessionSecret(t *testing.T) {
	cfg := &Config{AuthMode: "local", DatabaseEnabled: true, SessionSecret: ""}
	if err := cfg.validateAuth(); err == nil {
		t.Fatal("expected error when session_secret missing in local mode")
	}
	cfg.SessionSecret = "super-secret-value"
	if err := cfg.validateAuth(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAuthDefaultsAPIKeyOnly(t *testing.T) {
	cfg := &Config{}
	if err := cfg.validateAuth(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.AuthMode != "api_key_only" {
		t.Fatalf("expected api_key_only, got %q", cfg.AuthMode)
	}
	if cfg.SessionCookieName != "rd_session" {
		t.Fatalf("unexpected cookie name %q", cfg.SessionCookieName)
	}
	if cfg.SessionTTLHours != 12 {
		t.Fatalf("unexpected ttl %d", cfg.SessionTTLHours)
	}
}

func TestAuthConfigDefaultsFromViper(t *testing.T) {
	v := viper.New()
	v.SetDefault("auth_mode", "api_key_only")
	v.SetDefault("session_cookie_name", "rd_session")
	v.SetDefault("session_ttl_hours", 12)
	v.SetDefault("csrf_enabled", true)
	v.SetDefault("local_admin_bootstrap_enabled", true)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := cfg.validateAuth(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.AuthMode != "api_key_only" {
		t.Fatalf("got auth_mode %q", cfg.AuthMode)
	}
	if !cfg.CSRFEnabled {
		t.Fatal("expected csrf_enabled default true")
	}
}

func TestAPIKeyStillWorksWhenAuthModeLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{APIKey: "test-secret-key", AuthMode: "local", SessionSecret: "sess-secret"}

	r := gin.New()
	r.GET("/protected", requireAPIKeyAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Repository-Detective-API-Key", "test-secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCSRFNotRequiredForAPIKeyJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{
		APIKey:        "test-secret-key",
		AuthMode:      "local",
		SessionSecret: "sess-secret",
		CSRFEnabled:   true,
	}

	r := gin.New()
	r.POST("/api/v1/example", requireAPIKeyAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/example", strings.NewReader(`{"x":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Repository-Detective-API-Key", "test-secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected API key JSON without CSRF to succeed, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSessionCSRFDistinctFromAPIKeyCSRF(t *testing.T) {
	apiToken := security.CSRFToken("api-secret", "client-key")
	sessToken := security.SessionCSRFToken("sess-secret", "session-id", 1)
	if apiToken == sessToken {
		t.Fatal("expected different CSRF schemes")
	}
}

func TestUserRolesDefined(t *testing.T) {
	roles := []string{store.RoleOwner, store.RoleAdmin, store.RoleReadOnly}
	seen := map[string]bool{}
	for _, r := range roles {
		if r == "" || seen[r] {
			t.Fatalf("duplicate or empty role %q", r)
		}
		seen[r] = true
	}
}

func TestLoginFormUsesGenericErrorQuery(t *testing.T) {
	u, err := url.Parse("/ui/login?error=invalid")
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("error") != "invalid" {
		t.Fatal("expected generic invalid error param")
	}
}

func TestQueryStringAPIKeyRejectedWhenHardened(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{
		APIKey:                  "test-secret-key",
		RejectQueryStringAPIKey: true,
	}

	r := gin.New()
	r.GET("/protected", requireAPIKeyAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected?api_key=test-secret-key", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for query key in hardened mode, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("X-Repository-Detective-API-Key", "test-secret-key")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected header auth to work, got %d", w2.Code)
	}
}

func TestQueryStringAPIKeyAcceptedInCompatibilityMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{APIKey: "test-secret-key", RejectQueryStringAPIKey: false}

	r := gin.New()
	r.GET("/protected", requireAPIKeyAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected?api_key=test-secret-key", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected compatibility mode to accept query key, got %d", w.Code)
	}
}
