package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func TestFeedbackTargetRejectsUnsafeURLs(t *testing.T) {
	for _, test := range []struct{ url, expected string }{
		{"", ""},
		{"https://git.example.org/owner/repo", "https://git.example.org/owner/repo"},
		{"https://user:password@git.example.org/repo", ""},
		{"http://git.example.org/repo", ""},
		{"https://git.example.org/repo?token=abc", ""},
		{"http://localhost:3000/team/repo", "http://localhost:3000/team/repo"},
	} {
		t.Setenv("REPOSITORY_DETECTIVE_PROJECT_URL", test.url)
		if got := feedbackTarget(); got != test.expected {
			t.Errorf("feedbackTarget(%q) = %q; want %q", test.url, got, test.expected)
		}
	}
}

func TestFeedbackPageIsReadOnlyAndDisclosesSubmissionBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("REPOSITORY_DETECTIVE_PROJECT_URL", "https://git.example.org/team/rd")
	h, err := NewHandler(nil, store.GlobalSettingsSnapshot{}, "/ui", nil, nil, false, "")
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	h.RegisterRoutes(r.Group("/ui"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/feedback", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	for _, s := range []string{"Generated report", "you review and submit", "rd-fb-log-consent", "git.example.org/team/rd", "rd-fb-copy"} {
		if !strings.Contains(w.Body.String(), s) {
			t.Errorf("missing %q", s)
		}
	}
}
