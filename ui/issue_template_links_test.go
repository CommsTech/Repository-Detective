package ui

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildFindingIssueTemplateLinks(t *testing.T) {
	links := BuildFindingIssueTemplateLinks(store.FindingDetail{
		FindingListItem: store.FindingListItem{
			Finding: store.Finding{
				ID:          42,
				Fingerprint: "rd-abc",
				RuleID:      "rule-x",
				Source:      "semgrep",
				Severity:    "medium",
				Confidence:  0.8,
				FilePath:    "src/a.go",
				Line:        10,
			},
			RepoFullName: "owner/repo",
		},
	}, "http://localhost:8080/ui")

	if !strings.Contains(links.FalsePositiveURL, "template=scanner_false_positive") {
		t.Fatalf("expected false positive template URL, got %q", links.FalsePositiveURL)
	}
	if !strings.Contains(links.TemplateGuidance, "rd-abc") {
		t.Fatal("guidance must include fingerprint")
	}
	if strings.Contains(links.TemplateGuidance, "AKIA") {
		t.Fatal("guidance must not include secrets")
	}
}

func TestBuildScanBetaFeedbackLink(t *testing.T) {
	u := BuildScanBetaFeedbackLink("512145e55d4488ea", "commstech/PCAP_Analyser")
	if !strings.Contains(u, "template=beta_feedback") {
		t.Fatalf("expected beta_feedback template, got %q", u)
	}
	if !strings.Contains(u, "512145e55d44") {
		t.Fatal("expected scan id prefix in title")
	}
}
