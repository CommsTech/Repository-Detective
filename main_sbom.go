package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/sbom"
	"git.commsnet.org/commstech/repository-detective/store"
)

func sbomAlreadyDurable(dataRoot string, repositoryID int64, scanID, path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || !sbom.ArtifactFileAvailable(path) {
		return false
	}
	prefix := filepath.Clean(filepath.Join(dataRoot, "sbom", fmt.Sprintf("%d", repositoryID), scanID))
	return strings.HasPrefix(filepath.Clean(path), prefix+string(filepath.Separator)) || filepath.Clean(path) == prefix
}

func persistScanSBOM(ctx context.Context, scanID string, repositoryID int64, result *analyzers.AnalysisResult) {
	if result == nil || result.Sbom == nil || rdStore == nil || scanID == "" || repositoryID <= 0 {
		return
	}
	sb := *result.Sbom
	artifactPath := strings.TrimSpace(sb.ArtifactPath)
	dataRoot := sbom.DataRootFromDatabasePath(config.DatabasePath)
	// Engine normally persists before workspace cleanup. Re-persist only when the
	// path is still a workspace/tmp file (or missing durable copy).
	if artifactPath != "" && !sbomAlreadyDurable(dataRoot, repositoryID, scanID, artifactPath) {
		durable, err := sbom.PersistArtifact(dataRoot, repositoryID, scanID, artifactPath)
		if err != nil {
			logger.Warnf("Failed to copy SBOM artifact for scan %s to durable storage: %v", scanID, err)
		} else {
			artifactPath = durable
			sb.ArtifactPath = durable
		}
	}
	rec := store.SBOMArtifact{
		RepositoryID: repositoryID,
		ScanID:       scanID,
		Format:       sb.Format,
		PackageCount: sb.PackageCount,
		VulnCount:    sb.VulnCount,
		Status:       string(sb.Status),
		Detail:       sb.Detail,
		ArtifactPath: artifactPath,
	}
	if err := rdStore.SaveSBOMArtifact(ctx, rec); err != nil {
		logger.Warnf("Failed to persist SBOM for scan %s: %v", scanID, err)
		return
	}
	if bs, ok := rdStore.(interface {
		UpdateScanPipelineState(context.Context, string, string, map[string]any) error
	}); ok {
		_ = bs.UpdateScanPipelineState(ctx, scanID, "", map[string]any{
			"sbom_status":        string(sb.Status),
			"sbom_format":        sb.Format,
			"sbom_package_count": sb.PackageCount,
			"sbom_vuln_count":    sb.VulnCount,
			"sbom_detail":        sb.Detail,
		})
	}
}
