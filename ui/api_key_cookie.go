package ui

import (
	"net/http"
	"net/url"
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
			// The query value is never accepted as a credential. Older pages
			// put it on SBOM and other links; a signed-in browser still has
			// the session cookie, so drop the parameter and continue.
			if apiKeyFromCookie(c) != "" {
				h.redirectWithoutAPIKey(c)
				return
			}
			next := url.QueryEscape(requestPathWithoutAPIKey(c))
			c.Redirect(http.StatusSeeOther, h.basePath+"/unlock?next="+next)
			c.Abort()
			return
		}
		// gin SetCookie always sets HttpOnly + SameSite; Secure follows public_url
		// (https → true). Avoids gosec G124 false positives on http.Cookie literals
		// while keeping HTTP homelab unlock working.
		secure := h.auth.CookieSecure()
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(uiSessionCookieName, key, 86400*7, h.basePath, "", secure, true)
		h.redirectWithoutAPIKey(c)
	}
}

// redirectWithoutAPIKey sends the browser to the same path with api_key removed.
func (h *Handler) redirectWithoutAPIKey(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, requestPathWithoutAPIKey(c))
	c.Abort()
}

func requestPathWithoutAPIKey(c *gin.Context) string {
	q := c.Request.URL.Query()
	q.Del("api_key")
	target := c.Request.URL.Path
	if enc := q.Encode(); enc != "" {
		target += "?" + enc
	}
	return target
}

func apiKeyFromCookie(c *gin.Context) string {
	if key, err := c.Cookie(uiSessionCookieName); err == nil {
		return strings.TrimSpace(key)
	}
	return ""
}
