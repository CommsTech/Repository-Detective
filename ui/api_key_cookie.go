package ui

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// uiSessionCookieName is the HttpOnly cookie name for UI API key transport (not a secret).
const uiSessionCookieName = "rd_ui_sess"

// UIAPIKeyCookieMiddleware stores ?api_key= in an HttpOnly cookie and redirects to a clean URL.
// Legacy query-string auth still works for one hop; subsequent requests use the cookie.
func (h *Handler) UIAPIKeyCookieMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.auth.IsLocal() {
			c.Next()
			return
		}
		key := strings.TrimSpace(c.Query("api_key"))
		if key == "" {
			c.Next()
			return
		}
		if h.auth.RejectQueryStringAPIKey {
			c.String(http.StatusBadRequest, "Query string API keys are disabled. Use header X-Repository-Detective-API-Key or Authorization: Bearer.")
			c.Abort()
			return
		}
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
		q := c.Request.URL.Query()
		q.Del("api_key")
		c.Request.URL.RawQuery = q.Encode()
		target := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			target += "?" + c.Request.URL.RawQuery
		}
		c.Redirect(http.StatusSeeOther, target)
		c.Abort()
	}
}

func apiKeyFromCookie(c *gin.Context) string {
	if key, err := c.Cookie(uiSessionCookieName); err == nil {
		return strings.TrimSpace(key)
	}
	return ""
}
