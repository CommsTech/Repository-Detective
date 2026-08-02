package ui

import (
	"crypto/hmac"
	"net/http"
	"strings"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"github.com/gin-gonic/gin"
)

const unlockCSRFSalt = "unlock"

// RegisterUnlockRoutes mounts the API-key unlock page (no auth required).
func (h *Handler) RegisterUnlockRoutes(g *gin.RouterGroup) {
	if h == nil || h.auth.IsLocal() {
		return
	}
	g.GET("/unlock", h.UnlockPage)
	g.POST("/unlock", h.UnlockSubmit)
}

// APIKeyAuthMiddleware requires a valid API key for protected UI routes in api_key_only mode.
// Missing or invalid keys render a friendly HTML unlock page instead of JSON errors.
func (h *Handler) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.auth.IsLocal() {
			c.Next()
			return
		}
		if h.apiKeySecret == "" {
			h.renderUnlock(c, "API key is not configured on this server.", "")
			c.Abort()
			return
		}

		apiKey, fromQuery := h.extractUIAPIKey(c)
		if apiKey == "" {
			h.renderUnlock(c, "", c.Request.URL.RequestURI())
			c.Abort()
			return
		}
		if fromQuery && h.auth.RejectQueryStringAPIKey {
			h.renderUnlock(c, "Query string API keys are disabled. Use the unlock form or X-Repository-Detective-API-Key header.", "")
			c.Abort()
			return
		}

		if !hmac.Equal([]byte(apiKey), []byte(h.apiKeySecret)) {
			h.renderUnlock(c, "Invalid API key.", "")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) extractUIAPIKey(c *gin.Context) (string, bool) {
	if key := c.GetHeader("X-Repository-Detective-API-Key"); key != "" {
		return key, false
	}
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")), false
	}
	if key := apiKeyFromCookie(c); key != "" {
		return key, false
	}
	if key := strings.TrimSpace(c.Query("api_key")); key != "" {
		return key, true
	}
	return "", false
}

func (h *Handler) UnlockPage(c *gin.Context) {
	if h.auth.IsLocal() {
		c.Redirect(http.StatusFound, h.basePath+"/login")
		return
	}
	if key, _ := h.extractUIAPIKey(c); key != "" && hmac.Equal([]byte(key), []byte(h.apiKeySecret)) {
		c.Redirect(http.StatusFound, h.basePath+"/")
		return
	}
	errMsg := ""
	if c.Query("error") == "invalid" {
		errMsg = "Invalid API key."
	}
	next := strings.TrimSpace(c.Query("next"))
	if next == "" {
		next = h.basePath + "/"
	}
	h.renderUnlock(c, errMsg, next)
}

func (h *Handler) UnlockSubmit(c *gin.Context) {
	if h.auth.IsLocal() {
		c.Redirect(http.StatusFound, h.basePath+"/login")
		return
	}
	if h.auth.CSRFEnabled && h.apiKeySecret != "" {
		token := c.PostForm("csrf_token")
		if !security.ValidCSRFToken(h.apiKeySecret, unlockCSRFSalt, token) {
			c.String(http.StatusForbidden, "invalid or missing CSRF token")
			return
		}
	}
	apiKey := strings.TrimSpace(c.PostForm("api_key"))
	next := strings.TrimSpace(c.PostForm("next"))
	if next == "" || !strings.HasPrefix(next, h.basePath) {
		next = h.basePath + "/"
	}
	if apiKey == "" || !hmac.Equal([]byte(apiKey), []byte(h.apiKeySecret)) {
		h.renderUnlock(c, "Invalid API key.", next)
		return
	}
	h.setUIAPIKeyCookie(c, apiKey)
	c.Redirect(http.StatusSeeOther, next)
}

func (h *Handler) setUIAPIKeyCookie(c *gin.Context, key string) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     uiSessionCookieName,
		Value:    key,
		Path:     h.basePath,
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func (h *Handler) renderUnlock(c *gin.Context, errMsg, next string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusUnauthorized)
	pd := pageData{
		Title:         "Unlock dashboard",
		BasePath:      h.basePath,
		CSRFToken:     security.CSRFToken(h.apiKeySecret, unlockCSRFSalt),
		SetupComplete: h.isSetupComplete(c.Request.Context()),
		Data: map[string]any{
			"Error": errMsg,
			"Next":  next,
		},
	}
	if err := h.tmpl.ExecuteTemplate(c.Writer, "unlock.html", pd); err != nil {
		h.logger.Errorf("render unlock: %v", err)
		c.String(http.StatusInternalServerError, "template error")
	}
}
