package store_test

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/store"
)

func TestValidateScanProfile(t *testing.T) {
	if err := store.ValidateScanProfile("fast"); err != nil {
		t.Fatalf("expected valid profile: %v", err)
	}
	if err := store.ValidateScanProfile("not_a_profile"); err == nil {
		t.Fatal("expected invalid profile error")
	}
}

func TestFastProfileDisablesHeavyChecks(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileFast
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.EnableSemgrep || effective.EnableGrype || effective.EnableCodeGraph {
		t.Fatalf("fast profile should disable heavy checks: %+v", effective)
	}
	if !effective.EnableGitleaks || !effective.EnableTrivy {
		t.Fatal("fast profile should enable gitleaks and trivy")
	}
	if effective.AnalysisDepth != 2 || effective.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("unexpected fast profile depth/ai: depth=%d ai=%s", effective.AnalysisDepth, effective.AIPolicy)
	}
}

func TestStandardDeterministicEnablesScanners(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandardDeterministic
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if !effective.EnableTrivy || !effective.EnableGrype || !effective.EnableGitleaks || !effective.EnableSemgrep {
		t.Fatal("standard_deterministic should enable core security scanners")
	}
	if !effective.EnableGovulncheck || !effective.EnableHadolint || !effective.EnableCheckov {
		t.Fatal("standard_deterministic should enable Go and IaC scanners")
	}
	if !effective.EnableCodeGraph || !effective.EnableHealthChecks {
		t.Fatal("standard_deterministic should enable graph and health")
	}
}

func TestStrictSecurityGates(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStrictSecurity
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.SeverityGate != "medium" || effective.ConfidenceGate != 0.85 {
		t.Fatalf("unexpected gates: severity=%s confidence=%f", effective.SeverityGate, effective.ConfidenceGate)
	}
	if effective.PolicyLevel != store.PolicyGatePR {
		t.Fatalf("expected gate_pr, got %s", effective.PolicyLevel)
	}
}

func TestMaintainerDeepEnablesDeepChecks(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileMaintainerDeep
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.WorkspaceMode != "auto" || !effective.EnableTestGapChecks || !effective.EnablePerformanceChecks {
		t.Fatalf("maintainer_deep missing deep checks: %+v", effective)
	}
	if !effective.GraphIncludeFunctions {
		t.Fatal("maintainer_deep should include graph functions")
	}
	if !effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyAllowed || effective.AnalysisDepth < 3 {
		t.Fatalf("maintainer_deep should enable full LLM pipeline: llm=%v ai=%s depth=%d",
			effective.EnableLLMAuditors, effective.AIPolicy, effective.AnalysisDepth)
	}
}

func TestGlobalLLMConfigOverridesDeterministicProfile(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandardDeterministic
	global.EnableLLMAuditors = true
	global.AnalysisDepth = 3
	global.AIPolicy = store.AIPolicyAllowed
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if !effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyAllowed || effective.AnalysisDepth != 3 {
		t.Fatalf("global LLM config should override deterministic profile defaults: llm=%v ai=%s depth=%d",
			effective.EnableLLMAuditors, effective.AIPolicy, effective.AnalysisDepth)
	}
}

func TestPreinstallCautiousDisablesIssuesAndAI(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfilePreinstallCautious
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.IssuePolicy != store.IssuePolicyOff || effective.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("preinstall_cautious should disable issues and AI: issue=%s ai=%s", effective.IssuePolicy, effective.AIPolicy)
	}
}

func TestGlobalProfileApplied(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileFast
	_, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{})
	if meta.ScanProfile != store.ScanProfileFast || meta.ProfileSource != "global" {
		t.Fatalf("expected global fast profile, got %+v", meta)
	}
}

func TestRepoProfileApplied(t *testing.T) {
	global := store.DefaultGlobalSettings()
	profile := store.ScanProfileStrictSecurity
	effective, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{ScanProfile: &profile})
	if meta.ProfileSource != "repo" || meta.ScanProfile != store.ScanProfileStrictSecurity {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if effective.PolicyLevel != store.PolicyGatePR {
		t.Fatal("repo strict profile should apply")
	}
}

func TestRepoCustomUsesExplicitOverrides(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileFast
	custom := store.ScanProfileCustom
	off := false
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{
		ScanProfile: &custom,
		EnableSemgrep: &off,
	})
	if effective.EnableSemgrep {
		t.Fatal("custom repo should honor explicit override")
	}
}

func TestExplicitRepoOverrideWins(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandardDeterministic
	on := true
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{EnableGrype: &on})
	if !effective.EnableGrype {
		t.Fatal("explicit repo override should win")
	}
	_, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{EnableGrype: &on})
	if !meta.ProfileModified {
		t.Fatal("expected profile_modified when repo override present")
	}
}

func TestApplySettingsUpdateProfileOnlyClearsOverrides(t *testing.T) {
	profile := store.ScanProfileFast
	existing := store.RepoSettings{
		RepositoryID: 1,
		EnableSemgrep: boolPtr(true),
	}
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, store.SettingsUpdate{ScanProfile: &profile})
	if merged.ScanProfile == nil || *merged.ScanProfile != store.ScanProfileFast {
		t.Fatal("expected fast profile stored")
	}
	if merged.EnableSemgrep != nil {
		t.Fatal("profile-only update should clear explicit overrides")
	}
}

func TestApplySettingsUpdateAdvancedSwitchesCustom(t *testing.T) {
	profile := store.ScanProfileFast
	off := false
	merged := store.ApplySettingsUpdateWithProfilePolicy(store.RepoSettings{RepositoryID: 1}, store.SettingsUpdate{
		ScanProfile:   &profile,
		EnableSemgrep: &off,
	})
	if merged.ScanProfile == nil || *merged.ScanProfile != store.ScanProfileCustom {
		t.Fatalf("expected custom profile, got %v", merged.ScanProfile)
	}
}

func TestDBDisabledInheritsGlobal(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileCustom
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.EnableTrivy != global.EnableTrivy {
		t.Fatal("custom profile with no repo settings should match global config")
	}
}

func boolPtr(v bool) *bool { return &v }
