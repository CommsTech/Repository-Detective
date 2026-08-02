package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/containers"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/sbom"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

// JobExecuteInput configures execution of a claimed job in a prepared workspace.
type JobExecuteInput struct {
	WorkspaceDir string
	ScannerCfg   scanners.Config
	HealthCfg    health.Config
	GraphCfg     graph.Config
	SkipPatterns []string
	Logger       *logrus.Logger
}

// ExecuteJob runs a claimed job based on its job type and allowed tasks.
func ExecuteJob(ctx context.Context, spec JobSpec, in JobExecuteInput) (JobResult, error) {
	jobType := strings.TrimSpace(spec.JobType)
	if jobType == "" || jobType == JobTypeScanFullRepoLegacy {
		jobType = JobTypeScanFullRepo
	}

	switch jobType {
	case JobTypeGraph:
		return executeGraphJob(ctx, spec, in)
	case JobTypeSBOM:
		return executeSBOMJob(ctx, spec, in)
	case JobTypeRemediationVerify:
		return executeRemediationVerifyJob(ctx, spec, in)
	case JobTypeContainerImageScan:
		return executeContainerImageScanJob(ctx, spec, in)
	case JobTypeScanFullRepo, JobTypePreinstallAudit:
		return ExecuteWorkspaceScan(ctx, spec, ExecuteInput(in))
	default:
		return ExecuteWorkspaceScan(ctx, spec, ExecuteInput(in))
	}
}

func executeGraphJob(ctx context.Context, spec JobSpec, in JobExecuteInput) (JobResult, error) {
	graphSpec := spec
	graphSpec.JobType = JobTypeGraph
	graphSpec.AllowedTasks = []string{"graph"}
	policy := graphSpec.EffectiveSettings
	policy.AnalysisDepth = 2
	policy.EnableCodeGraph = true
	graphSpec.EffectiveSettings = policy
	in.GraphCfg.IncludeFindings = false
	in.ScannerCfg = scanners.Config{}
	in.HealthCfg = health.Config{Enabled: false}
	result, err := ExecuteWorkspaceScan(ctx, graphSpec, ExecuteInput(in))
	// Graph delegation returns metrics only in v1 worker transport (full graph ingest deferred to artifacts).
	nodeCount, edgeCount := 0, 0
	if result.Graph != nil {
		nodeCount = result.Graph.Metrics.NodeCount
		edgeCount = result.Graph.Metrics.EdgeCount
	}
	result.Findings = nil
	result.Graph = nil
	if nodeCount > 0 || edgeCount > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("graph_nodes=%d graph_edges=%d", nodeCount, edgeCount))
	}
	return result, err
}

func executeSBOMJob(ctx context.Context, spec JobSpec, in JobExecuteInput) (JobResult, error) {
	started := time.Now().UTC()
	result := JobResult{
		Version: ContractVersion, JobID: spec.JobID, ScanID: spec.ScanID,
		Status: JobStatusCompleted, StartedAt: started,
	}
	if in.WorkspaceDir == "" {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, "workspace directory required")
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}
	sbomResult, err := sbom.GenerateAndCheck(ctx, in.WorkspaceDir, in.WorkspaceDir)
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, RedactLogLine(err.Error()))
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}
	result.ScannerResults = append(result.ScannerResults, ScannerResultDTO{
		Scanner: "sbom", Status: string(sbomResult.Status), Detail: RedactLogLine(sbomResult.Detail),
	})
	if sbomResult.Status == sbom.StatusCheckFailed || sbomResult.Status == sbom.StatusVulnerabilitiesFound {
		result.Warnings = append(result.Warnings, fmt.Sprintf("sbom status: %s", sbomResult.Status))
	}
	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func executeRemediationVerifyJob(ctx context.Context, spec JobSpec, in JobExecuteInput) (JobResult, error) {
	started := time.Now().UTC()
	result := JobResult{
		Version: ContractVersion, JobID: spec.JobID, ScanID: spec.ScanID,
		Status: JobStatusCompleted, StartedAt: started,
	}
	if in.WorkspaceDir == "" {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, "workspace directory required")
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}
	// Verification-only: run lightweight static analysis as a stand-in for allowlisted test commands.
	verifySpec := spec
	verifySpec.AllowedTasks = []string{"scanners"}
	policy := verifySpec.EffectiveSettings
	policy.AnalysisDepth = 1
	verifySpec.EffectiveSettings = policy
	verifyIn := in
	verifyIn.HealthCfg = health.Config{Enabled: false}
	verifyIn.GraphCfg = graph.Config{Enabled: false}
	scanResult, err := ExecuteWorkspaceScan(ctx, verifySpec, ExecuteInput(verifyIn))
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, RedactLogLine(err.Error()))
	} else if scanResult.Status == JobStatusFailed || len(scanResult.Errors) > 0 {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, scanResult.Errors...)
	} else {
		result.ScannerResults = scanResult.ScannerResults
		result.FilesAnalyzed = scanResult.FilesAnalyzed
	}
	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func executeContainerImageScanJob(ctx context.Context, spec JobSpec, in JobExecuteInput) (JobResult, error) {
	started := time.Now().UTC()
	result := JobResult{
		Version: ContractVersion, JobID: spec.JobID, ScanID: spec.ScanID,
		Status: JobStatusCompleted, StartedAt: started,
	}
	if spec.ContainerScan == nil || strings.TrimSpace(spec.ContainerScan.Image) == "" {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, "container scan payload missing image")
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}
	payload := spec.ContainerScan
	timeout := payload.TimeoutSeconds
	if timeout <= 0 && spec.Limits.TimeoutSeconds > 0 {
		timeout = spec.Limits.TimeoutSeconds
	}
	workDir := in.WorkspaceDir
	if workDir == "" {
		workDir = "."
	}
	scanResult, err := containers.RunImageScan(ctx, containers.ScanOptions{
		Image:          payload.Image,
		PullPolicy:     containers.PullPolicy(payload.PullPolicy),
		Tools:          payload.Tools,
		GenerateSBOM:   payload.GenerateSBOM,
		TimeoutSeconds: timeout,
		WorkDir:        workDir,
	})
	coverage := map[string]string{
		"trivy": scanResult.Coverage.Trivy,
		"grype": scanResult.Coverage.Grype,
		"syft":  scanResult.Coverage.Syft,
	}
	result.ContainerScan = &ContainerScanDTO{
		Image: payload.Image, Digest: scanResult.Digest, VulnCount: scanResult.VulnCount,
		SBOMPath: scanResult.SBOMPath, SBOMFormat: scanResult.SBOMFormat,
		Coverage: coverage, Warnings: scanResult.Warnings,
	}
	for _, w := range scanResult.Warnings {
		result.Warnings = append(result.Warnings, RedactLogLine(w))
	}
	for _, f := range scanResult.Findings {
		result.Findings = append(result.Findings, FindingResult{
			Fingerprint: "container-" + payload.Image + "-" + f.RuleID,
			Category:    "container",
			Severity:    f.Severity,
			Confidence:  f.Confidence,
			Source:      "container",
			RuleID:      f.RuleID,
			Title:       f.Title,
			Description: f.Description,
			File:        payload.Image,
		})
	}
	for _, sr := range []struct {
		name, status string
	}{
		{"trivy", scanResult.Coverage.Trivy},
		{"grype", scanResult.Coverage.Grype},
		{"syft", scanResult.Coverage.Syft},
	} {
		result.ScannerResults = append(result.ScannerResults, ScannerResultDTO{
			Scanner: sr.name, Status: sr.status,
		})
	}
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, RedactLogLine(err.Error()))
	}
	result.FinishedAt = time.Now().UTC()
	return result, nil
}

