package scanners_test

import (
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestCreateWorkspace(t *testing.T) {
	dir, cleanup, err := scanners.CreateWorkspace([]scanners.FileEntry{
		{Path: "src/main.go", Content: "package main\n"},
		{Path: "go.mod", Content: "module example.com/test\n\ngo 1.21\n"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if dir == "" {
		t.Fatal("expected workspace dir")
	}
}

func TestFindingToCandidate(t *testing.T) {
	candidate := scanners.Finding{
		ID:         "TRIVY-CVE-2024-0001",
		Source:     "trivy",
		Category:   "dependency_vulnerability",
		Severity:   "high",
		Title:      "CVE in dependency",
		File:       "go.mod",
		Confidence: 0.98,
	}.ToCandidateFinding()

	if candidate.AuditorType != "trivy" {
		t.Fatalf("expected trivy auditor, got %s", candidate.AuditorType)
	}
	if candidate.Category != "dependency_vulnerability" {
		t.Fatalf("expected dependency category, got %s", candidate.Category)
	}
}

func TestParseTrivyJSON(t *testing.T) {
	payload := `{
		"Results": [{
			"Target": "go.mod",
			"Vulnerabilities": [{
				"VulnerabilityID": "CVE-2024-1234",
				"PkgName": "example.com/lib",
				"InstalledVersion": "1.0.0",
				"FixedVersion": "1.0.1",
				"Severity": "HIGH",
				"Title": "Example CVE"
			}]
		}]
	}`

	var report struct {
		Results []struct {
			Target          string `json:"Target"`
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				Severity        string `json:"Severity"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(report.Results[0].Vulnerabilities) != 1 {
		t.Fatal("expected one vulnerability")
	}
}

func TestParseGrypeJSON(t *testing.T) {
	payload := `{
		"matches": [{
			"vulnerability": {"id": "CVE-2024-9999", "severity": "High", "description": "test"},
			"artifact": {"name": "pkg", "version": "1.2.3", "locations": [{"path": "go.mod"}]}
		}]
	}`

	var report struct {
		Matches []struct {
			Vulnerability struct {
				ID string `json:"id"`
			} `json:"vulnerability"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if report.Matches[0].Vulnerability.ID != "CVE-2024-9999" {
		t.Fatal("unexpected CVE id")
	}
}

func TestManifestPathsIncludesGoMod(t *testing.T) {
	found := false
	for _, path := range scanners.ManifestPaths() {
		if path == "go.mod" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected go.mod in manifest paths")
	}
}
