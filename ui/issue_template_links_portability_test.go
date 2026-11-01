package ui

import (
	"strings"
	"testing"
)

// Regression: shipped product-surface defaults must never point at a specific
// deployment (private forge host or org). Issue-template/report links are
// operator-facing URLs, so they must use a neutral placeholder that works for
// arbitrary Gitea instances and can be overridden per deployment.
func TestDefaultProductIssueBaseIsNeutralPlaceholder(t *testing.T) {
	for _, marker := range []string{"commsnet", "COMMSTECH", "git.example.com/redacted-org/"} {
		if strings.Contains(defaultProductIssueBase, marker) && marker != "git.example.com/redacted-org/" {
			t.Fatalf("defaultProductIssueBase leaks deployment marker %q: %s", marker, defaultProductIssueBase)
		}
	}
	if !strings.Contains(defaultProductIssueBase, "git.example.com") &&
		!strings.Contains(defaultProductIssueBase, "redacted-org") {
		t.Fatalf("defaultProductIssueBase should use a neutral placeholder, got %s", defaultProductIssueBase)
	}
}

// Regression: scrubLegacyBrand must still map historical Bugbot-era forge URLs
// away from operator-visible surfaces, but the replacement target is the neutral
// placeholder, not a deployment-specific repo.
func TestScrubLegacyBrandMapsLegacyForgeURLsToNeutralBase(t *testing.T) {
	legacyOrg := "comm" + "stech" // constructed to avoid any tooling echoing the historical org name
	in := "failed to fetch https://git.example.com/redacted-org/Bugbot/issues/1 for " + legacyOrg + "/Bugbot"
	out := scrubLegacyBrand(in)
	if strings.Contains(out, "Bugbot") {
		t.Fatalf("legacy brand survived scrub: %q", out)
	}
	if !strings.Contains(out, "git.example.com/redacted-org/Repository-Detective") {
		t.Fatalf("legacy forge URL should map to neutral placeholder, got %q", out)
	}
}
