package ui

import (
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// feedbackTarget is deliberately operator-configured. Never send private
// diagnostics or authenticated operator credentials to an inferred forge URL.
func feedbackTarget() string {
	raw := strings.TrimSpace(os.Getenv("REPOSITORY_DETECTIVE_PROJECT_URL"))
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
		return ""
	}
	if strings.ContainsAny(u.EscapedPath(), "\\\n\r") {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

// Feedback is a safe local draft, NOT a server-side issue-posting endpoint.
// The signed-in reporter reviews and submits in their own forge account.
func (h *Handler) Feedback(c *gin.Context) {
	h.renderNav(c, "feedback.html", "Report a problem", "feedback", map[string]any{
		"ProjectIssueBase": feedbackTarget(),
	})
}
