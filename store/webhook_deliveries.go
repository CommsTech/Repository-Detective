package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Durable webhook replay protection (Beta 6 P1).
//
// Signed forge deliveries are deduplicated across process restarts so a
// replayed webhook can never dispatch analysis twice. A claim row is keyed by
// (forge, delivery ID, repository, event kind) and starts in status
// "processing"; the webhook handler flips it to "processed" once dispatch
// launched, or to "failed" when handling failed before dispatch — a "failed"
// claim re-allows dispatch on the next attempt (legitimate forge retry).
// Deliveries without a forge delivery-ID header cannot be identified and are
// never claimed (documented behavior: dispatch always proceeds for those).

const (
	// DefaultWebhookReplayRetention keeps 7 days of replay state; pruning stays
	// bounded via this window plus a hard row cap.
	DefaultWebhookReplayRetention = 168 * time.Hour
	// DefaultWebhookReplayMaxRows bounds the webhook_deliveries table size.
	DefaultWebhookReplayMaxRows = 100000
	// DefaultWebhookReplayPruneInterval throttles opportunistic pruning to
	// one pass per hour even under heavy forged-delivery load.
	DefaultWebhookReplayPruneInterval = time.Hour

	WebhookDeliveryStatusProcessing = "processing"
	WebhookDeliveryStatusProcessed  = "processed"
	WebhookDeliveryStatusFailed     = "failed"
)

var webhookDeliveryTerminalStatuses = map[string]bool{
	WebhookDeliveryStatusProcessed: true,
	WebhookDeliveryStatusFailed:    true,
}

// WebhookDeliveryKey identifies a signed forge delivery for replay protection.
// The key combines the forge delivery ID with the repository/event context it
// was delivered for, so replayed IDs observed for one target cannot suppress a
// different target's delivery.
type WebhookDeliveryKey struct {
	Forge      string
	DeliveryID string
	Repository string
	EventKind  string
}

// SetWebhookReplayConfig configures bounded retention for replay state.
// Non-positive values fall back to the package defaults. Call it once after
// store open; lazy pruning inside ClaimWebhookDelivery uses these bounds.
func (s *SQLiteStore) SetWebhookReplayConfig(retention time.Duration, maxRows int) {
	if retention <= 0 {
		retention = DefaultWebhookReplayRetention
	}
	if maxRows <= 0 {
		maxRows = DefaultWebhookReplayMaxRows
	}
	s.webhookReplayMu.Lock()
	s.webhookReplayRetention = retention
	s.webhookReplayMaxRows = maxRows
	s.webhookReplayMu.Unlock()
}

// ClaimWebhookDelivery records an accepted signed delivery and reports whether
// dispatch may proceed:
//   - first sighting  -> true (row created in "processing");
//   - duplicate of an in-flight ("processing") or already-dispatched
//     ("processed") delivery -> false;
//   - duplicate of a "failed" delivery -> true (our earlier handling failed
//     before dispatch, so the forge retry is legitimate);
//   - deliveries without a delivery ID -> true and are not recorded.
func (s *SQLiteStore) ClaimWebhookDelivery(ctx context.Context, key WebhookDeliveryKey) (bool, error) {
	if strings.TrimSpace(key.DeliveryID) == "" {
		// No forge delivery ID header — replays are indistinguishable from
		// legitimate retries, so deduplication is skipped by design.
		return true, nil
	}
	if key.Forge == "" {
		key.Forge = "gitea"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	// 1) Retry path: a previously failed claim re-opens its slot for dispatch.
	res, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		   SET status = ?, dispatch_count = dispatch_count + 1, last_seen_at = ?, processed_at = NULL
		 WHERE forge = ? AND delivery_id = ? AND repository = ? AND event_kind = ? AND status = ?`,
		WebhookDeliveryStatusProcessing, now,
		key.Forge, key.DeliveryID, key.Repository, key.EventKind, WebhookDeliveryStatusFailed,
	)
	if err != nil {
		return false, fmt.Errorf("claim webhook delivery retry: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		s.maybePruneWebhookDeliveries(ctx)
		return true, nil
	}

	// 2) First sighting: create the claim atomically via the UNIQUE key.
	// ON CONFLICT DO NOTHING is the duplicate path: an in-flight or already
	// processed delivery is not an infrastructure error and must return
	// (false, nil), allowing the handler to acknowledge it with HTTP 200
	// instead of causing the forge to retry a delivery we already dispatched.
	res, err = s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries
		       (forge, delivery_id, repository, event_kind, status, dispatch_count, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(forge, delivery_id, repository, event_kind) DO NOTHING`,
		key.Forge, key.DeliveryID, key.Repository, key.EventKind, WebhookDeliveryStatusProcessing, now, now,
	)
	if err != nil {
		return false, fmt.Errorf("claim webhook delivery insert: %w", err)
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim webhook delivery rows affected: %w", err)
	}
	if inserted == 0 {
		return false, nil
	}
	s.maybePruneWebhookDeliveries(ctx)
	return true, nil
}

