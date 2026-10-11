package store

import "context"

// GraphStatusForScan resolves repository map state for a scan ID.
func (s *SQLiteStore) GraphStatusForScan(ctx context.Context, scanID string, global GlobalSettingsSnapshot) (GraphStatus, error) {
	scan, err := s.GetScan(ctx, scanID)
	if err != nil {
		return ResolveGraphStatus(GraphStatusInput{ScanFound: false, RepoFound: true}), nil
	}
	_, repoErr := s.GetRepository(ctx, scan.RepositoryID)
	repoFound := repoErr == nil

	enabled, depth, truncated, graphErr := GraphMetaFromSummary(scan.SummaryJSON)
	settings, _ := s.GetRepoSettings(ctx, scan.RepositoryID)
	effective, _ := ResolveEffectiveSettingsFull(global, settings)
	if !enabled {
		enabled = effective.EnableCodeGraph
	}
	if depth <= 0 {
		depth = effective.AnalysisDepth
	}
	if depth <= 0 {
		depth = 2
	}

	var graphJSON []byte
	nodeCount, edgeCount := 0, 0
	if record, err := s.GetScanGraph(ctx, scanID); err == nil {
		graphJSON = record.GraphJSON
		nodeCount = record.NodeCount
		edgeCount = record.EdgeCount
	}
	if !truncated {
		truncated = graphTruncatedFromSummary(scan.SummaryJSON)
	}

	return ResolveGraphStatus(GraphStatusInput{
		ScanFound:     true,
		RepoFound:     repoFound,
		ScanID:        scanID,
		RepoID:        scan.RepositoryID,
		ScanStatus:    scan.Status,
		GraphEnabled:  enabled,
		AnalysisDepth: depth,
		GraphJSON:     graphJSON,
		NodeCount:     nodeCount,
		EdgeCount:     edgeCount,
		Truncated:     truncated,
		GraphError:    graphErr,
		SummaryJSON:   scan.SummaryJSON,
	}), nil
}

// GraphStatusForRepository resolves repository map state from the latest stored graph.
func (s *SQLiteStore) GraphStatusForRepository(ctx context.Context, repositoryID int64, global GlobalSettingsSnapshot) (GraphStatus, error) {
	if _, err := s.GetRepository(ctx, repositoryID); err != nil {
		return ResolveGraphStatus(GraphStatusInput{RepoFound: false}), nil
	}
	record, graphErr := s.GetLatestScanGraphForRepo(ctx, repositoryID)
	if graphErr != nil {
		scan, scanErr := s.GetLatestReconcilableScanForRepository(ctx, repositoryID)
		if scanErr != nil {
			return ResolveGraphStatus(GraphStatusInput{
				ScanFound: false, RepoFound: true, RepoID: repositoryID,
			}), nil
		}
		return s.GraphStatusForScan(ctx, scan.ID, global)
	}
	if _, err := s.GetScan(ctx, record.ScanID); err != nil {
		return ResolveGraphStatus(GraphStatusInput{
			ScanFound: false, RepoFound: true, RepoID: repositoryID,
		}), nil
	}
	return s.GraphStatusForScan(ctx, record.ScanID, global)
}

func graphTruncatedFromSummary(raw []byte) bool {
	_, _, truncated, _ := GraphMetaFromSummary(raw)
	return truncated
}
