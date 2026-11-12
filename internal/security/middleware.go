package security

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DefaultMaxRequestBody is the maximum JSON/form body size for API writes.
const DefaultMaxRequestBody = 1 << 20 // 1 MiB

// MiddlewareHeaders adds baseline HTTP security headers for UI and API responses.
func MiddlewareHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'none'")
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/ui") || strings.HasPrefix(path, "/api/v1") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

// MiddlewareMaxBody limits request body size to mitigate DoS via large payloads.
func MiddlewareMaxBody(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxRequestBody
	}
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
