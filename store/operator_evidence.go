package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Operator evidence keys (RD-017A / RD-014).
const (
	EvidenceWebhookLastDelivery = "webhook.last_delivery"
	EvidenceFirstScanProven     = "proof.first_scan"
	EvidenceWebhookScanProven   = "proof.webhook_scan"
)

// WebhookDeliveryEvidence is a sanitized record of a validated forge webhook.
// Payload bodies and secrets are never stored.
type WebhookDeliveryEvidence struct {
	ReceivedAt string `json:"received_at"`
	EventKind  string `json:"event_kind"` // push | pull_request
	Repository string `json:"repository"`
	DeliveryID string `json:"delivery_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	CommitSHA  string `json:"commit_sha,omitempty"`
	PRNumber   int    `json:"pr_number,omitempty"`
	Source     string `json:"source"` // gitea_webhook
}

// FirstScanEvidence records a terminal scan that satisfies FIRST_SCAN_PROVEN.
// Distinct from WEBHOOK_DELIVERY_E2E_PROVEN / webhook-triggered scan proof.
type FirstScanEvidence struct {
	ProvenAt       string `json:"proven_at"`
	ScanID         string `json:"scan_id"`
	RepositoryID   int64  `json:"repository_id"`
	RepositoryName string `json:"repository_name,omitempty"`
	TriggerType    string `json:"trigger_type"`
	Status         string `json:"status"`
	RequiredOK     int    `json:"required_scanners_ok"`
	RequiredTotal  int    `json:"required_scanners_total"`
	ViaWebhook     bool   `json:"via_webhook"`
}

// PutOperatorEvidence upserts a JSON evidence blob by key.
func (s *SQLiteStore) PutOperatorEvidence(ctx context.Context, key string, value any) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("evidence key required")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO operator_evidence (key, value_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at
	`, key, string(raw), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("put operator evidence: %w", err)
	}
	return nil
}

// GetOperatorEvidenceJSON returns raw JSON for a key, or empty string if missing.
func (s *SQLiteStore) GetOperatorEvidenceJSON(ctx context.Context, key string) (string, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value_json FROM operator_evidence WHERE key = ?`, key).Scan(&raw)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get operator evidence: %w", err)
	}
	return raw, nil
}

// GetWebhookDeliveryEvidence loads last webhook delivery proof if present.
func (s *SQLiteStore) GetWebhookDeliveryEvidence(ctx context.Context) (WebhookDeliveryEvidence, bool, error) {
	raw, err := s.GetOperatorEvidenceJSON(ctx, EvidenceWebhookLastDelivery)
	if err != nil || raw == "" {
		return WebhookDeliveryEvidence{}, false, err
	}
	var ev WebhookDeliveryEvidence
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		return WebhookDeliveryEvidence{}, false, err
	}
	return ev, true, nil
}

// GetFirstScanEvidence loads FIRST_SCAN_PROVEN record if present.
func (s *SQLiteStore) GetFirstScanEvidence(ctx context.Context) (FirstScanEvidence, bool, error) {
	raw, err := s.GetOperatorEvidenceJSON(ctx, EvidenceFirstScanProven)
	if err != nil || raw == "" {
		return FirstScanEvidence{}, false, err
	}
	var ev FirstScanEvidence
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		return FirstScanEvidence{}, false, err
	}
	return ev, true, nil
}

// RecordWebhookDelivery persists sanitized webhook acceptance evidence.
func (s *SQLiteStore) RecordWebhookDelivery(ctx context.Context, ev WebhookDeliveryEvidence) error {
	if ev.ReceivedAt == "" {
		ev.ReceivedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if ev.Source == "" {
		ev.Source = "gitea_webhook"
	}
	return s.PutOperatorEvidence(ctx, EvidenceWebhookLastDelivery, ev)
}

// RecordFirstScanProven persists FIRST_SCAN_PROVEN (and webhook-scan proof when applicable).
func (s *SQLiteStore) RecordFirstScanProven(ctx context.Context, ev FirstScanEvidence) error {
	if ev.ProvenAt == "" {
		ev.ProvenAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := s.PutOperatorEvidence(ctx, EvidenceFirstScanProven, ev); err != nil {
		return err
	}
	if ev.ViaWebhook {
		return s.PutOperatorEvidence(ctx, EvidenceWebhookScanProven, ev)
	}
	return nil
}
