package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/notify"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func testNotifyRouter(t *testing.T, cfg notify.Config) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	api.NewNotificationHandler(notify.NewManager(cfg, nil, nil, nil)).RegisterRoutes(g)
	return r
}

func enabledNotifyConfig() notify.Config {
	cfg := notify.DefaultConfig()
	cfg.Enabled = true
	cfg.TelegramEnabled = true
	cfg.TelegramBotToken = "token"
	cfg.TelegramChatID = "1"
	return cfg
}

func TestNotificationStatusWithoutSecrets(t *testing.T) {
	cfg := enabledNotifyConfig()
	r := testNotifyRouter(t, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if bytes.Contains([]byte(body), []byte("token")) {
		t.Fatal("token leaked in status response")
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", resp["enabled"])
	}
}

func TestNotificationTestDisabled(t *testing.T) {
	cfg := notify.DefaultConfig()
	r := testNotifyRouter(t, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/test", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestUpdateRepoSettingsInvalidNotificationSeverity(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	r := testRouter(t, s)

	body := []byte(`{"notification_min_severity":"bogus"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/repos/1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body %s", w.Code, w.Body.String())
	}
}

func TestGetRepoSettingsIncludesNotifications(t *testing.T) {
	s := testStore(t)
	seedRepo(t, s)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewHandler(s, store.DefaultGlobalSettings(), logrus.New())
	cfg := enabledNotifyConfig()
	h.SetNotificationGlobal(cfg)
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/repos/1/settings", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("notification_global")) {
		t.Fatal("expected notification_global in settings response")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("effective_notifications")) {
		t.Fatal("expected effective_notifications in settings response")
	}
}
