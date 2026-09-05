package openclaw

// ScanType identifies the review context.
type ScanType string

const (
	ScanTypeRepo       ScanType = "repo"
	ScanTypePreinstall ScanType = "preinstall"
	ScanTypeContainer  ScanType = "container"
)

// ReviewPacket is the redacted payload sent to OpenClaw.
type ReviewPacket struct {
	ScanID   string         `json:"scan_id"`
	RepoID   int64          `json:"repo_id"`
	RepoName string         `json:"repo_name"`
	ScanType ScanType       `json:"scan_type"`
	Policy   ReviewPolicy   `json:"policy"`
	Summary  ReviewSummary  `json:"summary"`
	Findings []FindingInput `json:"findings"`
}

// ReviewPolicy describes filing constraints visible to the reviewer.
type ReviewPolicy struct {
	IssueFiling   string `json:"issue_filing"`
	RemediationPR string `json:"remediation_pr"`
	AdvisoryOnly  bool   `json:"advisory_only"`
}

// ReviewSummary is high-level scan context without raw source.
type ReviewSummary struct {
	Languages          []string `json:"languages"`
	ScannerCoverage    []string `json:"scanner_coverage"`
	GraphState         string   `json:"graph_state"`
	SBOMState          string   `json:"sbom_state"`
	ContainerScanState string   `json:"container_scan_state"`
}

// FindingHistory is prior lifecycle hints for a finding.
type FindingHistory struct {
	SeenBefore            bool `json:"seen_before"`
	ClosedAsFalsePositive bool `json:"closed_as_false_positive"`
	ClosedAsFixed         bool `json:"closed_as_fixed"`
}

// FindingInput is one redacted finding for advisory review.
type FindingInput struct {
	Fingerprint         string         `json:"fingerprint"`
	RuleID              string         `json:"rule_id"`
	Title               string         `json:"title"`
	Severity            string         `json:"severity"`
	Confidence          string         `json:"confidence"`
	Source              string         `json:"source"`
	Path                string         `json:"path"`
	Line                int            `json:"line"`
	DescriptionRedacted string         `json:"description_redacted"`
	EvidenceRedacted    string         `json:"evidence_redacted"`
	History             FindingHistory `json:"history"`
}

// ReviewResponse is strict JSON expected from OpenClaw.
type ReviewResponse struct {
	ReviewID          string           `json:"review_id"`
	OverallAssessment string           `json:"overall_assessment"`
	Recommendations   []Recommendation `json:"recommendations"`
}

// Recommendation is one advisory suggestion (never auto-applied).
type Recommendation struct {
	Fingerprint         string   `json:"fingerprint"`
	Classification      string   `json:"classification"`
	SuggestedAction     string   `json:"suggested_action"`
	SuggestedSeverity   string   `json:"suggested_severity"`
	SuggestedConfidence string   `json:"suggested_confidence"`
	Reason              string   `json:"reason"`
	EvidenceGaps        []string `json:"evidence_gaps"`
}

// ReviewResult is the outcome of a review invocation.
type ReviewResult struct {
	ReviewID             string
	Status               string
	Model                string
	FindingsSent         int
	RedactionCount       int
	RecommendationsCount int
	OverallAssessment    string
	Response             *ReviewResponse
	Error                string
	PromptStored         bool
}

// Classification values from OpenClaw output.
const (
	ClassLikelyTruePositive    = "likely_true_positive"
	ClassPossibleFalsePositive = "possible_false_positive"
	ClassNeedsHumanReview      = "needs_human_review"
)
