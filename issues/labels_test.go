package issues

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestBuildLabelsUsesRepositoryDetectiveNamespace(t *testing.T) {
	issue := &ai.CodeIssue{
		Severity:   "high",
		Category:   "secret",
		Source:     "gitleaks",
		Confidence: 0.95,
	}
	labels := BuildLabels([]string{"custom"}, issue)

	want := map[string]bool{
		"custom":                      true,
		"repository-detective":        true,
		"automated-review":            true,
		"repository-detective/secret": true,
		"severity/high":               true,
		"repository-detective/open":   true,
	}
	for _, label := range labels {
		if strings.HasPrefix(label, "bugbot") {
			t.Fatalf("must not write legacy bugbot labels, got %q", label)
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
	if len(labels) != 1 || labels[0] != "repository-detective" {
		t.Fatalf("unexpected lookup labels: %v", labels)
	}
}
