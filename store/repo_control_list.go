package store

import "context"

func (s *SQLiteStore) ListRepositoryControlRows(ctx context.Context, opts ListOptions) ([]RepositoryControlRow, error) {
	summaries, err := s.ListRepositoriesWithSummary(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}
	ids := make([]int64, len(summaries))
	for i, r := range summaries {
		ids[i] = r.ID
	}
	metrics, err := s.batchRepositoryControlMetrics(ctx, ids)
	if err != nil {
		return nil, err
	}
	settingsByID, err := s.batchRepoSettings(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]RepositoryControlRow, 0, len(summaries))
	for _, sum := range summaries {
		row := RepositoryControlRow{RepositorySummary: sum}
		if m, ok := metrics[sum.ID]; ok {
			row.LastScanID = m.LastScanID
			row.IssueSyncStatus = m.IssueSyncStatus
			row.DryRunReportOnly = m.DryRunReportOnly
			row.ScanFindingsTotal = m.ScanFindingsTotal
			row.ActivePresentOpen = m.ActivePresentOpen
			row.ReportOnlyFindings = m.ReportOnlyFindings
			row.ForgeOpenIssues = m.ForgeOpenIssues
			row.UnmappedOpenIssues = m.UnmappedOpenIssues
			row.ResolvedVerified = m.ResolvedVerified
			row.Duplicates = m.Duplicates
		}
		settings := settingsByID[sum.ID]
		settings.RepositoryID = sum.ID
		row.RawSettings = settings
		ref := sum.DefaultBranch
		if ref == "" {
			ref = "main"
		}
		row.DefaultRef = ref
		out = append(out, row)
	}
	return out, nil
}
