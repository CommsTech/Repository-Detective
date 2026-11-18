package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func TestLocalMutationsRejectReadOnlyAndUnknownRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{auth: AuthConfig{Mode: "local"}}
	for _, tc := range []struct {
		role    string
		enabled bool
		want    int
	}{
		{store.RoleViewer, true, http.StatusForbidden},
		{store.RoleReadOnly, true, http.StatusForbidden},
		{"unrecognized", true, http.StatusForbidden},
		{store.RoleDeveloper, true, http.StatusNoContent},
		{store.RoleSecurity, true, http.StatusNoContent},
		{store.RoleOwner, true, http.StatusNoContent},
		{store.RoleOwner, false, http.StatusForbidden},
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(ctxAuthUser, store.User{Role: tc.role, Enabled: tc.enabled}); c.Next() })
		r.Use(h.rejectReadOnlyMutations())
		r.POST("/write", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		r.GET("/read", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/write", nil))
		if w.Code != tc.want {
			t.Fatalf("role %q enabled %t: got %d want %d", tc.role, tc.enabled, w.Code, tc.want)
		}
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/read", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("GET unexpectedly gated for %q: %d", tc.role, w.Code)
		}
	}
}
