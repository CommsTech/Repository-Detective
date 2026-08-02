package profile_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestRuffStyleDowngradedHomelab(t *testing.T) {
	p := profile.RepoProfile{Layout: profile.LayoutInfrastructure, FileCount: 40}
	sev, conf := profile.RuffSeverity("I001", "medium", p)
	if sev != "info" || conf > 0.5 {
		t.Fatalf("style rule got %s conf=%v", sev, conf)
	}
}

func TestRuffSecurityNotDowngradedHomelab(t *testing.T) {
	p := profile.RepoProfile{Layout: profile.LayoutInfrastructure, FileCount: 40}
	sev, conf := profile.RuffSeverity("S105", "high", p)
	if sev != "high" || conf < 0.8 {
		t.Fatalf("security rule got %s conf=%v", sev, conf)
	}
}

func TestRuffCalibrationRepoScoped(t *testing.T) {
	homelab := profile.RepoProfile{Layout: profile.LayoutInfrastructure, FileCount: 40}
	regular := profile.RepoProfile{Layout: profile.LayoutSingleApp, FileCount: 200, PrimaryEcosystem: profile.EcosystemPython}
	results := []scanners.RunResult{{
		Scanner: "ruff",
		Findings: []scanners.Finding{{
			Reference: "I001", Severity: "medium", Confidence: 0.9,
		}},
	}}
	outHomelab := profile.CalibrateRuffResults(results, homelab)
	if outHomelab[0].Findings[0].Severity != "info" {
		t.Fatalf("homelab expected info got %s", outHomelab[0].Findings[0].Severity)
	}
	outRegular := profile.CalibrateRuffResults(results, regular)
	if outRegular[0].Findings[0].Severity != "medium" {
		t.Fatalf("regular repo should stay medium got %s", outRegular[0].Findings[0].Severity)
	}
}
