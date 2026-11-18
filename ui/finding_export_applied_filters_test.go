package ui_test

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingExportLinksHonorEffectiveFilters(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "export-links.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	r, _ := testUI(t, s)
	path := "/ui/findings?focus=1&status=resolved&show_suppressed=1&page=3&source=trivy&q=" + url.QueryEscape("CVE 2026")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Inspect the actual rendered href; this catches html/template turning the
	// entire encoded filter query into a single URL-query value.
	for _, format := range []string{"csv", "json"} {
		re := regexp.MustCompile(`href="([^"]*findings/export[^"]*)"`)
		matches := re.FindAllStringSubmatch(body, -1)
		found := false
		for _, match := range matches {
			parsed, err := url.Parse(html.UnescapeString(match[1]))
			if err != nil || parsed.Query().Get("format") != format {
				continue
			}
			found = true
			q := parsed.Query()
			for key, want := range map[string]string{"format": format, "focus": "1", "status": "open", "source": "trivy", "q": "CVE 2026"} {
				if q.Get(key) != want {
					t.Errorf("export %s: %s=%q want %q; url=%s", format, key, q.Get(key), want, parsed.String())
				}
			}
			if q.Get("page") != "" || q.Get("show_suppressed") != "" || q.Get("severity") != "" {
				t.Errorf("stale filters leaked to %s export: %s", format, parsed.String())
			}
		}
		if !found {
			t.Fatalf("%s export link not found in page; first 200 bytes: %s", format, strings.TrimSpace(body[:min(200, len(body))]))
		}
	}
}
