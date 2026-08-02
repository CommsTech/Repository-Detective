package security_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"github.com/gin-gonic/gin"
)

func TestMiddlewareHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(security.MiddlewareHeaders())
	r.GET("/ui/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ui/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	for _, header := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Content-Security-Policy", "Cache-Control"} {
		if w.Header().Get(header) == "" {
			t.Fatalf("missing header %s", header)
		}
	}
}

func TestMiddlewareMaxBodyRejectsLargePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(security.MiddlewareMaxBody(32))
	r.POST("/api/v1/test", func(c *gin.Context) {
		var body map[string]any
		if err := c.BindJSON(&body); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.Status(http.StatusOK)
	})

	body := []byte(`{"repo_url":"` + strings.Repeat("a", 128) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusBadRequest {
		t.Fatalf("expected body limit rejection, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCSRFToken(t *testing.T) {
	tok := security.CSRFToken("secret", "client-key")
	if tok == "" {
		t.Fatal("expected token")
	}
	if !security.ValidCSRFToken("secret", "client-key", tok) {
		t.Fatal("token should validate")
	}
	if security.ValidCSRFToken("secret", "client-key", "wrong") {
		t.Fatal("wrong token should fail")
	}
}
