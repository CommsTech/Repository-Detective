package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression: shipped UI templates are embedded and served to operators of any
// Gitea instance. They must not contain deployment-specific forge hosts or orgs;
// docs links use neutral placeholders.
func TestEmbeddedTemplatesUseNeutralDocLinks(t *testing.T) {
	entries, err := filepath.Glob("templates/*.html")
	if err != nil {
		t.Fatalf("glob templates: %v", err)
	}
	if len(entries) == 0 {
		t.Skip("no embedded templates found on disk in this working copy")
	}
	for _, path := range entries {
		blob, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(blob)
		lowered := strings.ToLower(content)
		for _, bad := range []string{"commsnet", "commstech"} {
			if strings.Contains(lowered, bad) {
				t.Fatalf("%s contains deployment-specific marker %q", path, bad)
			}
		}
	}
}

// Regression: compose defaults must be buildable/runnable without a private
// registry: default image should be a local name (overridable via RD_IMAGE),
// never a hardcoded private registry path.
func TestComposeDefaultImageIsNeutral(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join("..", "docker-compose.yml"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("docker-compose.yml not found relative to ui package")
		}
		t.Fatalf("read compose: %v", err)
	}
	content := string(blob)
	lowered := strings.ToLower(content)
	for _, bad := range []string{"commsnet", "commstech"} {
		if strings.Contains(lowered, bad) {
			t.Fatalf("docker-compose.yml leaks deployment-specific registry/org marker %q", bad)
		}
	}
	if !strings.Contains(content, "RD_IMAGE:-repository-detective:all-in-one") {
		t.Fatalf("compose default image expected to be local neutral name, got snippet: %s", content)
	}
}
