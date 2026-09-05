package middleware

import (
	"fmt"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"github.com/gin-gonic/gin"
)

// RedactingAccessLogger returns Gin middleware that logs requests with secrets redacted
// from paths (e.g. ?api_key=), Authorization headers, and API key headers.
func RedactingAccessLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		path := security.RedactAccessLogLine(p.Request.URL.RequestURI())
		return fmt.Sprintf("%s - [%s] \"%s %s %s\" %d %s \"%s\" \"%s\"\n",
			security.RedactAccessLogLine(p.ClientIP),
			p.TimeStamp.Format(time.RFC1123),
			p.Method,
			path,
			p.Request.Proto,
			p.StatusCode,
			p.Latency,
			security.RedactAccessLogLine(p.Request.UserAgent()),
			security.RedactAccessLogLine(p.ErrorMessage),
		)
	})
}
