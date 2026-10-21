package ui

import (
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestBuildScannerFailureReportPrefillsIssue(t *testing.T) {
	link := BuildScannerFailureReport(store.ScannerFailureEvent{
		ScannerName:  "semgrep",
		Status:       "timed_out",
		Error:        "context deadline exceeded",
		ScanID:       "abc123def456",
		RepoFullName: "commstech/Repository-Detective",
		StartedAt:    time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
	}, "rc-health-ux", "http://127.0.0.1:8081/ui")

	if !strings.Contains(link.URL, "template=system_health.md") {
		t.Fatalf("expected system_health template, got %s", link.URL)
	}
	if !strings.Contains(link.URL, "title=") {
		t.Fatal("expected title query")
	}
	if !strings.Contains(link.Body, "semgrep") || !strings.Contains(link.Body, "timed_out") {
		t.Fatal("body must include scanner failure details")
	}
	if !strings.Contains(link.Body, "rc-health-ux") {
		t.Fatal("body must include product version")
	}
	if strings.Contains(link.Body, "AKIA") {
		t.Fatal("body must not leak secrets")
	}
}

func TestBuildToolHealthReportForMissingBinary(t *testing.T) {
	link := BuildToolHealthReport(operator.ToolStatus{
		Name:            "trivy",
		EnabledInConfig: true,
		StatusState:     operator.StatusEnabledMissingBinary,
	}, "rc-test", "http://localhost:8081/ui")
	if link.URL == "" || !strings.Contains(link.Title, "trivy") {
		t.Fatalf("unexpected link: %+v", link)
	}
}

func TestToolNeedsHealthReport(t *testing.T) {
	missing := operator.ToolStatus{StatusState: operator.StatusEnabledMissingBinary}
	if !toolNeedsHealthReport(missing) {
		t.Fatal("missing binary should be reportable")
	}
	ok := operator.ToolStatus{
		EnabledInConfig: true, BinaryInstalled: true, Available: true,
		StatusState: operator.StatusEnabledAvailable, Version: "v1",
	}
	if toolNeedsHealthReport(ok) {
		t.Fatal("healthy tool should not be reportable")
	}
}
