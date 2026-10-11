package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"github.com/gin-gonic/gin"
)

func TestAITestConnectionWithoutProviderReturnsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewAIHandler(nil)
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/test-connection", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for disabled AI, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["policy_disabled"] != true {
		t.Fatalf("expected policy_disabled=true, got %#v", body)
	}
	if body["ai_analysis"] != "Disabled" {
		t.Fatalf("expected ai_analysis=Disabled, got %#v", body)
	}
}
