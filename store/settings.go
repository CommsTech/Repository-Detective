package store

// ResolveRepoSettings merges profile defaults, global config, repo profile, and repo overrides.
// Nil fields in repoSettings inherit from the merged base.
func ResolveRepoSettings(global GlobalSettingsSnapshot, repoSettings RepoSettings) EffectiveSettings {
	effective, _ := ResolveEffectiveSettingsWithMeta(global, repoSettings)
	return effective
}

// DefaultGlobalSettings returns conservative defaults matching beta/runtime defaults
// (viper enable_llm_auditors=false). Deep profile and explicit operator settings can re-enable AI.
func DefaultGlobalSettings() GlobalSettingsSnapshot {
	return GlobalSettingsSnapshot{
		ScanProfile:                 ScanProfileCustom,
		Enabled:                     true,
		PolicyLevel:                 "issue_only",
		WorkspaceMode:               "api",
		AnalysisDepth:               3,
		EnableLLMAuditors:           false,
		EnableTrivy:                 true,
		EnableGrype:                 true,
		EnableGitleaks:              false,
		EnableSemgrep:               false,
		EnableGovulncheck:           false,
		EnableGosec:                 false,
		EnableStaticcheck:           false,
		EnableHadolint:              false,
		EnableCheckov:               false,
		EnableLinters:               true,
		SeverityGate:                "high",
		ConfidenceGate:              0.5,
		IssuePolicy:                 "all",
		RemediationPolicy:           "off",
		RunnerPolicy:                "core",
		ScheduleEnabled:             false,
		ScheduleCron:                "",
		AIPolicy:                    AIPolicyDisabled,
		EnableHealthChecks:          true,
		EnableTechDebtChecks:        true,
		EnableReliabilityChecks:     true,
		EnableMaintainabilityChecks: true,
		EnableTestGapChecks:         true,
		EnablePerformanceChecks:     true,
		EnableAIRiskChecks:          false,
		HealthMaxFindings:           100,
		HealthLargeFileLines:        1000,
		HealthLargeFunctionLines:    150,
		HealthMaxNestingDepth:       5,
		HealthMaxFunctionParams:     7,
		EnableCodeGraph:             true,
		GraphMaxNodes:               5000,
		GraphMaxEdges:               15000,
		GraphTimeoutSeconds:         120,
		GraphIncludeFunctions:       true,
		GraphIncludeFindings:        true,
		GovulncheckTimeoutSeconds:   0,
		GosecTimeoutSeconds:         0,
		StaticcheckTimeoutSeconds:   0,
		GoScannerMaxFindings:        100,
		HadolintTimeoutSeconds:      0,
		CheckovTimeoutSeconds:       0,
		IACScannerMaxFindings:       100,
	}
}
