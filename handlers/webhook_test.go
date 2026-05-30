package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestVerifyWebhookSecretPlainText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	handler := &WebhookHandler{
		logger: logger,
		config: &Config{WebhookSecret: "super-secret"},
	}

	t.Run("valid secret in query", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/webhook?secret=super-secret", nil)

		if err := handler.verifyWebhookSecret(c); err != nil {
			t.Fatalf("expected valid secret, got error: %v", err)
		}
	})

	t.Run("invalid secret", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/webhook?secret=wrong", nil)

		if err := handler.verifyWebhookSecret(c); err == nil {
			t.Fatal("expected invalid secret error")
		}
	})

	t.Run("missing secret when configured", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/webhook", nil)

		if err := handler.verifyWebhookSecret(c); err == nil {
			t.Fatal("expected missing secret error")
		}
	})
}

func TestVerifyWebhookSecretHexEncoded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	secretHex := "736563726574"
	handler := &WebhookHandler{
		logger: logger,
		config: &Config{WebhookSecret: secretHex},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhook?secret="+secretHex, nil)

	if err := handler.verifyWebhookSecret(c); err != nil {
		t.Fatalf("expected valid hex secret, got error: %v", err)
	}
}
