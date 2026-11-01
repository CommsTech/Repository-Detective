package preinstall

import (
	"strings"
	"testing"
)

// Regression: the pre-install audit DefaultConfig project link is included in
// operator-facing pre-install reports. It must be a neutral placeholder so a
// fresh install never advertises a deployment it is not part of.
func TestDefaultConfigProjectURLIsNeutralPlaceholder(t *testing.T) {
	got := DefaultConfig().RepositoryDetectiveProjectURL
	if got == "" {
		t.Fatal("project url default must not be empty")
	}
	for _, bad := range []string{"commsnet", "commstech"} {
		if strings.Contains(strings.ToLower(got), bad) {
			t.Fatalf("default project url leaks deployment marker %q: %s", bad, got)
		}
	}
	if !strings.Contains(got, "example.com") && !strings.Contains(got, "redacted-org") {
		t.Fatalf("expected neutral placeholder default, got %s", got)
	}
}
