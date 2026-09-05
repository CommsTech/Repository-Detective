package store

// ResolveEffectiveSettingsWithMeta merges global config, global/repo profile defaults, and repo overrides.
//
// Resolution order:
//  1. global config snapshot
//  2. global profile defaults when repo scan_profile is NULL and global profile is not custom
//  3. repo profile defaults when repo scan_profile is set and not custom
//  4. repo explicit overrides (non-null stored fields)
func ResolveEffectiveSettingsWithMeta(global GlobalSettingsSnapshot, repoSettings RepoSettings) (EffectiveSettings, EffectiveSettingsMeta) {
	merged := effectiveFromGlobalSnapshot(global)

	if repoSettings.ScanProfile == nil {
		globalProfile := normalizedGlobalProfile(global.ScanProfile)
		if globalProfile != ScanProfileCustom {
			merged = mergeEffectiveSettings(merged, ProfileDefaults(globalProfile))
		}
	}

	if repoSettings.ScanProfile != nil {
		repoProfile := NormalizeScanProfile(*repoSettings.ScanProfile)
		if repoProfile != "" && repoProfile != ScanProfileCustom {
			merged = mergeEffectiveSettings(merged, ProfileDefaults(repoProfile))
		}
	}

	merged = preserveGlobalAIPreferences(merged, global)
	effective := applyRepoOverrides(merged, repoSettings)
	meta := buildSettingsMeta(global, repoSettings, effective)
	effective.ScanProfile = meta.ScanProfile
	return effective, meta
}

// preserveGlobalAIPreferences keeps explicit global AI settings when config enables LLM auditors.
// Scan profiles may default to deterministic-only, but enable_llm_auditors + analysis_depth in
// config.yaml should still activate the full CAH pipeline for homelab and dogfood deployments.
func preserveGlobalAIPreferences(merged EffectiveSettings, global GlobalSettingsSnapshot) EffectiveSettings {
	raw := effectiveFromGlobalSnapshot(global)
	if !raw.EnableLLMAuditors {
		return merged
	}
	switch normalizedGlobalProfile(global.ScanProfile) {
	case ScanProfileLight:
		return merged
	}
	merged.EnableLLMAuditors = true
	if raw.AIPolicy != "" {
		merged.AIPolicy = raw.AIPolicy
	}
	if raw.AnalysisDepth >= 3 {
		merged.AnalysisDepth = raw.AnalysisDepth
	}
	if raw.EnableAIRiskChecks {
		merged.EnableAIRiskChecks = true
	}
	return merged
}

// MergeConfigOverProfile applies explicit config values over profile defaults.
func MergeConfigOverProfile(profileBase, configOverlay EffectiveSettings) EffectiveSettings {
	return mergeEffectiveSettings(profileBase, configOverlay)
}

// mergeEffectiveSettings overlays profile/repo-profile fields onto a base effective settings.
func mergeEffectiveSettings(base, overlay EffectiveSettings) EffectiveSettings {
	if overlay.PolicyLevel != "" {
		base.PolicyLevel = overlay.PolicyLevel
	}
	if overlay.WorkspaceMode != "" {
		base.WorkspaceMode = overlay.WorkspaceMode
	}
	if overlay.AnalysisDepth != 0 {
		base.AnalysisDepth = overlay.AnalysisDepth
	}
	base.EnableLLMAuditors = overlay.EnableLLMAuditors
	base.EnableTrivy = overlay.EnableTrivy
	base.EnableGrype = overlay.EnableGrype
	base.EnableGitleaks = overlay.EnableGitleaks
	base.EnableSemgrep = overlay.EnableSemgrep
	base.EnableGovulncheck = overlay.EnableGovulncheck
	base.EnableGosec = overlay.EnableGosec
	base.EnableStaticcheck = overlay.EnableStaticcheck
	base.EnableHadolint = overlay.EnableHadolint
	base.EnableCheckov = overlay.EnableCheckov
	base.EnableLinters = overlay.EnableLinters
	if overlay.SeverityGate != "" {
		base.SeverityGate = overlay.SeverityGate
	}
	if overlay.ConfidenceGate != 0 {
		base.ConfidenceGate = overlay.ConfidenceGate
	}
	if overlay.IssuePolicy != "" {
		base.IssuePolicy = overlay.IssuePolicy
	}
	if overlay.RemediationPolicy != "" {
		base.RemediationPolicy = overlay.RemediationPolicy
	}
	if overlay.RunnerPolicy != "" {
		base.RunnerPolicy = overlay.RunnerPolicy
	}
	if overlay.AIPolicy != "" {
		base.AIPolicy = overlay.AIPolicy
	}
	base.EnableHealthChecks = overlay.EnableHealthChecks
	base.EnableTechDebtChecks = overlay.EnableTechDebtChecks
	base.EnableReliabilityChecks = overlay.EnableReliabilityChecks
	base.EnableMaintainabilityChecks = overlay.EnableMaintainabilityChecks
	base.EnableTestGapChecks = overlay.EnableTestGapChecks
	base.EnablePerformanceChecks = overlay.EnablePerformanceChecks
	base.EnableAIRiskChecks = overlay.EnableAIRiskChecks
	base.EnableCodeGraph = overlay.EnableCodeGraph
	base.GraphIncludeFunctions = overlay.GraphIncludeFunctions
	base.GraphIncludeFindings = overlay.GraphIncludeFindings
	return base
}

