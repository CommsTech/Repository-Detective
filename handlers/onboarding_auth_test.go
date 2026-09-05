package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestOnboardDefaultsRecommendLocalForFreshInstall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOnboardingHandler(logrus.New(), OnboardingConfig{
		GiteaURL:  "https://git.example.com",
		PublicURL: "https://detective.example.com",
		AuthMode:  "api_key_only", // runtime default for upgrades
	})
	h.SetInstallCounters(
		func(context.Context) (int, error) { return 0, nil },
		func(context.Context) (int, error) { return 0, nil },
	)

	r := gin.New()
	r.GET("/defaults", h.handleDefaultsExtended)
	req := httptest.NewRequest(http.MethodGet, "/defaults", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["fresh_install"] != true {
		t.Fatalf("expected fresh_install=true, got %#v", body["fresh_install"])
	}
	if body["recommend_local_auth"] != true {
		t.Fatalf("expected recommend_local_auth=true, got %#v", body["recommend_local_auth"])
	}
	if body["auth_mode"] != "api_key_only" {
		t.Fatalf("runtime auth_mode should remain api_key_only until configured, got %#v", body["auth_mode"])
	}
	rec, _ := body["auth_recommendation"].(map[string]any)
	if rec == nil || rec["mode"] != "local" || rec["bootstrap"] != "/ui/bootstrap" {
		t.Fatalf("unexpected auth_recommendation: %#v", body["auth_recommendation"])
	}
}

func TestOnboardDefaultsDoesNotRecommendWhenUsersExist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOnboardingHandler(logrus.New(), OnboardingConfig{
		AuthMode: "api_key_only",
	})
	h.SetInstallCounters(
		func(context.Context) (int, error) { return 1, nil },
		func(context.Context) (int, error) { return 3, nil },
	)

	r := gin.New()
	r.GET("/defaults", h.handleDefaultsExtended)
	req := httptest.NewRequest(http.MethodGet, "/defaults", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["fresh_install"] != false {
		t.Fatalf("expected fresh_install=false, got %#v", body["fresh_install"])
	}
	if body["recommend_local_auth"] != false {
		t.Fatalf("existing api_key_only install must not auto-recommend migrate, got %#v", body["recommend_local_auth"])
	}
}
