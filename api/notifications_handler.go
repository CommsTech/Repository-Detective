package api

import (
	"net/http"

	"git.commsnet.org/commstech/repository-detective/notify"
	"github.com/gin-gonic/gin"
)

// NotificationHandler serves notification test routes.
type NotificationHandler struct {
	manager *notify.Manager
}

// NewNotificationHandler creates a notification API handler.
func NewNotificationHandler(m *notify.Manager) *NotificationHandler {
	return &NotificationHandler{manager: m}
}

// RegisterRoutes mounts notification routes on the authenticated API group.
func (h *NotificationHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/notifications/test", h.TestNotification)
	g.GET("/notifications/status", h.NotificationStatus)
}

// TestNotification sends a safe test message to configured channels.
func (h *NotificationHandler) TestNotification(c *gin.Context) {
	if h.manager == nil || !h.manager.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "notifications disabled or not configured"})
		return
	}
	if err := h.manager.SendTest(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

// NotificationStatus returns redacted global notification configuration.
func (h *NotificationHandler) NotificationStatus(c *gin.Context) {
	if h.manager == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "channels": []string{}})
		return
	}
	cfg := h.manager.Config()
	c.JSON(http.StatusOK, gin.H{
		"enabled":          cfg.Enabled,
		"min_severity":     cfg.MinSeverity,
		"cooldown_seconds": cfg.CooldownSeconds,
		"channels":         h.manager.ChannelsConfigured(),
		"telegram_enabled": cfg.TelegramEnabled,
		"slack_enabled":    cfg.SlackEnabled,
		"discord_enabled":  cfg.DiscordEnabled,
		"webhook_enabled":  cfg.WebhookEnabled,
	})
}