// ApplyRepoOverridesToEffective applies stored per-repo overrides onto resolved settings.
func ApplyRepoOverridesToEffective(base EffectiveSettings, repo RepoSettings) EffectiveSettings {
	return applyRepoOverrides(base, repo)
}

func applyRepoOverrides(base EffectiveSettings, repo RepoSettings) EffectiveSettings {
	return EffectiveSettings{
		Enabled:                     derefBool(repo.Enabled, base.Enabled),
		PolicyLevel:                 derefString(repo.PolicyLevel, base.PolicyLevel),
		WorkspaceMode:               derefString(repo.WorkspaceMode, base.WorkspaceMode),
		AnalysisDepth:               derefInt(repo.AnalysisDepth, base.AnalysisDepth),
		EnableLLMAuditors:           derefBool(repo.EnableLLMAuditors, base.EnableLLMAuditors),
		EnableTrivy:                 derefBool(repo.EnableTrivy, base.EnableTrivy),
		EnableGrype:                 derefBool(repo.EnableGrype, base.EnableGrype),
		EnableGitleaks:              derefBool(repo.EnableGitleaks, base.EnableGitleaks),
		EnableSemgrep:               derefBool(repo.EnableSemgrep, base.EnableSemgrep),
		EnableGovulncheck:           derefBool(repo.EnableGovulncheck, base.EnableGovulncheck),
		EnableGosec:                 derefBool(repo.EnableGosec, base.EnableGosec),
		EnableStaticcheck:           derefBool(repo.EnableStaticcheck, base.EnableStaticcheck),
		EnableHadolint:              derefBool(repo.EnableHadolint, base.EnableHadolint),
		EnableCheckov:               derefBool(repo.EnableCheckov, base.EnableCheckov),
		EnableLinters:               derefBool(repo.EnableLinters, base.EnableLinters),
		SeverityGate:                derefString(repo.SeverityGate, base.SeverityGate),
		ConfidenceGate:              derefFloat(repo.ConfidenceGate, base.ConfidenceGate),
		IssuePolicy:                 derefString(repo.IssuePolicy, base.IssuePolicy),
		RemediationPolicy:           derefString(repo.RemediationPolicy, base.RemediationPolicy),
		RunnerPolicy:                derefString(repo.RunnerPolicy, base.RunnerPolicy),
		ScheduleEnabled:             derefBool(repo.ScheduleEnabled, base.ScheduleEnabled),
		ScheduleCron:                derefString(repo.ScheduleCron, base.ScheduleCron),
		AIPolicy:                    derefString(repo.AIPolicy, base.AIPolicy),
		EnableHealthChecks:          derefBool(repo.EnableHealthChecks, base.EnableHealthChecks),
		EnableTechDebtChecks:        derefBool(repo.EnableTechDebtChecks, base.EnableTechDebtChecks),
		EnableReliabilityChecks:     derefBool(repo.EnableReliabilityChecks, base.EnableReliabilityChecks),
		EnableMaintainabilityChecks: derefBool(repo.EnableMaintainabilityChecks, base.EnableMaintainabilityChecks),
		EnableTestGapChecks:         derefBool(repo.EnableTestGapChecks, base.EnableTestGapChecks),
		EnablePerformanceChecks:     derefBool(repo.EnablePerformanceChecks, base.EnablePerformanceChecks),
		EnableAIRiskChecks:          derefBool(repo.EnableAIRiskChecks, base.EnableAIRiskChecks),
		HealthMaxFindings:           derefInt(repo.HealthMaxFindings, base.HealthMaxFindings),
		HealthLargeFileLines:        derefInt(repo.HealthLargeFileLines, base.HealthLargeFileLines),
		HealthLargeFunctionLines:    derefInt(repo.HealthLargeFunctionLines, base.HealthLargeFunctionLines),
		HealthMaxNestingDepth:       derefInt(repo.HealthMaxNestingDepth, base.HealthMaxNestingDepth),
		HealthMaxFunctionParams:     derefInt(repo.HealthMaxFunctionParams, base.HealthMaxFunctionParams),
		EnableCodeGraph:             derefBool(repo.EnableCodeGraph, base.EnableCodeGraph),
		GraphMaxNodes:               derefInt(repo.GraphMaxNodes, base.GraphMaxNodes),
		GraphMaxEdges:               derefInt(repo.GraphMaxEdges, base.GraphMaxEdges),
		GraphTimeoutSeconds:         derefInt(repo.GraphTimeoutSeconds, base.GraphTimeoutSeconds),
		GraphIncludeFunctions:       derefBool(repo.GraphIncludeFunctions, base.GraphIncludeFunctions),
		GraphIncludeFindings:        derefBool(repo.GraphIncludeFindings, base.GraphIncludeFindings),
		GovulncheckTimeoutSeconds:   derefInt(repo.GovulncheckTimeoutSeconds, base.GovulncheckTimeoutSeconds),
		GosecTimeoutSeconds:         derefInt(repo.GosecTimeoutSeconds, base.GosecTimeoutSeconds),
		StaticcheckTimeoutSeconds:   derefInt(repo.StaticcheckTimeoutSeconds, base.StaticcheckTimeoutSeconds),
		GoScannerMaxFindings:        derefInt(repo.GoScannerMaxFindings, base.GoScannerMaxFindings),
		HadolintTimeoutSeconds:      derefInt(repo.HadolintTimeoutSeconds, base.HadolintTimeoutSeconds),
		CheckovTimeoutSeconds:       derefInt(repo.CheckovTimeoutSeconds, base.CheckovTimeoutSeconds),
		IACScannerMaxFindings:       derefInt(repo.IACScannerMaxFindings, base.IACScannerMaxFindings),
	}
}

