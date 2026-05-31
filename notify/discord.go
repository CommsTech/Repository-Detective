package notify

import "context"

// DiscordChannel sends messages via Discord webhook.
type DiscordChannel struct {
	enabled bool
	url     string
	poster  HTTPPoster
}

// NewDiscordChannel creates a Discord notifier.
func NewDiscordChannel(cfg Config, poster HTTPPoster) *DiscordChannel {
	if poster == nil {
		poster = newDefaultHTTPPoster()
	}
	return &DiscordChannel{
		enabled: cfg.DiscordEnabled && cfg.DiscordWebhookURL != "",
		url:     cfg.DiscordWebhookURL,
		poster:  poster,
	}
}

func (c *DiscordChannel) Name() string  { return "discord" }
func (c *DiscordChannel) Enabled() bool { return c.enabled }

func (c *DiscordChannel) Send(ctx context.Context, msg Message) error {
	if !c.enabled {
		return nil
	}
	payload := map[string]string{"content": msg.Text}
	return postJSON(ctx, c.poster, c.url, payload, nil)
}
