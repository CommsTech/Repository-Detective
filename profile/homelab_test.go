package profile

import "testing"

func TestIsHomelabInfra(t *testing.T) {
	if !IsHomelabInfra(RepoProfile{
		Layout:    LayoutInfrastructure,
		FileCount: 10,
	}) {
		t.Fatal("infrastructure layout should match")
	}
	if !IsHomelabInfra(RepoProfile{
		FileCount:        40,
		PrimaryEcosystem: EcosystemPython,
		Manifests:        []string{"docker-compose.yml"},
	}) {
		t.Fatal("small python repo with compose should match")
	}
	if !IsHomelabInfra(RepoProfile{
		FileCount:        28,
		PrimaryEcosystem: EcosystemPython,
		Manifests:        []string{"requirements.txt"},
	}) {
		t.Fatal("small python repo with requirements.txt should match homelab infra")
	}
	if IsHomelabInfra(RepoProfile{
		FileCount:        500,
		PrimaryEcosystem: EcosystemGo,
		Layout:           LayoutSingleApp,
	}) {
		t.Fatal("large go app should not match homelab infra")
	}
}

func TestHomelabInternalIPDowngraded(t *testing.T) {
	p := RepoProfile{FileCount: 30, Manifests: []string{"docker-compose.yml"}}
	sev, conf := HomelabInfraSeverity("REL-INTERNAL-INFRA-REF", "medium", 0.75, "config.yaml", "host: 192.168.1.10", p)
	if sev != "info" || conf >= 0.75 {
		t.Fatalf("expected downgrade, got %s conf=%v", sev, conf)
	}
}

func TestHomelabInternalIPWithCredentialNotDowngraded(t *testing.T) {
	p := RepoProfile{FileCount: 30, Manifests: []string{"docker-compose.yml"}}
	sev, conf := HomelabInfraSeverity("REL-INTERNAL-INFRA-REF", "medium", 0.75, "config.yaml", "password=192.168.1.10", p)
	if sev != "medium" || conf != 0.75 {
		t.Fatalf("credential context should not downgrade: %s conf=%v", sev, conf)
	}
}

func TestHomelabEvalNotDowngraded(t *testing.T) {
	p := RepoProfile{FileCount: 30, Manifests: []string{"docker-compose.yml"}}
	sev, conf := HomelabInfraSeverity("SEC-EVAL", "critical", 0.95, "app.py", "eval(user_input)", p)
	if sev != "critical" || conf != 0.95 {
		t.Fatalf("SEC-EVAL must not be downgraded: %s conf=%v", sev, conf)
	}
}
