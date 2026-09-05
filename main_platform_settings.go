package main

import (
	"context"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
)

func applyPlatformSettingsToRuntime(settings store.PlatformSettings) {
	if p := strings.TrimSpace(settings.ScanProfile); p != "" {
		config.ScanProfile = store.NormalizeScanProfile(p)
	}
	if settings.SchedulerEnabled != nil {
		config.SchedulerEnabled = *settings.SchedulerEnabled
	}
	if settings.NotificationsEnabled != nil {
		config.NotificationsEnabled = *settings.NotificationsEnabled
		if notifyManager != nil {
			notifyManager.SetEnabled(*settings.NotificationsEnabled)
		}
	}
	if settings.PreinstallAuditEnabled != nil {
		config.PreinstallAuditEnabled = *settings.PreinstallAuditEnabled
	}
	if settings.RemediationPlannerEnabled != nil {
		config.RemediationPlannerEnabled = *settings.RemediationPlannerEnabled
	}
	if settings.RemediationPREnabled != nil {
		config.RemediationPREnabled = *settings.RemediationPREnabled
	}
	if settings.EvidenceClosureEnabled != nil {
		config.EvidenceClosureEnabled = *settings.EvidenceClosureEnabled
	}
	if settings.AutoCreateIssues != nil {
		config.AutoCreateIssues = *settings.AutoCreateIssues
	}
	if settings.AnalysisDepth != nil {
		config.AnalysisDepth = *settings.AnalysisDepth
	}
	if settings.ConfidenceGate != nil {
		config.MinIssueConfidence = *settings.ConfidenceGate
	}
	if settings.SeverityGate != "" {
		config.GiteaStatusFailOn = strings.ToLower(strings.TrimSpace(settings.SeverityGate))
	}
	applyConfigBool(&config.EnableTrivy, settings.EnableTrivy)
	applyConfigBool(&config.EnableGrype, settings.EnableGrype)
	applyConfigBool(&config.EnableGitleaks, settings.EnableGitleaks)
	applyConfigBool(&config.EnableSemgrep, settings.EnableSemgrep)
	applyConfigBool(&config.EnableGovulncheck, settings.EnableGovulncheck)
	applyConfigBool(&config.EnableGosec, settings.EnableGosec)
	applyConfigBool(&config.EnableStaticcheck, settings.EnableStaticcheck)
	applyConfigBool(&config.EnableHadolint, settings.EnableHadolint)
	applyConfigBool(&config.EnableCheckov, settings.EnableCheckov)
	applyConfigBool(&config.EnableLinters, settings.EnableLinters)
	applyConfigBool(&config.EnablePerformanceChecks, settings.EnablePerformanceChecks)
	applyConfigBool(&config.EnableCodeGraph, settings.EnableCodeGraph)

	if settings.AIRecommendationsEnabled != nil {
		config.OpenClawAIReview.Enabled = *settings.AIRecommendationsEnabled
	}
	if settings.AIRecommendationsMaxTokensPerScan != nil {
		config.OpenClawAIReview.MaxTokensPerScan = *settings.AIRecommendationsMaxTokensPerScan
	}
	if settings.AIRecommendationsTokenBudgetPerScan != nil {
		config.OpenClawAIReview.CAH.TokenBudgetPerScan = *settings.AIRecommendationsTokenBudgetPerScan
	}
	if settings.AIRecommendationsMaxFindingsPerScan != nil {
		config.OpenClawAIReview.MaxFindingsPerScan = *settings.AIRecommendationsMaxFindingsPerScan
	}

	snap := apiGlobalSnapshotFromConfig()
	snap = store.ApplyPlatformSettingsToGlobal(snap, settings)
	appGlobalSnapshot = snap
	operator.InvalidateToolsCache()

	if controlPlaneHandler != nil {
		controlPlaneHandler.SetGlobal(snap)
		if notifyManager != nil {
			controlPlaneHandler.SetNotificationGlobal(notifyManager.Config())
		}
	}
	if operatorUI != nil {
		operatorUI.SetGlobal(snap)
		operatorUI.SetPlatformContext(uhPlatformContextFromConfig())
		if notifyManager != nil {
			operatorUI.SetNotificationGlobal(notifyManager.Config())
		}
		operatorUI.SetPreinstallEnabled(config.PreinstallAuditEnabled)
		if config.RemediationPlannerEnabled {
			operatorUI.SetRemediationBackend(true, remediationUIBridge{})
		} else {
			operatorUI.SetRemediationBackend(false, nil)
		}
		if config.RemediationPREnabled {
			operatorUI.SetRemediationPRBackend(true, remediationPRUIBridge{})
		} else {
			operatorUI.SetRemediationPRBackend(false, nil)
		}
		if config.EvidenceClosureEnabled {
			operatorUI.SetClosureBackend(true, closureUIBridge{})
		} else {
			operatorUI.SetClosureBackend(false, nil)
		}
	}
}

func applyConfigBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

