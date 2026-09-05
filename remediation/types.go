package remediation

import "time"

const (
	StatusProposed   = "proposed"
	StatusApproved   = "approved"
	StatusRejected   = "rejected"
	StatusSuperseded = "superseded"

	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"

	ComplexitySmall  = "small"
	ComplexityMedium = "medium"
	ComplexityLarge  = "large"
)

// Config holds global remediation planner settings.
type Config struct {
	Enabled          bool
	MinSeverity      string
	MinConfidence    float64
	UseAI            bool
	CommentOnIssue   bool
	GlobalAIAllowed  bool
}

// FindingContext is sanitized finding input for planning.
type FindingContext struct {
	FindingID    int64
	RepositoryID int64
	AuditID      string
	Fingerprint  string
	Category     string
	Severity     string
	Source       string
	RuleID       string
	Title        string
	Summary      string
	Confidence   float64
	FilePath     string
	Line         int
	PackageName  string
	FromAI       bool
	RepoFullName string
	ConnectedRepo bool
}

// Plan is a structured remediation plan (no patches).
type Plan struct {
	ID                  string
	FindingID           int64
	Fingerprint         string
	RepositoryID        int64
	AuditID             string
	Category            string
	Severity            string
	Source              string
	RuleID              string
	Title               string
	Summary             string
	FixStrategy         string
	AffectedFiles       []string
	TargetLine          int
	RequiredTests       []string
	ValidationCommands  []string
	RegressionRisk      string
	FixComplexity       string
	SafeForAutoPR       bool
	RequiresHumanReview bool
	BlockedReasons      []string
	Advisory            bool
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// AIAdvisor optionally enriches plans when deterministic recipes are insufficient.
type AIAdvisor interface {
	SuggestPlan(ctx FindingContext, draft Plan) (Plan, error)
}
