package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager routes sanitized events to configured channels.
type Manager struct {
	cfg      Config
	channels []Channel
	limiter  *RateLimiter
	resolve  SettingsResolver
	logger   *logrus.Logger
}

// NewManager creates a notification manager. Pass nil logger for a no-op logger.
func NewManager(cfg Config, resolve SettingsResolver, logger *logrus.Logger, poster HTTPPoster) *Manager {
	if logger == nil {
		logger = logrus.New()
	}
	if resolve == nil {
		resolve = func(int64) EffectiveSettings { return EffectiveSettings{Enabled: cfg.Enabled} }
	}
	channels := []Channel{
		NewTelegramChannel(cfg, poster),
		NewSlackChannel(cfg, poster),
		NewDiscordChannel(cfg, poster),
		NewWebhookChannel(cfg, poster),
	}
	return &Manager{
		cfg:      cfg,
		channels: channels,
		limiter:  NewRateLimiter(),
		resolve:  resolve,
		logger:   logger,
	}
}

// Enabled reports whether any notification path is active.
func (m *Manager) Enabled() bool {
	if m == nil || !m.cfg.Enabled {
		return false
	}
	for _, ch := range m.channels {
		if ch.Enabled() {
			return true
		}
	}
	return false
}

// SetEnabled toggles the global notifications master switch without rebuild.
func (m *Manager) SetEnabled(enabled bool) {
	if m != nil {
		m.cfg.Enabled = enabled
	}
}

// Config returns a redacted copy safe for API display.
func (m *Manager) Config() Config {
	if m == nil {
		return DefaultConfig()
	}
	c := m.cfg
	c.TelegramBotToken = redactSecret(c.TelegramBotToken)
	c.SlackWebhookURL = redactURL(c.SlackWebhookURL)
	c.DiscordWebhookURL = redactURL(c.DiscordWebhookURL)
	c.WebhookURL = redactURL(c.WebhookURL)
	c.WebhookSecret = redactSecret(c.WebhookSecret)
	return c
}

func redactSecret(s string) string {
	if s == "" {
		return ""
	}
	return "configured"
}

func redactURL(s string) string {
	if s == "" {
		return ""
	}
	return "configured"
}

// Emit sends a notification when policy allows.
func (m *Manager) Emit(ctx context.Context, repositoryID int64, ev Event) {
	if m == nil || !m.cfg.Enabled {
		return
	}
	settings := m.resolve(repositoryID)
	if !settings.Enabled {
		return
	}
	ev.Type = strings.ToLower(strings.TrimSpace(ev.Type))
	if ev.Type == "" {
		return
	}
	if !settings.Events[ev.Type] && ev.Type != EventTest {
		return
	}
	if !PassesSeverityThreshold(ev.Severity, settings.MinSeverity) {
		return
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	if ev.Repository != "" {
		ev.Repository = SafeRepoLabel(ev.Repository)
	}
	ev.Title = SanitizeTitle(ev.Title)
	ev.Summary = SanitizeSummary(ev.Summary)
	if ev.URL == "" && m.cfg.PublicURL != "" && ev.ScanID != "" {
		ev.URL = strings.TrimRight(m.cfg.PublicURL, "/") + "/ui/scans/" + ev.ScanID
	}

	key := cooldownKey(ev.Repository, ev.Type)
	if !m.limiter.Allow(key, time.Duration(settings.CooldownSeconds)*time.Second) {
		m.logger.Debugf("notification cooldown active for %s", key)
		return
	}

	msg := FormatMessage(ev)
	for _, ch := range m.channels {
		if !ch.Enabled() {
			continue
		}
		out := msg
		out.Channel = ch.Name()
		if err := ch.Send(ctx, out); err != nil {
			m.logger.Warnf("notification %s via %s failed: %v", ev.Type, ch.Name(), err)
		}
	}
}

// SendTest emits a safe test notification to all enabled channels.
func (m *Manager) SendTest(ctx context.Context) error {
	if m == nil || !m.Enabled() {
		return fmt.Errorf("notifications disabled or no channels configured")
	}
	ev := Event{
		Type:       EventTest,
		Severity:   "info",
		Repository: "test/repo",
		Title:      "Repository Detective test notification",
		Summary:    "This is a safe test message. No secrets or evidence are included.",
		CreatedAt:  time.Now().UTC(),
	}
	msg := FormatMessage(ev)
	var lastErr error
	sent := false
	for _, ch := range m.channels {
		if !ch.Enabled() {
			continue
		}
		out := msg
		out.Channel = ch.Name()
		if err := ch.Send(ctx, out); err != nil {
			lastErr = err
			m.logger.Warnf("test notification via %s failed: %v", ch.Name(), err)
			continue
		}
		sent = true
	}
	if !sent && lastErr != nil {
		return lastErr
	}
	if !sent {
		return fmt.Errorf("no notification channels enabled")
	}
	return nil
}

// ChannelsConfigured returns enabled channel names (for UI).
func (m *Manager) ChannelsConfigured() []string {
	if m == nil {
		return nil
	}
	var out []string
	for _, ch := range m.channels {
		if ch.Enabled() {
			out = append(out, ch.Name())
		}
	}
	return out
}
