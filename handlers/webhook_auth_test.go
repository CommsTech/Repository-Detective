package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestVerifyWebhookSecretRequiresHMAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &WebhookHandler{
		logger: logrus.New(),
		config: &Config{WebhookSecret: "super-secret"},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhook?secret=super-secret", nil)

	if err := handler.verifyWebhookSecret(c, []byte(`{}`)); err == nil {
		t.Fatal("expected query-string secret to be rejected")
	}
}

func TestVerifyWebhookSecretRejectsMissingSecretConfig(t *testing.T) {
	handler := &WebhookHandler{
		logger: logrus.New(),
		config: &Config{WebhookSecret: ""},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhook", nil)
	c.Request.Header.Set("X-Gitea-Signature", "abc")
	if err := handler.verifyWebhookSecret(c, []byte(`{}`)); err == nil {
		t.Fatal("expected error when webhook secret is not configured")
	}
}

func TestVerifyWebhookSecretAllowsInsecureMode(t *testing.T) {
	handler := &WebhookHandler{
		logger: logrus.New(),
		config: &Config{AllowInsecureWebhooks: true},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhook", nil)
	if err := handler.verifyWebhookSecret(c, []byte(`{}`)); err != nil {
		t.Fatalf("expected insecure mode to allow webhook: %v", err)
	}
}
