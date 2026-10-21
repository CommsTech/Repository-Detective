package store

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func openWebhookReplayTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := OpenSQLite(filepath.Join(t.TempDir(), "replay.db"))
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func testWebhookKey() WebhookDeliveryKey {
	return WebhookDeliveryKey{
		Forge: "gitea", DeliveryID: "delivery-1",
		Repository: "org/repo", EventKind: "push",
	}
}

func TestClaimWebhookDeliveryDuplicateContract(t *testing.T) {
	s := openWebhookReplayTestStore(t)
	ctx := context.Background()
	key := testWebhookKey()

	allowed, err := s.ClaimWebhookDelivery(ctx, key)
	if err != nil || !allowed {
		t.Fatalf("first claim: allowed=%v err=%v", allowed, err)
	}
	allowed, err = s.ClaimWebhookDelivery(ctx, key)
	if err != nil {
		t.Fatalf("in-flight duplicate must not be an error: %v", err)
	}
	if allowed {
		t.Fatal("in-flight duplicate must not dispatch")
	}

	if err := s.CompleteWebhookDelivery(ctx, key, WebhookDeliveryStatusProcessed); err != nil {
		t.Fatalf("complete processed: %v", err)
	}
	allowed, err = s.ClaimWebhookDelivery(ctx, key)
	if err != nil {
		t.Fatalf("processed duplicate must not be an error: %v", err)
	}
	if allowed {
		t.Fatal("processed duplicate must not dispatch")
	}
}

func TestClaimWebhookDeliveryFailedClaimCanRetry(t *testing.T) {
	s := openWebhookReplayTestStore(t)
	ctx := context.Background()
	key := testWebhookKey()

	allowed, err := s.ClaimWebhookDelivery(ctx, key)
	if err != nil || !allowed {
		t.Fatalf("first claim: allowed=%v err=%v", allowed, err)
	}
	if err := s.CompleteWebhookDelivery(ctx, key, WebhookDeliveryStatusFailed); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	allowed, err = s.ClaimWebhookDelivery(ctx, key)
	if err != nil || !allowed {
		t.Fatalf("failed delivery retry must dispatch: allowed=%v err=%v", allowed, err)
	}
}

func TestClaimWebhookDeliveryWithoutIDIsNotRecorded(t *testing.T) {
	s := openWebhookReplayTestStore(t)
	ctx := context.Background()
	key := testWebhookKey()
	key.DeliveryID = ""

	allowed, err := s.ClaimWebhookDelivery(ctx, key)
	if err != nil || !allowed {
		t.Fatalf("missing-ID delivery must proceed: allowed=%v err=%v", allowed, err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_deliveries").Scan(&count); err != nil {
		t.Fatalf("count replay rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("missing-ID delivery must not be recorded, rows=%d", count)
	}
}

func TestClaimWebhookDeliveryConcurrentExactlyOneDispatch(t *testing.T) {
	s := openWebhookReplayTestStore(t)
	ctx := context.Background()
	key := testWebhookKey()

	const workers = 24
	var allowed atomic.Int32
	var failures atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.ClaimWebhookDelivery(ctx, key)
			if err != nil {
				failures.Add(1)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if failures.Load() != 0 {
		t.Fatalf("concurrent duplicate claims returned %d errors", failures.Load())
	}
	if allowed.Load() != 1 {
		t.Fatalf("expected exactly one dispatch, got %d", allowed.Load())
	}
}