// GlobalSnapshotFromEffective converts resolved effective settings to a global snapshot.
func GlobalSnapshotFromEffective(e EffectiveSettings) GlobalSettingsSnapshot {
	return GlobalSettingsSnapshot{
		Enabled:                     e.Enabled,
		PolicyLevel:                 e.PolicyLevel,
		WorkspaceMode:               e.WorkspaceMode,
		AnalysisDepth:               e.AnalysisDepth,
		EnableLLMAuditors:           e.EnableLLMAuditors,
		EnableTrivy:                 e.EnableTrivy,
		EnableGrype:                 e.EnableGrype,
		EnableGitleaks:              e.EnableGitleaks,
		EnableSemgrep:               e.EnableSemgrep,
		EnableGovulncheck:           e.EnableGovulncheck,
		EnableGosec:                 e.EnableGosec,
		EnableStaticcheck:           e.EnableStaticcheck,
		EnableHadolint:              e.EnableHadolint,
		EnableCheckov:               e.EnableCheckov,
		EnableLinters:               e.EnableLinters,
		SeverityGate:                e.SeverityGate,
		ConfidenceGate:              e.ConfidenceGate,
		IssuePolicy:                 e.IssuePolicy,
		RemediationPolicy:           e.RemediationPolicy,
		RunnerPolicy:                e.RunnerPolicy,
		ScheduleEnabled:             e.ScheduleEnabled,
		ScheduleCron:                e.ScheduleCron,
		AIPolicy:                    e.AIPolicy,
		EnableHealthChecks:          e.EnableHealthChecks,
		EnableTechDebtChecks:        e.EnableTechDebtChecks,
		EnableReliabilityChecks:     e.EnableReliabilityChecks,
		EnableMaintainabilityChecks: e.EnableMaintainabilityChecks,
		EnableTestGapChecks:         e.EnableTestGapChecks,
		EnablePerformanceChecks:     e.EnablePerformanceChecks,
		EnableAIRiskChecks:          e.EnableAIRiskChecks,
		HealthMaxFindings:           e.HealthMaxFindings,
		HealthLargeFileLines:        e.HealthLargeFileLines,
		HealthLargeFunctionLines:    e.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       e.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     e.HealthMaxFunctionParams,
		EnableCodeGraph:             e.EnableCodeGraph,
		GraphMaxNodes:               e.GraphMaxNodes,
		GraphMaxEdges:               e.GraphMaxEdges,
		GraphTimeoutSeconds:         e.GraphTimeoutSeconds,
		GraphIncludeFunctions:       e.GraphIncludeFunctions,
		GraphIncludeFindings:        e.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   e.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         e.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   e.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        e.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      e.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       e.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       e.IACScannerMaxFindings,
	}
}

