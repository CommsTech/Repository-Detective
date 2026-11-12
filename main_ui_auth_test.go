package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIStatusWithoutKeyReturnsJSON401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config
	defer func() { config = saved }()
	config = &Config{APIKey: "test-secret-key"}

	r := gin.New()
	r.GET("/api/v1/status", requireAPIKeyAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "API key required") {
		t.Fatalf("expected JSON error body, got %q", w.Body.String())
	}
}
