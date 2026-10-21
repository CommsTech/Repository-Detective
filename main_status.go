package main

import (
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/operator"
	"github.com/gin-gonic/gin"
)

func operatorScannerConfig() operator.ScannerConfig {
	return operator.ScannerConfig{
		EnableTrivy:       config.EnableTrivy,
		EnableGrype:       config.EnableGrype,
		EnableGitleaks:    config.EnableGitleaks,
		EnableSemgrep:     config.EnableSemgrep,
		EnableGovulncheck: config.EnableGovulncheck,
		EnableGosec:       config.EnableGosec,
		EnableStaticcheck: config.EnableStaticcheck,
		EnableHadolint:    config.EnableHadolint,
		EnableCheckov:     config.EnableCheckov,
		EnableLinters:     config.EnableLinters,
		PreinstallGit:     config.PreinstallAllowGitClone,
		RemediationGit:    config.RemediationPREnabled,
	}
}

func operatorFeatureFlags() operator.FeatureFlags {
	healthy := config.DatabaseEnabled && rdStore != nil
	return operator.FeatureFlags{
		DatabaseEnabled:           config.DatabaseEnabled,
		DatabaseHealthy:           healthy,
		SchedulerEnabled:          config.SchedulerEnabled,
		RunnerDelegationEnabled:   runnerCfg.DelegationEnabled,
		NotificationsEnabled:      config.NotificationsEnabled,
		PreinstallAuditEnabled:    config.PreinstallAuditEnabled,
		RemediationPlannerEnabled: config.RemediationPlannerEnabled,
		RemediationPREnabled:      config.RemediationPREnabled,
		EvidenceClosureEnabled:    config.EvidenceClosureEnabled,
		PublicURLConfigured:       config.PublicURL != "",
		UIEnabled:                 config.UIEnabled,
		ScanProfile:               config.ScanProfile,
	}
}

func buildReadiness(status string) operator.Readiness {
	r := operator.Readiness{
		ProductName: "Repository Detective",
		Tagline:     "Inspect. Analyze. Improve.",
		Version:     version,
		Commit:      commit,
		BuildDate:   buildDate,
		Service:     "repository-detective",
		Status:      status,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Features:    operatorFeatureFlags(),
		Tools:       operator.CheckTools(operatorScannerConfig()),
	}
	if aiClient != nil {
		r.AIProvider = string(aiClient.Provider())
		r.AIModel = aiClient.Model()
		r.AIAnalysis = "Enabled"
	} else {
		r.AIProvider = "disabled"
		r.AIAnalysis = "Disabled"
	}
	r.PrivacyMode = privacy.NormalizeMode(config.PrivacyMode)
	d := privacy.EvaluateAIEgress(config.PrivacyMode, config.effectiveAIProvider(), firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL), config.EnableLLMAuditors)
	r.AIEndpointClass = d.EndpointClass
	switch r.PrivacyMode {
	case privacy.ModeLocalOnly:
		r.CodeEgressPolicy = "BLOCKED_BY_POLICY"
	case privacy.ModeExternalAIEnabled:
		r.CodeEgressPolicy = "EXTERNAL_AI_ENABLED"
	default:
		r.CodeEgressPolicy = "HYBRID"
	}
	return r
}

func healthPayload(ready bool) gin.H {
	status := "starting"
	code := gin.H{
		"status":                status,
		"ready":                 ready,
		"service":               "repository-detective",
		"product_name":          "Repository Detective",
		"tagline":               "Inspect. Analyze. Improve.",
		"version":               version,
		"commit":                commit,
		"build_date":            buildDate,
		"public_url_configured": config.PublicURL != "",
	}
	if ready {
		status = "healthy"
		code["status"] = status
		code["ready"] = true
		r := buildReadiness(status)
		code["features"] = r.Features
		code["tools_summary"] = toolsSummary(r.Tools)
	}
	return code
}

func toolsSummary(tools []operator.ToolStatus) gin.H {
	missing := []string{}
	configured := 0
	available := 0
	for _, t := range tools {
		if t.Configured {
			configured++
			if t.Available {
				available++
			} else {
				missing = append(missing, t.Name)
			}
		}
	}
	return gin.H{
		"configured_count": configured,
		"available_count":  available,
		"missing":          missing,
	}
}
