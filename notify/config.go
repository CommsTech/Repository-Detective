package notify

import (
	"sort"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// Config holds global notification configuration (credentials stay here only).
type Config struct {
	Enabled           bool
	MinSeverity       string
	CooldownSeconds   int
	PublicURL         string
	TelegramEnabled   bool
	TelegramBotToken  string
	TelegramChatID    string
	SlackEnabled      bool
	SlackWebhookURL   string
	DiscordEnabled    bool
	DiscordWebhookURL string
	WebhookEnabled    bool
	WebhookURL        string
	WebhookSecret     string
	DefaultEvents     []string
}

// EffectiveSettings is resolved notification policy for a repository.
type EffectiveSettings struct {
	Enabled         bool
	MinSeverity     string
	CooldownSeconds int
	Events          map[string]bool
}

// DefaultConfig returns disabled notification config.
func DefaultConfig() Config {
	return Config{
		MinSeverity:     "high",
		CooldownSeconds: 300,
		DefaultEvents:   append([]string(nil), DefaultEnabledEvents...),
	}
}

// ResolveEffective merges global notification config with optional repo overrides.
func ResolveEffective(global Config, repo store.RepoSettings) EffectiveSettings {
	enabled := global.Enabled
	if repo.NotificationsEnabled != nil {
		enabled = *repo.NotificationsEnabled
	}

	minSev := global.MinSeverity
	if repo.NotificationMinSeverity != nil && strings.TrimSpace(*repo.NotificationMinSeverity) != "" {
		minSev = strings.ToLower(strings.TrimSpace(*repo.NotificationMinSeverity))
	}

	cooldown := global.CooldownSeconds
	if repo.NotificationCooldownSeconds != nil && *repo.NotificationCooldownSeconds >= 0 {
		cooldown = *repo.NotificationCooldownSeconds
	}

	events := parseEventSet(global.DefaultEvents)
	if repo.NotificationEvents != nil && strings.TrimSpace(*repo.NotificationEvents) != "" {
		events = parseEventCSV(*repo.NotificationEvents)
	}

	return EffectiveSettings{
		Enabled:         enabled,
		MinSeverity:     minSev,
		CooldownSeconds: cooldown,
		Events:          events,
	}
}

func parseEventSet(list []string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, e := range list {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" {
			out[e] = true
		}
	}
	return out
}

func parseEventCSV(raw string) map[string]bool {
	parts := strings.Split(raw, ",")
	out := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = true
		}
	}
	return out
}

// IsValidEventType reports whether name is a known event type.
func IsValidEventType(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, e := range AllEvents {
		if e == name {
			return true
		}
	}
	return name == EventTest
}

// GlobalNotificationConfigFromMain builds notify.Config from application config fields.
func GlobalNotificationConfigFromMain(
	enabled bool,
	minSeverity string,
	cooldown int,
	publicURL string,
	telegramEnabled bool, telegramToken, telegramChat string,
	slackEnabled bool, slackURL string,
	discordEnabled bool, discordURL string,
	webhookEnabled bool, webhookURL, webhookSecret string,
) Config {
	cfg := DefaultConfig()
	cfg.Enabled = enabled
	if minSeverity != "" {
		cfg.MinSeverity = strings.ToLower(minSeverity)
	}
	if cooldown > 0 {
		cfg.CooldownSeconds = cooldown
	}
	cfg.PublicURL = publicURL
	cfg.TelegramEnabled = telegramEnabled
	cfg.TelegramBotToken = telegramToken
	cfg.TelegramChatID = telegramChat
	cfg.SlackEnabled = slackEnabled
	cfg.SlackWebhookURL = slackURL
	cfg.DiscordEnabled = discordEnabled
	cfg.DiscordWebhookURL = discordURL
	cfg.WebhookEnabled = webhookEnabled
	cfg.WebhookURL = webhookURL
	cfg.WebhookSecret = webhookSecret
	return cfg
}

// ChannelsConfigured returns channel names that have credentials and are enabled.
func ChannelsConfigured(cfg Config) []string {
	var out []string
	if cfg.TelegramEnabled && cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		out = append(out, "telegram")
	}
	if cfg.SlackEnabled && cfg.SlackWebhookURL != "" {
		out = append(out, "slack")
	}
	if cfg.DiscordEnabled && cfg.DiscordWebhookURL != "" {
		out = append(out, "discord")
	}
	if cfg.WebhookEnabled && cfg.WebhookURL != "" {
		out = append(out, "webhook")
	}
	return out
}

// EventList returns sorted enabled event type names.
func EventList(events map[string]bool) []string {
	if len(events) == 0 {
		return nil
	}
	out := make([]string, 0, len(events))
	for name, enabled := range events {
		if enabled {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
