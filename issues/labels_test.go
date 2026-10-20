package issues

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestBuildLabelsUsesScopedTaxonomy(t *testing.T) {
	issue := &ai.CodeIssue{
		Severity:      "high",
		Category:      "secret",
		Source:        "gitleaks",
		Confidence:    0.95,
		Fixable:       "true",
		SafeForAutoPR: false,
	}
	labels := BuildLabels([]string{"custom"}, issue)

	want := map[string]bool{
		"custom":                      true,
		"source/repository-detective": true,
		"category/secret":             true,
		"severity/high":               true,
		"scanner/gitleaks":            true,
		"triage/needs-review":         true,
		"remediation/manual":          true,
	}
	for _, label := range labels {
		if strings.HasPrefix(label, "bugbot") {
			t.Fatalf("must not write legacy bugbot labels, got %q", label)
		}
		if label == "automated-review" || label == "repository-detective/open" {
			t.Fatalf("redundant label still present: %q", label)
		}
		if !want[label] {
			t.Fatalf("unexpected label %q in %v", label, labels)
		}
		delete(want, label)
	}
	if len(want) > 0 {
		t.Fatalf("missing labels: %v", want)
	}
}

func TestIssueLookupBaseLabels(t *testing.T) {
	labels := IssueLookupBaseLabels()
	if len(labels) < 1 || labels[0] != "source/repository-detective" {
		t.Fatalf("unexpected lookup labels: %v", labels)
	}
}
