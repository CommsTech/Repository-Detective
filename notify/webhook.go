package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const signatureHeader = "X-Repository-Detective-Signature"

// WebhookChannel posts signed JSON events to a generic webhook URL.
type WebhookChannel struct {
	enabled bool
	url     string
	secret  string
	poster  HTTPPoster
}

// NewWebhookChannel creates a generic webhook notifier.
func NewWebhookChannel(cfg Config, poster HTTPPoster) *WebhookChannel {
	if poster == nil {
		poster = newDefaultHTTPPoster()
	}
	return &WebhookChannel{
		enabled: cfg.WebhookEnabled && cfg.WebhookURL != "",
		url:     cfg.WebhookURL,
		secret:  cfg.WebhookSecret,
		poster:  poster,
	}
}

func (c *WebhookChannel) Name() string  { return "webhook" }
func (c *WebhookChannel) Enabled() bool { return c.enabled }

type webhookPayload struct {
	Event      string         `json:"event"`
	Severity   string         `json:"severity,omitempty"`
	Repository string         `json:"repository,omitempty"`
	ScanID     string         `json:"scan_id,omitempty"`
	Title      string         `json:"title,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Counts     map[string]int `json:"counts,omitempty"`
	URL        string         `json:"url,omitempty"`
	SentAt     time.Time      `json:"sent_at"`
}

func (c *WebhookChannel) Send(ctx context.Context, msg Message) error {
	if !c.enabled {
		return nil
	}
	payload := webhookPayload{
		Event:      msg.Event.Type,
		Severity:   msg.Event.Severity,
		Repository: msg.Event.Repository,
		ScanID:     msg.Event.ScanID,
		Title:      SanitizeTitle(msg.Event.Title),
		Summary:    SanitizeSummary(msg.Event.Summary),
		Counts:     msg.Event.Counts,
		URL:        msg.Event.URL,
		SentAt:     time.Now().UTC(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	headers := map[string]string{}
	if c.secret != "" {
		mac := hmac.New(sha256.New, []byte(c.secret))
		if _, err := mac.Write(body); err != nil {
			return fmt.Errorf("sign webhook body: %w", err)
		}
		headers[signatureHeader] = hex.EncodeToString(mac.Sum(nil))
	}
	code, err := c.poster.Post(ctx, c.url, "application/json", body, headers)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("webhook HTTP %d", code)
	}
	return nil
}

// SignWebhookBody computes HMAC-SHA256 hex for tests and verification docs.
func SignWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body) // hash.Hash.Write never fails; signature helper for tests/docs
	return hex.EncodeToString(mac.Sum(nil))
}
