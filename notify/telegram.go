package notify

import (
	"context"
	"fmt"
)

// TelegramChannel sends messages via Telegram Bot API.
type TelegramChannel struct {
	enabled bool
	token   string
	chatID  string
	poster  HTTPPoster
}

// NewTelegramChannel creates a Telegram notifier.
func NewTelegramChannel(cfg Config, poster HTTPPoster) *TelegramChannel {
	if poster == nil {
		poster = newDefaultHTTPPoster()
	}
	return &TelegramChannel{
		enabled: cfg.TelegramEnabled && cfg.TelegramBotToken != "" && cfg.TelegramChatID != "",
		token:   cfg.TelegramBotToken,
		chatID:  cfg.TelegramChatID,
		poster:  poster,
	}
}

func (c *TelegramChannel) Name() string  { return "telegram" }
func (c *TelegramChannel) Enabled() bool { return c.enabled }

func (c *TelegramChannel) Send(ctx context.Context, msg Message) error {
	if !c.enabled {
		return nil
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)
	payload := map[string]string{
		"chat_id": c.chatID,
		"text":    msg.Text,
	}
	return postJSON(ctx, c.poster, url, payload, nil)
}
