package analyzers_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestConfigFromPolicyOverridesScanners(t *testing.T) {
	base := &analyzers.Config{
		AnalysisDepth:     3,
		EnableLLMAuditors: true,
	}
	base.Scanners.EnableTrivy = true
	base.Scanners.EnableSemgrep = true
	base.Workspace.Mode = "api"

	policy := analyzers.ScanPolicy{
		AnalysisDepth:     2,
		EnableTrivy:       false,
		EnableSemgrep:     false,
		EnableGitleaks:    true,
		WorkspaceMode:     "archive",
		AIPolicy:          "disabled",
		EnableLLMAuditors: true,
	}

	cfg := analyzers.ConfigFromPolicy(base, policy, true)
	if cfg.AnalysisDepth != 2 {
		t.Fatalf("depth %d", cfg.AnalysisDepth)
	}
	if cfg.Scanners.EnableTrivy || cfg.Scanners.EnableSemgrep {
		t.Fatal("expected scanners disabled by repo settings")
	}
	if !cfg.Scanners.EnableGitleaks {
		t.Fatal("expected gitleaks enabled by repo settings")
	}
	if cfg.Workspace.NormalizedMode() != "archive" {
		t.Fatalf("workspace %s", cfg.Workspace.Mode)
	}
	if cfg.EnableLLMAuditors {
		t.Fatal("ai_policy disabled should disable LLM auditors")
	}
}

func TestEngineConfigForUsesContext(t *testing.T) {
	engine := analyzers.NewEngine(nil, nil, nil, &analyzers.Config{
		AnalysisDepth: 3,
	}, nil)
	policy := analyzers.ScanPolicy{AnalysisDepth: 1, WorkspaceMode: "api", AIPolicy: "disabled"}
	ctx := analyzers.WithScanPolicy(context.Background(), policy)
	cfg := engine.ConfigFor(ctx)
	if cfg.AnalysisDepth != 1 {
		t.Fatalf("expected context override depth 1, got %d", cfg.AnalysisDepth)
	}
}

func TestLLMEnabledForPolicy(t *testing.T) {
	policy := analyzers.ScanPolicy{AIPolicy: "disabled", EnableLLMAuditors: true, AnalysisDepth: 3}
	if analyzers.LLMEnabledForPolicy(policy, true) {
		t.Fatal("expected disabled")
	}
}

func TestConfigFromPolicyHealthDisabled(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Health: health.DefaultConfig()}
	policy := analyzers.ScanPolicy{
		AnalysisDepth: 2, EnableHealthChecks: false,
		EnableTechDebtChecks: true, EnableReliabilityChecks: true,
	}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Health.Enabled {
		t.Fatal("repo health disabled should skip health checks")
	}
}

func TestConfigFromPolicyReliabilityDisabled(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Health: health.DefaultConfig()}
	policy := analyzers.ScanPolicy{
		AnalysisDepth: 2, EnableHealthChecks: true,
		EnableReliabilityChecks: false,
	}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Health.EnableReliability {
		t.Fatal("reliability should be disabled by repo policy")
	}
}

func TestConfigFromPolicyLargeFileThreshold(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Health: health.DefaultConfig()}
	policy := analyzers.ScanPolicy{
		AnalysisDepth: 2, EnableHealthChecks: true,
		EnableMaintainabilityChecks: true,
		HealthLargeFileLines: 10,
	}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Health.LargeFileLines != 10 {
		t.Fatalf("expected threshold 10, got %d", cfg.Health.LargeFileLines)
	}
	content := strings.Repeat("line\n", 20)
	findings := health.Run(health.RunInput{Files: []health.FileInput{{
		Path: "big.go", Content: content, Language: "go",
	}}}, cfg.Health, nil)
	found := false
	for _, f := range findings {
		if f.RuleID == "HEALTH-LARGE-FILE" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected large file finding with repo threshold override")
	}
}

func TestConfigFromPolicyAIRiskDisabled(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Health: health.DefaultConfig()}
	base.Health.EnableAIRisk = true
	policy := analyzers.ScanPolicy{
		AnalysisDepth: 2, EnableHealthChecks: true, EnableAIRiskChecks: false,
	}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Health.EnableAIRisk {
		t.Fatal("repo AI risk disabled should suppress AI risk checks")
	}
}

func TestConfigFromPolicyGraphDisabled(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Graph: graph.DefaultConfig()}
	policy := analyzers.ScanPolicy{AnalysisDepth: 2, EnableCodeGraph: false}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Graph.Enabled {
		t.Fatal("repo graph disabled should skip code graph")
	}
}

