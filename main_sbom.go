package main

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/store"
)

func persistScanSBOM(ctx context.Context, scanID string, repositoryID int64, result *analyzers.AnalysisResult) {
	if result == nil || result.Sbom == nil || rdStore == nil || scanID == "" || repositoryID <= 0 {
		return
	}
	sb := *result.Sbom
	rec := store.SBOMArtifact{
		RepositoryID: repositoryID,
		ScanID:       scanID,
		Format:       sb.Format,
		PackageCount: sb.PackageCount,
		VulnCount:    sb.VulnCount,
		Status:       string(sb.Status),
		Detail:       sb.Detail,
		ArtifactPath: sb.ArtifactPath,
	}
	if err := rdStore.SaveSBOMArtifact(ctx, rec); err != nil {
		logger.Warnf("Failed to persist SBOM for scan %s: %v", scanID, err)
		return
	}
	if bs, ok := rdStore.(interface {
		UpdateScanPipelineState(context.Context, string, string, map[string]any) error
	}); ok {
		_ = bs.UpdateScanPipelineState(ctx, scanID, "", map[string]any{
			"sbom_status":       string(sb.Status),
			"sbom_format":       sb.Format,
			"sbom_package_count": sb.PackageCount,
			"sbom_vuln_count":   sb.VulnCount,
			"sbom_detail":       sb.Detail,
		})
	}
}
