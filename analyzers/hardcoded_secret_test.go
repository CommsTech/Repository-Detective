package analyzers

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/profile"
)

func TestAssessHardcodedSecretPlaceholderSkipped(t *testing.T) {
	a := assessHardcodedSecret("wifi_collector.py", `password = "Decryption failed"`)
	if !a.Skip {
		t.Fatalf("expected placeholder skip, got %+v", a)
	}
}

func TestAssessHardcodedSecretTrueSecretHigh(t *testing.T) {
	// Runtime-built sample keeps the analyzer under test while avoiding gitleaks on this file.
	line := `api_key := "` + "AKI" + `AIOSFODNN7EXAMPLE"`
	a := assessHardcodedSecret("config.go", line)
	if a.Severity != "high" || a.Confidence < 0.85 {
		t.Fatalf("expected high confidence secret, got %+v", a)
	}
}

func TestAssessHardcodedSecretExamplePathLow(t *testing.T) {
	a := assessHardcodedSecret("examples/config.py", `password = "super-secret-token-12345"`)
	if a.Severity != "low" {
		t.Fatalf("expected low in example path, got %+v", a)
	}
}

func TestRunStaticAnalysisSkipsDecryptionFailedPlaceholder(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "wifi_collector.py",
		Content: `password = "Decryption failed"`,
	}}, true, false)
	for _, f := range findings {
		if f.ID == "SEC-HARDCODED-SECRET" {
			t.Fatalf("expected placeholder to be skipped, got %+v", f)
		}
	}
}

func TestRunStaticAnalysisFindsHighEntropySecret(t *testing.T) {
	// Build a high-entropy token at runtime. Avoid Stripe/AWS-shaped literals in
	// source so forge secret scanning does not block public mirrors.
	secret := "rd_test_" + "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "config.go",
		Content: `api_key := "` + secret + `"`,
	}}, true, false)
	found := false
	for _, f := range findings {
		if f.ID == "SEC-HARDCODED-SECRET" {
			found = true
			if f.Severity != "high" {
				t.Fatalf("expected high severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected SEC-HARDCODED-SECRET finding")
	}
}

func TestRunStaticAnalysisSkipsInstallScriptExampleCIDR(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "install.py",
		Content: `f.write("# 192.168.1.0/24\n")`,
	}}, true, false)
	for _, f := range findings {
		if f.ID == "REL-INTERNAL-INFRA-REF" {
			t.Fatalf("expected example CIDR write to be skipped, got %+v", f)
		}
	}
}

func TestRunStaticAnalysisDowngradesHomelabInfraRef(t *testing.T) {
	p := profileFromManifests(28, []string{"requirements.txt"}, "python")
	findings := RunStaticAnalysisWithProfile([]FileContent{{
		Path:    "config.yaml",
		Content: "host: 192.168.1.10\n",
	}}, true, false, p)
	for _, f := range findings {
		if f.ID == "REL-INTERNAL-INFRA-REF" && f.Severity != "info" {
			t.Fatalf("expected info downgrade in homelab repo, got %s", f.Severity)
		}
	}
}

func profileFromManifests(fileCount int, manifests []string, eco string) profile.RepoProfile {
	return profile.RepoProfile{
		FileCount:        fileCount,
		Manifests:        manifests,
		PrimaryEcosystem: profile.EcosystemPython,
	}
}
