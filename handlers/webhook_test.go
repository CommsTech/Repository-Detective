package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestVerifyWebhookSecretHMACSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "my-webhook-secret"
	body := []byte(`{"ref":"refs/heads/main","commits":[{"id":"abc"}]}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	handler := &WebhookHandler{
		logger: logrus.New(),
		config: &Config{WebhookSecret: secret},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-Gitea-Signature", sig)
	c.Request = req

	if err := handler.verifyWebhookSecret(c, body); err != nil {
		t.Fatalf("expected valid HMAC signature, got error: %v", err)
	}
}

func TestVerifyWebhookSecretInvalidHMAC(t *testing.T) {
	handler := &WebhookHandler{
		logger: logrus.New(),
		config: &Config{WebhookSecret: "secret"},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhook", nil)
	c.Request.Header.Set("X-Gitea-Signature", "deadbeef")

	if err := handler.verifyWebhookSecret(c, []byte(`{"test":true}`)); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestClassifyWebhookEventPush(t *testing.T) {
	payload := &GiteaWebhookPayload{
		Ref:     "refs/heads/main",
		After:   "abc123",
		Commits: []Commit{{ID: "abc123"}},
	}
	if classifyWebhookEvent(payload) != webhookEventPush {
		t.Fatal("expected push event")
	}
}

func TestClassifyWebhookEventPullRequest(t *testing.T) {
	payload := &GiteaWebhookPayload{
		Action: "synchronized",
		PullRequest: PullRequest{
			Number: 7,
			State:  "open",
		},
	}
	if classifyWebhookEvent(payload) != webhookEventPullRequest {
		t.Fatal("expected pull request event")
	}
}

type captureProcessor struct {
	pushCalled bool
	prCalled   bool
	pushDone   chan struct{}
}

func (p *captureProcessor) ProcessPush(_ context.Context, _ *GiteaWebhookPayload) {
	p.pushCalled = true
	if p.pushDone != nil {
		close(p.pushDone)
	}
}

func (p *captureProcessor) ProcessPullRequest(_ context.Context, _ *GiteaWebhookPayload) {
	p.prCalled = true
}

func TestHandleWebhookPushWithoutActionField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	processor := &captureProcessor{pushDone: make(chan struct{})}
	handler := &WebhookHandler{
		logger:    logrus.New(),
		config:    &Config{WebhookSecret: "secret"},
		processor: processor,
	}

	payload := GiteaWebhookPayload{
		Ref:     "refs/heads/main",
		After:   "abc",
		Commits: []Commit{{ID: "abc", Added: []string{"main.go"}}},
		Repository: Repository{
			FullName: "org/repo",
			Name:     "repo",
			Owner:    User{Login: "org"},
		},
	}
	body, _ := json.Marshal(payload)

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-Gitea-Signature", hex.EncodeToString(mac.Sum(nil)))
	c.Request = req

	handler.HandleWebhook(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	select {
	case <-processor.pushDone:
	case <-time.After(2 * time.Second):
		t.Fatal("expected push handler to run for payload without action field")
	}
	if !processor.pushCalled {
		t.Fatal("expected pushCalled flag to be set")
	}
}

func TestUserLoginName(t *testing.T) {
	if (User{Login: "from-login"}).LoginName() != "from-login" {
		t.Fatal("expected login field")
	}
	if (User{Username: "from-username"}).LoginName() != "from-username" {
		t.Fatal("expected username field")
	}
}

var _ AnalysisProcessor = (*captureProcessor)(nil)
