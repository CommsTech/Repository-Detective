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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"git.commsnet.org/commstech/repository-detective/store"
)

func resetWebhookRateLimitersForTest() {
	rateLimiterMu.Lock()
	for key := range rateLimiterMap {
		delete(rateLimiterMap, key)
	}
	rateLimiterMu.Unlock()
}

// fakeReplayStore is a controllable WebhookReplayStore: first sighting of a
// delivery-ID allows dispatch; repeats are rejected unless marked failed.
type fakeReplayStore struct {
	mu          sync.Mutex
	rows        map[string]string // deliveryKey -> status
	errOnClaim  bool
	claims      []store.WebhookDeliveryKey
	completions []store.WebhookDeliveryKey
	dispatches  int
}

func newTestReplayStore() *fakeReplayStore {
	return &fakeReplayStore{rows: map[string]string{}}
}

func replayRowKey(key store.WebhookDeliveryKey) string {
	return key.Forge + "/" + key.DeliveryID + "/" + key.Repository + "/" + key.EventKind
}

func (f *fakeReplayStore) ClaimWebhookDelivery(ctx context.Context, key store.WebhookDeliveryKey) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.errOnClaim {
		return false, context.DeadlineExceeded
	}
	k := replayRowKey(key)
	status, seen := f.rows[k]
	if !seen {
		f.rows[k] = store.WebhookDeliveryStatusProcessing
		f.claims = append(f.claims, key)
		f.dispatches++
		return true, nil
	}
	if status == store.WebhookDeliveryStatusFailed {
		// previously failed before dispatch; forge retry is legitimate
		f.rows[k] = store.WebhookDeliveryStatusProcessing
		return true, nil
	}
	return false, nil
}

func (f *fakeReplayStore) CompleteWebhookDelivery(ctx context.Context, key store.WebhookDeliveryKey, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[replayRowKey(key)] = status
	f.completions = append(f.completions, key)
	return nil
}

// signedWebhookRequest builds a push webhook with a valid HMAC signature.
func (f *fakeReplayStore) dispatchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dispatches
}

func (f *fakeReplayStore) claimCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.claims)
}

// countingProcessor counts push dispatches without shared-channel teardown.
type countingProcessor struct {
	pushes atomic.Int32
}

func (p *countingProcessor) ProcessPush(context.Context, *GiteaWebhookPayload)        { p.pushes.Add(1) }
func (p *countingProcessor) ProcessPullRequest(context.Context, *GiteaWebhookPayload) {}

func signedWebhookRequest(t *testing.T, secret string, body string, deliveryID string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitea-Signature", hex.EncodeToString(mac.Sum(nil)))
	if deliveryID != "" {
		req.Header.Set("X-Gitea-Delivery", deliveryID)
	}
	return req
}

func pushPayloadBody() string {
	payload := map[string]any{
		"ref":   "refs/heads/main",
		"after": "abc123",
		"commits": []map[string]any{
			{"id": "c1", "added": []string{}, "removed": []string{}, "modified": []string{"main.go"}},
		},
		"repository": map[string]any{"full_name": "org/repo"},
	}
	body, _ := json.Marshal(payload)
	return string(body)
}

func serveWebhook(h *WebhookHandler, req *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook", h.HandleWebhook)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// A replayed signed delivery with the same delivery-ID must dispatch exactly
// once across both sightings.
func TestHandleWebhookReplayRejected(t *testing.T) {
	resetWebhookRateLimitersForTest()
	secretKey := "test-webhook-secret"
	fs := newTestReplayStore()
	h := NewWebhookHandler(logrus.New(), &Config{WebhookSecret: secretKey}, &countingProcessor{})
	h.SetReplayStore(fs)

	body := pushPayloadBody()
	w1 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, "delivery-1"))
	if w1.Code != http.StatusOK {
		t.Fatalf("first delivery: got %d, want 200 (%s)", w1.Code, w1.Body.String())
	}

	w2 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, "delivery-1"))
	if !strings.Contains(w2.Body.String(), "duplicate") {
		t.Fatalf("replayed delivery must be reported as duplicate, got %d %s", w2.Code, w2.Body.String())
	}

	if got := fs.dispatchCount(); got != 1 {
		t.Fatalf("expected exactly 1 dispatch across replay, got %d", got)
	}
}