func TestConfigFromPolicyGraphMaxNodesTruncation(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2, Graph: graph.DefaultConfig()}
	policy := analyzers.ScanPolicy{AnalysisDepth: 2, EnableCodeGraph: true, GraphMaxNodes: 120}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if cfg.Graph.MaxNodes != 120 {
		t.Fatalf("expected max nodes 120, got %d", cfg.Graph.MaxNodes)
	}
	files := make([]graph.FileInput, 0, 150)
	for i := 0; i < 150; i++ {
		path := "pkg/file" + strconv.Itoa(i) + ".go"
		files = append(files, graph.FileInput{Path: path, Content: "package pkg\n", Language: "go"})
	}
	g, _ := graph.Build(context.Background(), graph.BuildInput{Files: files}, cfg.Graph, nil)
	if !g.Metrics.Truncated {
		t.Fatal("expected truncated graph with low max nodes override")
	}
}

func TestConfigFromPolicyGoScanners(t *testing.T) {
	base := &analyzers.Config{AnalysisDepth: 2}
	base.Scanners.GoScannerMaxFindings = 100
	policy := analyzers.ScanPolicy{
		AnalysisDepth:             2,
		EnableGovulncheck:         true,
		EnableGosec:               true,
		EnableStaticcheck:         true,
		GovulncheckTimeoutSeconds: 30,
		GoScannerMaxFindings:      50,
	}
	cfg := analyzers.ConfigFromPolicy(base, policy, false)
	if !cfg.Scanners.EnableGovulncheck || !cfg.Scanners.EnableGosec || !cfg.Scanners.EnableStaticcheck {
		t.Fatal("expected Go scanners enabled from policy")
	}
	if cfg.Scanners.GovulncheckTimeoutSeconds != 30 || cfg.Scanners.GoScannerMaxFindings != 50 {
		t.Fatalf("unexpected Go scanner tuning: %+v", cfg.Scanners)
	}
}

func TestScannersConfigFromSnapshot(t *testing.T) {
	base := analyzers.ScannersConfigFromSnapshot(scanners.DefaultConfig(), analyzers.PolicySnapshot{
		EnabledScanners:           []string{"govulncheck", "gosec", "staticcheck"},
		EnableGovulncheck:         true,
		EnableGosec:               true,
		EnableStaticcheck:         true,
		GoScannerMaxFindings:      25,
		GovulncheckTimeoutSeconds: 45,
	})
	if !base.EnableGovulncheck || !base.EnableGosec || !base.EnableStaticcheck {
		t.Fatal("expected Go scanners enabled from snapshot")
	}
	if base.GoScannerMaxFindings != 25 || base.GovulncheckTimeoutSeconds != 45 {
		t.Fatalf("unexpected snapshot merge: %+v", base)
	}
}

func TestScannersConfigFromSnapshotIncludesIAC(t *testing.T) {
	base := analyzers.ScannersConfigFromSnapshot(scanners.DefaultConfig(), analyzers.PolicySnapshot{
		EnabledScanners:       []string{"hadolint", "checkov"},
		EnableHadolint:        true,
		EnableCheckov:         true,
		IACScannerMaxFindings: 40,
		CheckovTimeoutSeconds: 90,
	})
	if !base.EnableHadolint || !base.EnableCheckov {
		t.Fatal("expected IaC scanners enabled from snapshot")
	}
	if base.IACScannerMaxFindings != 40 || base.CheckovTimeoutSeconds != 90 {
		t.Fatalf("unexpected snapshot merge: %+v", base)
	}
}

func TestSnapshotFromPolicyIncludesGoScanners(t *testing.T) {
	snap := analyzers.SnapshotFromPolicy(analyzers.ScanPolicy{
		AnalysisDepth:     2,
		EnableGovulncheck: true,
		EnableGosec:       true,
		EnableStaticcheck: false,
	})
	found := map[string]bool{}
	for _, name := range snap.EnabledScanners {
		found[name] = true
	}
	if !found["govulncheck"] || !found["gosec"] || found["staticcheck"] {
		t.Fatalf("unexpected enabled scanners: %v", snap.EnabledScanners)
	}
	if !snap.EnableGovulncheck || !snap.EnableGosec || snap.EnableStaticcheck {
		t.Fatalf("unexpected snapshot flags: %+v", snap)
	}
}
