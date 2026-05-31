package notify

import "context"

// SlackChannel sends messages via Slack incoming webhook.
type SlackChannel struct {
	enabled bool
	url     string
	poster  HTTPPoster
}

// NewSlackChannel creates a Slack notifier.
func NewSlackChannel(cfg Config, poster HTTPPoster) *SlackChannel {
	if poster == nil {
		poster = newDefaultHTTPPoster()
	}
	return &SlackChannel{
		enabled: cfg.SlackEnabled && cfg.SlackWebhookURL != "",
		url:     cfg.SlackWebhookURL,
		poster:  poster,
	}
}

func (c *SlackChannel) Name() string  { return "slack" }
func (c *SlackChannel) Enabled() bool { return c.enabled }

func (c *SlackChannel) Send(ctx context.Context, msg Message) error {
	if !c.enabled {
		return nil
	}
	payload := map[string]string{"text": msg.Text}
	return postJSON(ctx, c.poster, c.url, payload, nil)
}
