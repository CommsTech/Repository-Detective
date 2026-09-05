package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestResolveEffectiveSettingsInheritsGlobal(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.WorkspaceMode = "api"
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.WorkspaceMode != "api" {
		t.Fatalf("expected global workspace mode, got %s", effective.WorkspaceMode)
	}
}

func TestResolveEffectiveSettingsRepoOverride(t *testing.T) {
	global := store.DefaultGlobalSettings()
	archive := "archive"
	depth := 2
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{
		WorkspaceMode: &archive,
		AnalysisDepth: &depth,
	})
	if effective.WorkspaceMode != "archive" || effective.AnalysisDepth != 2 {
		t.Fatalf("unexpected override: %+v", effective)
	}
}

func TestSanitizeInvalidPolicyLevel(t *testing.T) {
	global := store.DefaultGlobalSettings()
	effective := store.SanitizeEffectiveSettings(store.EffectiveSettings{
		PolicyLevel: "not_valid",
	}, global)
	if effective.PolicyLevel != global.PolicyLevel {
		t.Fatalf("expected fallback policy level")
	}
}

func TestLLMEnabledForSettings(t *testing.T) {
	e := store.EffectiveSettings{
		AIPolicy:          store.AIPolicyAllowed,
		EnableLLMAuditors: true,
		AnalysisDepth:     3,
	}
	if !store.LLMEnabledForSettings(e, true) {
		t.Fatal("expected LLM enabled")
	}
	e.AIPolicy = store.AIPolicyDisabled
	if store.LLMEnabledForSettings(e, true) {
		t.Fatal("expected LLM disabled when ai_policy=disabled")
	}
}

func TestShouldCreateForgeIssues(t *testing.T) {
	e := store.EffectiveSettings{Enabled: true, PolicyLevel: store.PolicyIssueOnly, IssuePolicy: store.IssuePolicyAll}
	if !store.ShouldCreateForgeIssues(e) {
		t.Fatal("expected forge issues")
	}
	e.PolicyLevel = store.PolicyMonitorOnly
	if store.ShouldCreateForgeIssues(e) {
		t.Fatal("monitor_only should not create issues")
	}
	e.PolicyLevel = store.PolicyIssueOnly
	e.IssuePolicy = store.IssuePolicyOff
	if store.ShouldCreateForgeIssues(e) {
		t.Fatal("issue_policy off should not create issues")
	}
}

func TestApplyReportOnlyDryRunSettings(t *testing.T) {
	e := store.EffectiveSettings{
		Enabled: true, PolicyLevel: store.PolicyIssueOnly, IssuePolicy: store.IssuePolicyAll,
		AIPolicy: store.AIPolicyAllowed, EnableLLMAuditors: true,
	}
	store.ApplyReportOnlyDryRunSettings(&e)
	if store.ShouldCreateForgeIssues(e) {
		t.Fatal("report-only dry run must not create forge issues")
	}
	if e.AIPolicy != store.AIPolicyDisabled || e.EnableLLMAuditors {
		t.Fatal("report-only dry run must disable AI")
	}
}

func TestPassesIssueGates(t *testing.T) {
	e := store.EffectiveSettings{SeverityGate: "high", ConfidenceGate: 0.8}
	if store.PassesIssueGates("low", 0.9, e) {
		t.Fatal("low severity should not pass high gate")
	}
	if store.PassesIssueGates("high", 0.5, e) {
		t.Fatal("low confidence should not pass gate")
	}
	if !store.PassesIssueGates("high", 0.9, e) {
		t.Fatal("expected gate pass")
	}
}

func TestShouldFailCommitStatus(t *testing.T) {
	if !store.ShouldFailCommitStatus(store.EffectiveSettings{PolicyLevel: store.PolicyGatePR}) {
		t.Fatal("gate_pr should fail status")
	}
	if store.ShouldFailCommitStatus(store.EffectiveSettings{PolicyLevel: store.PolicyIssueOnly}) {
		t.Fatal("issue_only should not fail status")
	}
}

func TestEnabledScannersList(t *testing.T) {
	list := store.EnabledScannersList(store.EffectiveSettings{EnableTrivy: true, EnableGitleaks: true})
	if len(list) != 2 {
		t.Fatalf("expected 2 scanners, got %v", list)
	}
}

func TestSeveritiesForStatus(t *testing.T) {
	e := store.EffectiveSettings{PolicyLevel: store.PolicyGatePR, SeverityGate: "high", ConfidenceGate: 0.5}
	out := store.SeveritiesForStatus([]string{"high", "low"}, []float64{0.9, 0.9}, e)
	if len(out) != 1 || out[0] != "high" {
		t.Fatalf("unexpected severities: %v", out)
	}
}
