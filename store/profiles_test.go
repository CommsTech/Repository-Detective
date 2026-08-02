package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestValidateScanProfile(t *testing.T) {
	for _, name := range []string{"light", "standard", "deep", "custom", "fast", "beta_standard", "maintainer_deep"} {
		if err := store.ValidateScanProfile(name); err != nil {
			t.Fatalf("expected valid profile %q: %v", name, err)
		}
	}
	if err := store.ValidateScanProfile("not_a_profile"); err == nil {
		t.Fatal("expected invalid profile error")
	}
}

func TestNormalizeScanProfileAliases(t *testing.T) {
	cases := map[string]string{
		"fast":                   store.ScanProfileLight,
		"preinstall_cautious":    store.ScanProfileLight,
		"beta_standard":          store.ScanProfileStandard,
		"standard_deterministic": store.ScanProfileStandard,
		"strict_security":        store.ScanProfileStandard,
		"maintainer_deep":        store.ScanProfileDeep,
		"Light":                  store.ScanProfileLight,
		"STANDARD":               store.ScanProfileStandard,
	}
	for in, want := range cases {
		if got := store.NormalizeScanProfile(in); got != want {
			t.Fatalf("NormalizeScanProfile(%q)=%q want %q", in, got, want)
		}
	}
}

func TestLightProfileReadOnlyFast(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileLight
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.EnableSemgrep || effective.EnableGrype || effective.EnableCodeGraph {
		t.Fatalf("light profile should disable heavy checks: %+v", effective)
	}
	if !effective.EnableGitleaks || !effective.EnableTrivy {
		t.Fatal("light profile should enable gitleaks and trivy")
	}
	if effective.IssuePolicy != store.IssuePolicyOff || effective.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("light should be read-only / no AI: issue=%s ai=%s", effective.IssuePolicy, effective.AIPolicy)
	}
}

func TestLegacyFastAliasMapsToLight(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileFast
	_, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{})
	if meta.ScanProfile != store.ScanProfileLight {
		t.Fatalf("fast should canonicalize to light, got %q", meta.ScanProfile)
	}
}

func TestStandardProfileFilesIssues(t *testing.T) {
	effective := store.ProfileDefaults(store.ScanProfileStandard)
	if effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("standard should be deterministic-only: llm=%v ai=%s", effective.EnableLLMAuditors, effective.AIPolicy)
	}
	if effective.IssuePolicy != store.IssuePolicyAll {
		t.Fatalf("standard should file issues, got %s", effective.IssuePolicy)
	}
	if !effective.EnableTrivy || !effective.EnableSemgrep || !effective.EnableCodeGraph {
		t.Fatal("standard should enable full deterministic scanners + graph")
	}
	if effective.SeverityGate != "high" || effective.ConfidenceGate != 0.85 {
		t.Fatalf("unexpected standard gates: severity=%s confidence=%f", effective.SeverityGate, effective.ConfidenceGate)
	}
}

func TestDeepProfileEnablesAI(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileDeep
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.WorkspaceMode != "auto" || !effective.EnableTestGapChecks || !effective.EnablePerformanceChecks {
		t.Fatalf("deep missing heavy checks: %+v", effective)
	}
	if !effective.GraphIncludeFunctions {
		t.Fatal("deep should include graph functions")
	}
	if !effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyAllowed || effective.AnalysisDepth < 3 {
		t.Fatalf("deep should enable AI cross-checks: llm=%v ai=%s depth=%d",
			effective.EnableLLMAuditors, effective.AIPolicy, effective.AnalysisDepth)
	}
}

func TestGlobalLLMConfigOverridesDeterministicProfile(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandard
	global.EnableLLMAuditors = true
	global.AnalysisDepth = 3
	global.AIPolicy = store.AIPolicyAllowed
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if !effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyAllowed || effective.AnalysisDepth != 3 {
		t.Fatalf("global LLM config should override standard profile defaults: llm=%v ai=%s depth=%d",
			effective.EnableLLMAuditors, effective.AIPolicy, effective.AnalysisDepth)
	}
}

func TestLightBlocksGlobalLLMOverride(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileLight
	global.EnableLLMAuditors = true
	global.AIPolicy = store.AIPolicyAllowed
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{})
	if effective.EnableLLMAuditors || effective.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("light should stay AI-off even if global llm enabled: llm=%v ai=%s",
			effective.EnableLLMAuditors, effective.AIPolicy)
	}
}

func TestGlobalProfileApplied(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileLight
	_, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{})
	if meta.ScanProfile != store.ScanProfileLight || meta.ProfileSource != "global" {
		t.Fatalf("expected global light profile, got %+v", meta)
	}
}

func TestRepoProfileApplied(t *testing.T) {
	global := store.DefaultGlobalSettings()
	profile := store.ScanProfileDeep
	effective, meta := store.ResolveEffectiveSettingsFull(global, store.RepoSettings{ScanProfile: &profile})
	if meta.ProfileSource != "repo" || meta.ScanProfile != store.ScanProfileDeep {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if !effective.EnableLLMAuditors {
		t.Fatal("repo deep profile should apply AI")
	}
}

func TestRepoCustomUsesExplicitOverrides(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileLight
	custom := store.ScanProfileCustom
	off := false
	effective := store.ResolveEffectiveSettings(global, store.RepoSettings{
		ScanProfile:   &custom,
		EnableSemgrep: &off,
	})
	if effective.EnableSemgrep {
		t.Fatal("custom repo should honor explicit override")
	}
}

func TestExplicitRepoOverrideWins(t *testing.T) {
	global := store.DefaultGlobalSettings()
	global.ScanProfile = store.ScanProfileStandard
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
	profile := store.ScanProfileLight
	existing := store.RepoSettings{
		RepositoryID:  1,
		EnableSemgrep: boolPtr(true),
	}
	merged := store.ApplySettingsUpdateWithProfilePolicy(existing, store.SettingsUpdate{ScanProfile: &profile})
	if merged.ScanProfile == nil || *merged.ScanProfile != store.ScanProfileLight {
		t.Fatal("expected light profile stored")
	}
	if merged.EnableSemgrep != nil {
		t.Fatal("profile-only update should clear explicit overrides")
	}
}

func TestApplySettingsUpdateLegacyAliasCanonicalized(t *testing.T) {
	legacy := store.ScanProfileBetaStandard
	merged := store.ApplySettingsUpdateWithProfilePolicy(store.RepoSettings{RepositoryID: 1}, store.SettingsUpdate{
		ScanProfile: &legacy,
	})
	if merged.ScanProfile == nil || *merged.ScanProfile != store.ScanProfileStandard {
		t.Fatalf("expected beta_standard saved as standard, got %v", merged.ScanProfile)
	}
}

func TestApplySettingsUpdateAdvancedSwitchesCustom(t *testing.T) {
	profile := store.ScanProfileLight
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

func TestScanProfileLabel(t *testing.T) {
	if store.ScanProfileLabel("beta_standard") != "Standard" {
		t.Fatal(store.ScanProfileLabel("beta_standard"))
	}
	if store.ScanProfileLabel("deep") != "Deep" {
		t.Fatal(store.ScanProfileLabel("deep"))
	}
}

func boolPtr(v bool) *bool { return &v }
