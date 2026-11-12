package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestControlPlaneRoutesRequireAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.OpenSQLite(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	r := gin.New()
	r.Use(security.MiddlewareHeaders())
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) {
		if c.GetHeader("X-Repository-Detective-API-Key") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}
		c.Next()
	})
	api.NewHandler(s, store.DefaultGlobalSettings(), logrus.New()).RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestControlPlaneSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(security.MiddlewareHeaders())
	r.GET("/api/v1/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing X-Frame-Options")
	}
}
