package ai

import (
	"time"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/models"
)

// AttackSurfaceRequest is the input for attack surface analysis.
type AttackSurfaceRequest struct {
	RepositoryName string
	Files          string
}

// AttackSurfaceResponse is the output of attack surface analysis.
type AttackSurfaceResponse struct {
	EntryPoints     []models.EntryPoint
	AttackSurface   []models.AttackSurfaceEntry
	TrustBoundaries []models.TrustBoundary
}

// FileContent is source code passed to auditor prompts.
type FileContent struct {
	Path     string
	Content  string
	Language string
}

// AuditorRequest is the input for an auditor agent.
type AuditorRequest struct {
	RepositoryName     string
	VulnerabilityClass string
	Files              []gitea.RepositoryContent
	FileContents       []FileContent
	AttackSurface      []models.AttackSurfaceEntry
	AuditorType        string
}

// AuditorFinding is a candidate finding from an auditor.
type AuditorFinding struct {
	File        string
	Line        int
	Hypothesis  string
	CodeSnippet string
	CallChain   []string
	Severity    string
	Confidence  float64
}

// AuditorResponse is the output of an auditor agent.
type AuditorResponse struct {
	Findings []AuditorFinding
}

// DebaterRequest is the input for a debater agent.
type DebaterRequest struct {
	Finding models.CandidateFinding
	Role    string
}

// DebaterResponse is the output of a debater agent.
type DebaterResponse struct {
	Confidence float64
	Arguments  string
}

// PoCRequest is the input for PoC generation.
type PoCRequest struct {
	Finding models.DedupedFinding
}

// PoCResponse is the output of PoC generation.
type PoCResponse struct {
	Type        string
	Command     string
	Language    string
	Explanation string
}

// CodeAnalysisRequest represents a request for code analysis.
type CodeAnalysisRequest struct {
	RepositoryName string
	FilePath       string
	CodeContent    string
	Language       string
	Context        string
	AnalysisType   string
}

// CodeAnalysisResult represents the result of code analysis.
type CodeAnalysisResult struct {
	Issues                []CodeIssue
	Suggestions           []CodeSuggestion
	OverallScore          float64
	ScoreComplete         bool
	ScoreIncompleteReason string
	ScoreExplanation      string
	AnalysisTime          time.Duration
	ModelUsed             string
}

// CodeIssue represents a detected issue in the code.
type CodeIssue struct {
	Severity               string  `json:"severity"`
	Category               string  `json:"category"`
	Title                  string  `json:"title"`
	Description            string  `json:"description"`
	File                   string  `json:"file,omitempty"`
	LineNumber             int     `json:"line_number,omitempty"`
	ColumnNumber           int     `json:"column_number,omitempty"`
	CodeSnippet            string  `json:"code_snippet,omitempty"`
	ProofOfConcept         string  `json:"proof_of_concept,omitempty"`
	Confidence             float64 `json:"confidence"`
	ClusterID              string  `json:"cluster_id,omitempty"`
	Source                 string  `json:"source,omitempty"`
	RuleID                 string  `json:"rule_id,omitempty"`
	ScanID                 string  `json:"scan_id,omitempty"`
	Fingerprint            string  `json:"fingerprint,omitempty"`
	PackageName            string  `json:"package_name,omitempty"`
	LifecycleState         string  `json:"lifecycle_state,omitempty"`
	FromAI                 bool    `json:"from_ai,omitempty"`
	Fixable                string  `json:"fixable,omitempty"`
	FixComplexity          string  `json:"fix_complexity,omitempty"`
	RegressionRisk           string  `json:"regression_risk,omitempty"`
	RequiredTests            string  `json:"required_tests,omitempty"`
	SuggestedPatchStrategy string  `json:"suggested_patch_strategy,omitempty"`
	SafeForAutoPR            bool    `json:"safe_for_auto_pr,omitempty"`
	// Normalized finding metadata (repo-structure aware reporting)
	NormalizedPath      string  `json:"normalized_path,omitempty"`
	SourceType          string  `json:"source_type,omitempty"`
	ReportingAction     string  `json:"reporting_action,omitempty"`
	FalsePositiveRisk   string  `json:"false_positive_risk,omitempty"`
	SuppressionReason   string  `json:"suppression_reason,omitempty"`
	RepoProfileSummary  string  `json:"repo_profile_summary,omitempty"`
	CommitSHA           string  `json:"commit_sha,omitempty"`
	Evidence            string  `json:"evidence,omitempty"`
	Remediation         string  `json:"remediation,omitempty"`
}

// CodeSuggestion represents a suggested improvement.
type CodeSuggestion struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CodeExample string `json:"code_example,omitempty"`
	Priority    string `json:"priority"`
}