func buildSettingsMeta(global GlobalSettingsSnapshot, repo RepoSettings, effective EffectiveSettings) EffectiveSettingsMeta {
	profileName, source := effectiveProfileName(global, repo)
	modified := RepoHasExplicitOverrides(repo)

	return EffectiveSettingsMeta{
		ScanProfile:             profileName,
		ProfileModified:         modified,
		ProfileSource:           source,
		EffectiveProfileSummary: BuildEffectiveProfileSummary(effective),
	}
}

func effectiveProfileName(global GlobalSettingsSnapshot, repo RepoSettings) (string, string) {
	if repo.ScanProfile != nil {
		p := NormalizeScanProfile(*repo.ScanProfile)
		if p == "" {
			p = ScanProfileCustom
		}
		return p, "repo"
	}
	gp := normalizedGlobalProfile(global.ScanProfile)
	if gp != ScanProfileCustom {
		return gp, "global"
	}
	return ScanProfileCustom, "default"
}

func normalizedGlobalProfile(profile string) string {
	p := NormalizeScanProfile(profile)
	if p == "" {
		return ScanProfileCustom
	}
	return p
}

// RepoHasExplicitOverrides reports whether the repo has stored non-profile overrides.
func RepoHasExplicitOverrides(repo RepoSettings) bool {
	return repo.Enabled != nil ||
		repo.PolicyLevel != nil ||
		repo.WorkspaceMode != nil ||
		repo.AnalysisDepth != nil ||
		repo.EnableLLMAuditors != nil ||
		repo.EnableTrivy != nil ||
		repo.EnableGrype != nil ||
		repo.EnableGitleaks != nil ||
		repo.EnableSemgrep != nil ||
		repo.EnableGovulncheck != nil ||
		repo.EnableGosec != nil ||
		repo.EnableStaticcheck != nil ||
		repo.EnableHadolint != nil ||
		repo.EnableCheckov != nil ||
		repo.EnableLinters != nil ||
		repo.SeverityGate != nil ||
		repo.ConfidenceGate != nil ||
		repo.IssuePolicy != nil ||
		repo.RemediationPolicy != nil ||
		repo.RunnerPolicy != nil ||
		repo.ScheduleEnabled != nil ||
		repo.ScheduleCron != nil ||
		repo.AIPolicy != nil ||
		repo.EnableHealthChecks != nil ||
		repo.EnableTechDebtChecks != nil ||
		repo.EnableReliabilityChecks != nil ||
		repo.EnableMaintainabilityChecks != nil ||
		repo.EnableTestGapChecks != nil ||
		repo.EnablePerformanceChecks != nil ||
		repo.EnableAIRiskChecks != nil ||
		repo.HealthMaxFindings != nil ||
		repo.HealthLargeFileLines != nil ||
		repo.HealthLargeFunctionLines != nil ||
		repo.HealthMaxNestingDepth != nil ||
		repo.HealthMaxFunctionParams != nil ||
		repo.EnableCodeGraph != nil ||
		repo.GraphMaxNodes != nil ||
		repo.GraphMaxEdges != nil ||
		repo.GraphTimeoutSeconds != nil ||
		repo.GraphIncludeFunctions != nil ||
		repo.GraphIncludeFindings != nil ||
		repo.GovulncheckTimeoutSeconds != nil ||
		repo.GosecTimeoutSeconds != nil ||
		repo.StaticcheckTimeoutSeconds != nil ||
		repo.GoScannerMaxFindings != nil ||
		repo.HadolintTimeoutSeconds != nil ||
		repo.CheckovTimeoutSeconds != nil ||
		repo.IACScannerMaxFindings != nil
}

