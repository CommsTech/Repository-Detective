package ui

import (
	"encoding/json"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestActionableFindingViewSections(t *testing.T) {
	meta, _ := json.Marshal(map[string]any{
		"cve": "CVE-2024-0001", "package": "openssl", "version": "3.1.0", "fixed_version": "3.1.4",
	})
	detail := store.FindingDetail{
		FindingListItem: store.FindingListItem{
			Finding: store.Finding{
				ID: 1, Title: "OpenSSL CVE", Severity: "high", Confidence: 0.8,
				Category: "vulnerability", Source: "trivy", RuleID: "CVE-2024-0001",
				FilePath: "Dockerfile", Line: 3, Fingerprint: "fp1",
				FirstSeenAt: time.Now(), LastSeenAt: time.Now(),
			},
			RepoFullName: "org/repo",
		},
		Instances: []store.FindingInstance{{
			EvidenceRedacted: "pkg: openssl@3.1.0",
			RawMetadataJSON:  meta,
		}},
	}
	view := buildActionableFindingView(detail)
	if view.Summary == "" || view.WhyItMatters == "" || view.RecommendedFix == "" {
		t.Fatal("expected populated actionable sections")
	}
	if view.CVEID != "CVE-2024-0001" {
		t.Fatalf("cve: %s", view.CVEID)
	}
	if len(view.VerificationSteps) == 0 {
		t.Fatal("expected verification steps")
	}
}

func TestSecretFindingFlagsRedaction(t *testing.T) {
	detail := store.FindingDetail{
		FindingListItem: store.FindingListItem{
			Finding: store.Finding{Category: "secret", Source: "gitleaks", Title: "key"},
		},
	}
	view := buildActionableFindingView(detail)
	if !view.HasSecretEvidence {
		t.Fatal("expected secret evidence flag")
	}
}