// PolicyConfigsFromSpec builds scanner/health/graph configs from a job spec snapshot.
func PolicyConfigsFromSpec(spec JobSpec) (scanners.Config, health.Config, graph.Config) {
	policy := spec.EffectiveSettings
	return analyzers.ScannersConfigFromSnapshot(scanners.DefaultConfig(), policy),
		healthFromPolicySnapshot(policy),
		graphFromPolicySnapshot(policy)
}

func healthFromPolicySnapshot(p analyzers.PolicySnapshot) health.Config {
	return health.Config{
		Enabled: p.EnableHealthChecks, EnableTechDebt: p.EnableTechDebtChecks,
		EnableReliability: p.EnableReliabilityChecks, EnableMaintainability: p.EnableMaintainabilityChecks,
		EnableTestGap: p.EnableTestGapChecks, EnablePerformance: p.EnablePerformanceChecks,
		EnableAIRisk: p.EnableAIRiskChecks, MaxFindings: p.HealthMaxFindings,
		LargeFileLines: p.HealthLargeFileLines, LargeFunctionLines: p.HealthLargeFunctionLines,
		MaxNestingDepth: p.HealthMaxNestingDepth, MaxFunctionParams: p.HealthMaxFunctionParams,
	}
}

func graphFromPolicySnapshot(p analyzers.PolicySnapshot) graph.Config {
	return graph.Config{
		Enabled: p.EnableCodeGraph, MaxNodes: p.GraphMaxNodes, MaxEdges: p.GraphMaxEdges,
		TimeoutSeconds: p.GraphTimeoutSeconds, IncludeFunctions: p.GraphIncludeFunctions,
		IncludeFindings: p.GraphIncludeFindings,
	}
}

// JobTypeAllowed reports whether a worker may execute a job type.
func JobTypeAllowed(allowed []string, jobType string) bool {
	if len(allowed) == 0 {
		return true
	}
	jobType = strings.TrimSpace(jobType)
	if jobType == JobTypeScanFullRepoLegacy {
		jobType = JobTypeScanFullRepo
	}
	for _, t := range allowed {
		if strings.TrimSpace(t) == jobType {
			return true
		}
	}
	return false
}

// SBOMSummaryJSON returns a compact SBOM summary for job telemetry.
func SBOMSummaryJSON(result JobResult) json.RawMessage {
	summary := map[string]any{"job_type": JobTypeSBOM, "status": result.Status}
	for _, sr := range result.ScannerResults {
		if sr.Scanner == "sbom" {
			summary["sbom_status"] = sr.Status
			summary["detail"] = sr.Detail
		}
	}
	b, _ := json.Marshal(summary)
	return b
}