// SettingsUpdateHasAdvancedFields reports whether an update touches non-profile fields.
func SettingsUpdateHasAdvancedFields(u SettingsUpdate) bool {
	return u.Enabled != nil ||
		u.PolicyLevel != nil ||
		u.WorkspaceMode != nil ||
		u.AnalysisDepth != nil ||
		u.EnableLLMAuditors != nil ||
		u.EnableTrivy != nil ||
		u.EnableGrype != nil ||
		u.EnableGitleaks != nil ||
		u.EnableSemgrep != nil ||
		u.EnableGovulncheck != nil ||
		u.EnableGosec != nil ||
		u.EnableStaticcheck != nil ||
		u.EnableHadolint != nil ||
		u.EnableCheckov != nil ||
		u.EnableLinters != nil ||
		u.SeverityGate != nil ||
		u.ConfidenceGate != nil ||
		u.IssuePolicy != nil ||
		u.RemediationPolicy != nil ||
		u.RunnerPolicy != nil ||
		u.ScheduleEnabled != nil ||
		u.ScheduleCron != nil ||
		u.AIPolicy != nil ||
		u.EnableHealthChecks != nil ||
		u.EnableTechDebtChecks != nil ||
		u.EnableReliabilityChecks != nil ||
		u.EnableMaintainabilityChecks != nil ||
		u.EnableTestGapChecks != nil ||
		u.EnablePerformanceChecks != nil ||
		u.EnableAIRiskChecks != nil ||
		u.HealthMaxFindings != nil ||
		u.HealthLargeFileLines != nil ||
		u.HealthLargeFunctionLines != nil ||
		u.HealthMaxNestingDepth != nil ||
		u.HealthMaxFunctionParams != nil ||
		u.EnableCodeGraph != nil ||
		u.GraphMaxNodes != nil ||
		u.GraphMaxEdges != nil ||
		u.GraphTimeoutSeconds != nil ||
		u.GraphIncludeFunctions != nil ||
		u.GraphIncludeFindings != nil ||
		u.GovulncheckTimeoutSeconds != nil ||
		u.GosecTimeoutSeconds != nil ||
		u.StaticcheckTimeoutSeconds != nil ||
		u.GoScannerMaxFindings != nil ||
		u.HadolintTimeoutSeconds != nil ||
		u.CheckovTimeoutSeconds != nil ||
		u.IACScannerMaxFindings != nil
}

// ApplySettingsUpdateWithProfilePolicy merges updates and applies profile selection rules.
func ApplySettingsUpdateWithProfilePolicy(existing RepoSettings, u SettingsUpdate) RepoSettings {
	if u.ScanProfile != nil {
		if err := ValidateScanProfile(*u.ScanProfile); err == nil {
			profile := NormalizeScanProfile(*u.ScanProfile)
			u.ScanProfile = &profile
			if profile != ScanProfileCustom && !SettingsUpdateHasAdvancedFields(u) {
				return RepoSettings{
					RepositoryID: existing.RepositoryID,
					ScanProfile:  &profile,
					UpdatedAt:    existing.UpdatedAt,
				}
			}
		}
	}
	if SettingsUpdateHasAdvancedFields(u) {
		custom := ScanProfileCustom
		u.ScanProfile = &custom
	}
	return ApplySettingsUpdate(existing, u)
}
