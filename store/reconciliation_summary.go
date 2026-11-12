package store

import "time"

// ReconciliationSummary explains how scan findings relate to forge issues for a repo or scan.
type ReconciliationSummary struct {
	RepositoryID int64  `json:"repository_id"`
	ScanID       string `json:"scan_id,omitempty"`
	LatestScanID string `json:"latest_scan_id"`

	// Scan-side counts
	ScanFindingsTotal       int `json:"scan_findings_total"`
	ActivePresentOpen       int `json:"active_present_open"`
	ActionableActiveOpen    int `json:"actionable_active_open"`
	InformationalActiveOpen int `json:"informational_active_open"`
	OpenFindingsTotal       int `json:"open_findings_total"`
	ReportOnlyFindings      int `json:"report_only_findings"`
	ResolvedVerifiedOpen    int `json:"resolved_verified_open"`
	DuplicateFindings       int `json:"duplicate_findings"`

	// Forge-side counts (from tracked external_issues mappings)
	ForgeOpenIssues      int `json:"forge_open_issues"`
	MappedOpenIssues     int `json:"mapped_open_issues"`
	UnmappedOpenIssues   int `json:"unmapped_open_issues"`
	FindingsWithIssue    int `json:"findings_with_open_issue"`
	FindingsWithoutIssue int `json:"findings_without_open_issue"`

	SkippedDueReportOnly     int `json:"skipped_due_report_only"`
	SkippedDueBacklogControl int `json:"skipped_due_backlog_control"`

	// Pipeline / policy context
	IssueSyncStatus       string `json:"issue_sync_status"`
	PersistenceStatus     string `json:"persistence_status"`
	DryRunReportOnly      bool   `json:"dry_run_report_only"`
	IssueFilingEnabled    bool   `json:"issue_filing_enabled"`
	ReportOnlyExplanation string `json:"report_only_explanation"`
	CountsDifferExpected  bool   `json:"counts_differ_expected"`
	MismatchWarning       string `json:"mismatch_warning,omitempty"`

	LastIssueSyncAt      *time.Time `json:"last_issue_sync_at,omitempty"`
	LastReconciliationAt *time.Time `json:"last_reconciliation_at,omitempty"`
}