// CompleteWebhookDelivery finalizes a claimed delivery with a terminal status
// ("processed" or "failed"). Missing rows are tolerated so completion stays
// idempotent even if a concurrent prune removed the row.
func (s *SQLiteStore) CompleteWebhookDelivery(ctx context.Context, key WebhookDeliveryKey, status string) error {
	if strings.TrimSpace(key.DeliveryID) == "" {
		return nil
	}
	if !webhookDeliveryTerminalStatuses[status] {
		return fmt.Errorf("invalid webhook delivery status %q", status)
	}
	if key.Forge == "" {
		key.Forge = "gitea"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		   SET status = ?, processed_at = ?
		 WHERE forge = ? AND delivery_id = ? AND repository = ? AND event_kind = ? AND status = ?`,
		status, now, key.Forge, key.DeliveryID, key.Repository, key.EventKind, WebhookDeliveryStatusProcessing,
	)
	if err != nil {
		return fmt.Errorf("complete webhook delivery: %w", err)
	}
	return nil
}

// PruneWebhookDeliveries enforces bounded replay-state retention: rows whose
// first_seen_at passed the retention window are deleted, and the table is
// trimmed to the newest maxRows entries. Returns the number of rows pruned.
func (s *SQLiteStore) PruneWebhookDeliveries(ctx context.Context) (int64, error) {
	retention := DefaultWebhookReplayRetention
	maxRows := DefaultWebhookReplayMaxRows
	s.webhookReplayMu.Lock()
	if s.webhookReplayRetention > 0 {
		retention = s.webhookReplayRetention
	}
	if s.webhookReplayMaxRows > 0 {
		maxRows = s.webhookReplayMaxRows
	}
	s.webhookReplayMu.Unlock()

	var pruned int64
	cutoff := time.Now().UTC().Add(-retention).Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE first_seen_at < ?`, cutoff)
	if err != nil {
		return pruned, fmt.Errorf("prune webhook deliveries (retention): %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		pruned += n
	}
	res, err = s.db.ExecContext(ctx, `
		DELETE FROM webhook_deliveries
		 WHERE id NOT IN (SELECT id FROM webhook_deliveries ORDER BY id DESC LIMIT ?)`,
		maxRows,
	)
	if err != nil {
		if pruned > 0 {
			return pruned, fmt.Errorf("prune webhook deliveries (row cap): %w", err)
		}
		return 0, fmt.Errorf("prune webhook deliveries (row cap): %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		pruned += n
	}
	return pruned, nil
}

// maybePruneWebhookDeliveries prunes at most once per interval; best effort —
// pruning failures never block webhook acceptance.
func (s *SQLiteStore) maybePruneWebhookDeliveries(ctx context.Context) {
	interval := s.webhookReplayPruneEvery
	if interval <= 0 {
		interval = DefaultWebhookReplayPruneInterval
	}
	s.webhookReplayMu.Lock()
	due := time.Since(s.webhookReplayLastPrune) >= interval
	if due {
		s.webhookReplayLastPrune = time.Now()
	}
	s.webhookReplayMu.Unlock()
	if !due {
		return
	}
	_, _ = s.PruneWebhookDeliveries(ctx)
}

// GetWebhookDeliveryCount returns the number of stored delivery claims
// (operational troubleshooting helper).
func (s *SQLiteStore) GetWebhookDeliveryCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM webhook_deliveries`).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("count webhook deliveries: %w", err)
	}
	return count, nil
}
