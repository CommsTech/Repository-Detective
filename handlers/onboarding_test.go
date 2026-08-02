package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/handlers"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestOnboardingPageNoRedirectLoop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewOnboardingHandler(logrus.New(), handlers.OnboardingConfig{})
	h.RegisterRoutes(router, router.Group("/api/v1/onboard"))

	for _, path := range []string{"/onboard", "/onboard/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d (Location: %q)", path, rec.Code, rec.Header().Get("Location"))
		}
		if loc := rec.Header().Get("Location"); loc != "" {
			t.Fatalf("GET %s: unexpected redirect to %q", path, loc)
		}
		if body := rec.Body.String(); len(body) < 50 || body[:15] != "<!DOCTYPE html>" {
			t.Fatalf("GET %s: expected HTML body", path)
		}
	}
}

func TestRootRedirectsToOnboardSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/onboard/")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/onboard/" {
		t.Fatalf("expected Location /onboard/, got %q", got)
	}
}
