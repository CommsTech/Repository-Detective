package profile

// Repository layout classifications.
const (
	LayoutSingleApp        = "single_app"
	LayoutMonorepo         = "monorepo"
	LayoutNestedServices   = "nested_services"
	LayoutLibrary          = "library"
	LayoutInfrastructure   = "infrastructure"
	LayoutDocumentation    = "documentation"
	LayoutMixed            = "mixed"
)

// Language/ecosystem tags.
const (
	EcosystemGo           = "go"
	EcosystemPython       = "python"
	EcosystemJavaScript   = "javascript"
	EcosystemTypeScript   = "typescript"
	EcosystemShell        = "shell"
	EcosystemDocker       = "docker"
	EcosystemKubernetes   = "kubernetes"
	EcosystemTerraform    = "terraform"
	EcosystemAnsible      = "ansible"
	EcosystemMarkdownDocs = "markdown_docs"
	EcosystemUnknown      = "unknown"
	EcosystemMixed        = "mixed"
)

// Source type classifications for findings.
const (
	SourceTypeSource     = "source"
	SourceTypeTest       = "test"
	SourceTypeDocs       = "docs"
	SourceTypeGenerated  = "generated"
	SourceTypeVendor     = "vendor"
	SourceTypeExample    = "example"
	SourceTypeConfig     = "config"
	SourceTypeDependency = "dependency"
	SourceTypeUnknown    = "unknown"
)

// Reporting actions — findings are never deleted, only routed.
const (
	ActionAutoIssue            = "auto_issue"
	ActionManualReview         = "manual_review"
	ActionReportOnly           = "report_only"
	ActionSuppressedWithReason = "suppressed_with_reason"
	ActionDisabledByPolicy     = "disabled_by_policy"
)

// Scanner applicability reasons.
const (
	ApplicabilityApplicable              = "applicable"
	ApplicabilitySkippedNoMatchingFiles  = "skipped_no_matching_files"
	ApplicabilitySkippedDisabledByPolicy = "skipped_disabled_by_repo_policy"
	ApplicabilitySkippedGeneratedVendor  = "skipped_generated_or_vendor_only"
	ApplicabilitySkippedToolUnavailable  = "skipped_tool_unavailable"
	ApplicabilitySkippedDocsOnlyRepo     = "skipped_docs_only_repo"
	ApplicabilitySkippedUnsupportedType  = "skipped_unsupported_repo_type"
)

// Reporting modes.
const (
	ModeMonitorOnly = "monitor_only"
	ModeHighSignal  = "high_signal"
	ModeStandard    = "standard"
	ModeStrict      = "strict"
	ModeCompliance  = "compliance"
)

// Lifecycle events for deduplication tracking.
const (
	LifecycleFirstSeen        = "first_seen"
	LifecycleStillPresent     = "still_present"
	LifecycleChangedLine      = "changed_line"
	LifecycleMovedFile        = "moved_file"
	LifecycleResolved         = "resolved"
	LifecycleVerifiedResolved = "verified_resolved"
	LifecycleSuppressed       = "suppressed"
	LifecycleReopened         = "reopened"
)
