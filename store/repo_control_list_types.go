package store

// RepositoryControlRow is a fleet-control list entry for /ui/repos.
type RepositoryControlRow struct {
	RepositorySummary
	LastScanID         string
	IssueSyncStatus    string
	DryRunReportOnly   bool
	ScanFindingsTotal  int
	ActivePresentOpen  int
	ReportOnlyFindings int
	ForgeOpenIssues    int
	UnmappedOpenIssues int
	ResolvedVerified   int
	Duplicates         int
	SkippedReportOnly  int

	ScanEnabled       bool
	ScheduleEnabled   bool
	IssueFilingOn     bool
	ScanProfile       string
	DefaultReportOnly bool
	DefaultRef        string
	RawSettings       RepoSettings
}

type repoControlMetrics struct {
	LastScanID         string
	IssueSyncStatus    string
	DryRunReportOnly   bool
	ScanFindingsTotal  int
	ActivePresentOpen  int
	ReportOnlyFindings int
	ForgeOpenIssues    int
	UnmappedOpenIssues int
	ResolvedVerified   int
	Duplicates         int
}
