package api

import (
	"net/http"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/gin-gonic/gin"
)

// AIStatusProvider exposes AI status and manual connection tests.
type AIStatusProvider interface {
	Status(c *gin.Context) ai.ProviderStatus
	TestConnection(c *gin.Context, force bool) (ai.ProviderStatus, error)
}

// AIHandler serves AI status and manual test routes.
type AIHandler struct {
	provider AIStatusProvider
}

// NewAIHandler creates an AI status handler.
func NewAIHandler(p AIStatusProvider) *AIHandler {
	return &AIHandler{provider: p}
}

// RegisterRoutes mounts AI routes.
func (h *AIHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/ai/status", h.Status)
	g.POST("/ai/test-connection", h.TestConnection)
}

func (h *AIHandler) Status(c *gin.Context) {
	if h.provider == nil {
		c.JSON(http.StatusOK, ai.ProviderStatus{Configured: false, TestMode: ai.TestModeMetadataOnly})
		return
	}
	c.JSON(http.StatusOK, h.provider.Status(c))
}

func (h *AIHandler) TestConnection(c *gin.Context) {
	if h.provider == nil {
		// AI is optional — report disabled posture, not a broken install.
		c.JSON(http.StatusOK, gin.H{
			"configured":      false,
			"policy_disabled": true,
			"ai_analysis":     "Disabled",
			"message":         "AI Analysis: Disabled — deterministic scanners do not require an AI provider",
		})
		return
	}
	force := c.Query("force") == "true"
	st, err := h.provider.TestConnection(c, force)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"status": st, "error": err.Error(), "warning": "This test may incur API cost when using chat_completion mode."})
		return
	}
	c.JSON(http.StatusOK, st)
}
