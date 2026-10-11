package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestGetRepoSettingsIncludesProfile(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandardDeterministic
	r := testRouterWithGlobal(t, s, global)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/repos/1/settings", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["scan_profile"] != store.ScanProfileStandard {
		t.Fatalf("expected scan_profile in response, got %v", resp["scan_profile"])
	}
	if _, ok := resp["effective_profile_summary"]; !ok {
		t.Fatal("expected effective_profile_summary")
	}
}

func TestUpdateRepoSettingsInvalidProfile(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"scan_profile":"invalid_profile"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body %s", w.Code, w.Body.String())
	}
}

func TestUpdateRepoSettingsProfile(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"scan_profile":"fast"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["scan_profile"] != store.ScanProfileLight {
		t.Fatalf("expected light profile (canonical for fast), got %v", resp["scan_profile"])
	}
}

func testRouterWithGlobal(t *testing.T, s store.QueryStore, global store.GlobalSettingsSnapshot) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewHandler(s, global, logrus.New())
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)
	return r
}
