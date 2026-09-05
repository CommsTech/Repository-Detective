package reconcile

// Issue status detected during reconciliation.
const (
	StatusStillPresent        = "still_present"
	StatusAlreadyFixedVerify  = "already_fixed_verify"
	StatusDuplicate           = "duplicate"
	StatusFalsePositive       = "false_positive"
	StatusSuppressed          = "suppressed"
	StatusStaleRule           = "stale_rule"
	StatusScannerNotRun       = "scanner_not_run"
	StatusNeedsEnrichment     = "needs_enrichment"
	StatusNeedsHumanReview    = "needs_human_review"
)

// Proposed actions for apply mode.
const (
	ActionNone              = "none"
	ActionComment           = "comment"
	ActionLabel             = "label"
	ActionEnrich            = "enrich"
	ActionMarkFalsePositive = "mark_false_positive"
	ActionSuppress          = "suppress"
	ActionCloseVerified     = "close_verified"
	ActionCloseDuplicate    = "close_duplicate"
)

// Item is one external issue reconciliation result.
type Item struct {
	IssueNumber     int    `json:"issue_number"`
	IssueURL        string `json:"issue_url"`
	FindingID       int64  `json:"finding_id"`
	Title           string `json:"title"`
	Fingerprint     string `json:"fingerprint"`
	Source          string `json:"source"`
	RuleID          string `json:"rule_id"`
	Severity        string `json:"severity"`
	Category        string `json:"category"`
	FindingStatus   string `json:"finding_status"`
	LatestScanID    string `json:"latest_scan_id"`
	InLatestScan    bool   `json:"in_latest_scan"`
	Status          string `json:"status"`
	ProposedAction  string `json:"proposed_action"`
	Reason          string `json:"reason"`
	CanonicalIssue  int    `json:"canonical_issue,omitempty"`
	LabelsToAdd     []string `json:"labels_to_add,omitempty"`
}

// Result is the output of preview or apply.
type Result struct {
	RunID           string `json:"run_id"`
	RepositoryID    int64  `json:"repository_id"`
	Preview         bool   `json:"preview"`
	Items           []Item `json:"items"`
	Applied         int    `json:"applied"`
	Skipped         int    `json:"skipped"`
	Errors          []string `json:"errors,omitempty"`
}
