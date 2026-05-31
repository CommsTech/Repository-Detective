package issues

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/bugbot/ai"
)

func TestBuildLabelsDualMode(t *testing.T) {
	SetLabelCompatMode(LabelCompatDual)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })

	issue := &ai.CodeIssue{
		Severity:   "high",
		Category:   "secret",
		Source:     "gitleaks",
		Confidence: 0.95,
	}
	labels := BuildLabels([]string{"custom"}, issue)

	want := map[string]bool{
		"custom":                            true,
		"bugbot":                            true,
		"repository-detective":              true,
		"automated-review":                  true,
		"bugbot/secret":                     true,
		"repository-detective/secret":       true,
		"severity/high":                     true,
		"bugbot/open":                       true,
		"repository-detective/open":         true,
	}
	for _, label := range labels {
		if !want[label] {
			t.Fatalf("unexpected label %q in %v", label, labels)
		}
		delete(want, label)
	}
	if len(want) > 0 {
		t.Fatalf("missing labels: %v", want)
	}
}

func TestBuildLabelsLegacyOnly(t *testing.T) {
	SetLabelCompatMode(LabelCompatLegacyOnly)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })

	issue := &ai.CodeIssue{Severity: "low", Category: "reliability", Source: "health", Confidence: 0.9}
	labels := BuildLabels(nil, issue)
	for _, label := range labels {
		if label == "repository-detective" || strings.HasPrefix(label, "repository-detective/") {
			t.Fatalf("legacy_only should not write new label %q", label)
		}
	}
}

func TestBuildLabelsNewOnly(t *testing.T) {
	SetLabelCompatMode(LabelCompatNewOnly)
	t.Cleanup(func() { SetLabelCompatMode(LabelCompatDual) })

	issue := &ai.CodeIssue{Severity: "low", Category: "tech_debt", Source: "health", Confidence: 0.9}
	labels := BuildLabels(nil, issue)
	for _, label := range labels {
		if label == "bugbot" || label == "bugbot/open" || strings.HasPrefix(label, "bugbot/") {
			t.Fatalf("new_only should not write legacy label %q", label)
		}
	}
}