// A different delivery-ID is a distinct event and must dispatch normally.
func TestHandleWebhookDistinctDeliveryDispatches(t *testing.T) {
	resetWebhookRateLimitersForTest()
	secretKey := "test-webhook-secret"
	fs := newTestReplayStore()
	h := NewWebhookHandler(logrus.New(), &Config{WebhookSecret: secretKey}, &countingProcessor{})
	h.SetReplayStore(fs)

	body := pushPayloadBody()
	w1 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, "delivery-a"))
	w2 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, "delivery-b"))
	if w1.Code != http.StatusOK || strings.Contains(w1.Body.String(), "duplicate") {
		t.Fatalf("delivery-a must dispatch, got %d %s", w1.Code, w1.Body.String())
	}
	if w2.Code != http.StatusOK || strings.Contains(w2.Body.String(), "duplicate") {
		t.Fatalf("delivery-b must dispatch, got %d %s", w2.Code, w2.Body.String())
	}
	if got := fs.dispatchCount(); got != 2 {
		t.Fatalf("expected 2 dispatches, got %d", got)
	}
}

// Without a delivery-ID header the gate cannot identify the delivery; dispatch
// always proceeds and nothing is claimed (documented compatibility behavior).
func TestHandleWebhookMissingDeliveryIDAlwaysDispatches(t *testing.T) {
	resetWebhookRateLimitersForTest()
	secretKey := "test-webhook-secret"
	fs := newTestReplayStore()
	proc := &countingProcessor{}
	h := NewWebhookHandler(logrus.New(), &Config{WebhookSecret: secretKey}, proc)
	h.SetReplayStore(fs)

	body := pushPayloadBody()
	w1 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, ""))
	w2 := serveWebhook(h, signedWebhookRequest(t, secretKey, body, ""))
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("both identical deliveries without ID must dispatch, got %d and %d", w1.Code, w2.Code)
	}
	if !strings.Contains(w1.Body.String(), "processing") || !strings.Contains(w2.Body.String(), "processing") {
		t.Fatalf("both deliveries must proceed to processing, got %q and %q", w1.Body.String(), w2.Body.String())
	}
	if got := fs.claimCount(); got != 0 {
		t.Fatalf("deliveries without ID must never be claimed, got %d claims", got)
	}
	// Dispatch happens in goroutines after the response; wait briefly for both.
	deadline := time.Now().Add(2 * time.Second)
	for proc.pushes.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := proc.pushes.Load(); got != 2 {
		t.Fatalf("expected 2 dispatch goroutines without delivery IDs, got %d", got)
	}
}

// A replay-store outage must fail closed (503): the forge retries later rather
// than risking an unchecked duplicate dispatch.
func TestHandleWebhookReplayStoreErrorFailsClosed(t *testing.T) {
	resetWebhookRateLimitersForTest()
	secretKey := "test-webhook-secret"
	fs := newTestReplayStore()
	fs.errOnClaim = true
	h := NewWebhookHandler(logrus.New(), &Config{WebhookSecret: secretKey}, &countingProcessor{})
	h.SetReplayStore(fs)

	w := serveWebhook(h, signedWebhookRequest(t, secretKey, pushPayloadBody(), "delivery-x"))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("replay store error must fail closed with 503, got %d (%s)", w.Code, w.Body.String())
	}
	if got := fs.dispatchCount(); got != 0 {
		t.Fatalf("failed claim must not dispatch, got %d dispatches", got)
	}
}