func apiGlobalSnapshotFromConfig() store.GlobalSettingsSnapshot {
	return api.GlobalSnapshotFromConfig(api.GlobalConfigInput{
		ScanProfile:                 config.ScanProfile,
		WorkspaceMode:               config.WorkspaceMode,
		AnalysisDepth:               config.AnalysisDepth,
		EnableLLMAuditors:           config.EnableLLMAuditors,
		EnableTrivy:                 config.EnableTrivy,
		EnableGrype:                 config.EnableGrype,
		EnableGitleaks:              config.EnableGitleaks,
		EnableSemgrep:               config.EnableSemgrep,
		EnableGovulncheck:           config.EnableGovulncheck,
		EnableGosec:                 config.EnableGosec,
		EnableStaticcheck:           config.EnableStaticcheck,
		EnableHadolint:              config.EnableHadolint,
		EnableCheckov:               config.EnableCheckov,
		EnableLinters:               config.EnableLinters,
		GiteaStatusFailOn:           config.GiteaStatusFailOn,
		MinIssueConfidence:          config.MinIssueConfidence,
		AutoCreateIssues:            config.AutoCreateIssues,
		EnableHealthChecks:          config.EnableHealthChecks,
		EnableTechDebtChecks:        config.EnableTechDebtChecks,
		EnableReliabilityChecks:     config.EnableReliabilityChecks,
		EnableMaintainabilityChecks: config.EnableMaintainabilityChecks,
		EnableTestGapChecks:         config.EnableTestGapChecks,
		EnablePerformanceChecks:     config.EnablePerformanceChecks,
		EnableAIRiskChecks:          config.EnableAIRiskChecks,
		HealthMaxFindings:           config.HealthMaxFindings,
		HealthLargeFileLines:        config.HealthLargeFileLines,
		HealthLargeFunctionLines:    config.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       config.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     config.HealthMaxFunctionParams,
		EnableCodeGraph:             config.EnableCodeGraph,
		GraphMaxNodes:               config.GraphMaxNodes,
		GraphMaxEdges:               config.GraphMaxEdges,
		GraphTimeoutSeconds:         config.GraphTimeoutSeconds,
		GraphIncludeFunctions:       config.GraphIncludeFunctions,
		GraphIncludeFindings:        config.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   config.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         config.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   config.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        config.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      config.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       config.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       config.IACScannerMaxFindings,
	})
}

func uhPlatformContextFromConfig() ui.PlatformContext {
	return ui.PlatformContext{
		GiteaURLConfigured:                 strings.TrimSpace(config.GiteaURL) != "",
		GiteaTokenConfigured:               strings.TrimSpace(config.GiteaToken) != "",
		APIKeyConfigured:                   strings.TrimSpace(config.APIKey) != "",
		WebhookSecretConfigured:            strings.TrimSpace(config.WebhookSecret) != "",
		RunnerSharedSecretSet:              strings.TrimSpace(config.RunnerSharedSecret) != "",
		RunnerCallbackBaseURL:              config.RunnerCallbackBaseURL,
		PublicURL:                          config.PublicURL,
		RemediationPRRequireApproval:       config.RemediationPRRequireApproval,
		RemediationPRMaxFiles:              config.RemediationPRMaxFilesChanged,
		RemediationPRMaxDiffLines:          config.RemediationPRMaxDiffLines,
		RemediationPRBranchPrefix:          config.RemediationPRBranchPrefix,
		LLMSanityGateEnabled:               config.LLMSanityGateEnabled,
		BacklogControlEnabled:              config.DogfoodBacklogControlEnabled,
		MaxIssuesPerScan:                   config.Reporting.MaxIssuesPerScan,
		ScanPolicyMode:                     store.DeploymentScanMode(appGlobalSnapshot),
		NotificationsEnabled:               config.NotificationsEnabled,
		SchedulerEnabled:                   config.SchedulerEnabled,
		RunnerDelegationEnabled:            config.RunnerDelegationEnabled,
		RunnerRequireHMAC:                  config.RunnerRequireHMAC,
		RunnerMode:                         config.RunnerMode,
		RemediationPRRequireTests:          config.RemediationPRRequireTests,
		RemediationPRUseRunnerVerification: config.RemediationPRUseRunnerVerification,
		GiteaActionsTestBackendEnabled:     config.GiteaActionsTestBackendEnabled,
		OpenClawAIReviewEnabled:            config.OpenClawAIReview.Enabled,
		OpenClawEndpointConfigured:         config.OpenClawAIReview.EndpointConfigured(),
		ContainerScanningEnabled:           config.ContainerScan.Enabled,
		ContainerScanRequireRunner:         config.ContainerScan.RequireRunner,
		ContainerScanAllowCoreSocket:       config.ContainerScan.AllowCoreDockerSocket,
		ContainerScanCreateIssues:          config.ContainerScan.CreateIssues,
	}
}

func loadAndApplyPlatformSettingsOverrides() error {
	if rdStore == nil {
		return nil
	}
	settings, err := rdStore.GetPlatformSettings(context.Background())
	if err != nil {
		return fmt.Errorf("load platform settings: %w", err)
	}
	if settings.UpdatedAt == "" && settings.ScanProfile == "" && settings.IssuePolicy == "" &&
		settings.SchedulerEnabled == nil && settings.EnableTrivy == nil && settings.AIRecommendationsEnabled == nil {
		return nil
	}
	applyPlatformSettingsToRuntime(settings)
	logger.Infof("Applied platform settings overrides from database (updated_at=%s)", settings.UpdatedAt)
	return nil
}
