package scanners

import "git.commsnet.org/commstech/repository-detective/models"

// Config controls external deterministic scanners.
type Config struct {
	EnableTrivy            bool
	EnableGrype              bool
	EnableGitleaks           bool
	EnableSemgrep            bool
	EnableGovulncheck        bool
	EnableGosec              bool
	EnableStaticcheck        bool
	EnableHadolint           bool
	EnableCheckov            bool
	EnableLinters            bool
	TrivySeverity            string // comma-separated: UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL
	GrypeFailOn              string // negligible, low, medium, high, critical
	GitleaksConfig                   string // optional path to gitleaks config TOML
	GitleaksTimeoutSeconds           int    // 0 = use TimeoutSeconds
	SecretScanGitHistoryEnabled      bool
	SecretScanHistoryMaxCommits      int  // 0 = full history when cloning
	SecretScanRecentCommitsMax       int  // recent-commit window for scoped/quick scans
	SecretScanHistoryTimeoutSeconds  int  // 0 = use GitleaksTimeoutSeconds / TimeoutSeconds / 600
	SecretScanHistoryReportOnly      bool // pre-install: report findings without filing (enforced upstream)
	SecretScanRedact                 bool
	SemgrepConfig            string // registry ruleset or operator path, default p/ci
	SemgrepTimeoutSeconds    int    // 0 = use TimeoutSeconds
	SemgrepMaxFindings       int    // cap normalized findings, default 100
	SemgrepSeverityThreshold string // INFO, WARNING, ERROR — minimum Semgrep severity to include
	GovulncheckTimeoutSeconds int   // 0 = use TimeoutSeconds
	GosecTimeoutSeconds       int   // 0 = use TimeoutSeconds
	StaticcheckTimeoutSeconds int   // 0 = use TimeoutSeconds
	GoScannerMaxFindings      int   // cap per Go scanner, default 100
	HadolintTimeoutSeconds    int   // 0 = use TimeoutSeconds
	CheckovTimeoutSeconds     int   // 0 = use TimeoutSeconds
	IACScannerMaxFindings     int   // cap per IaC scanner, default 100
	LinterMinSeverity        string // error, warning, info
	TimeoutSeconds           int
}

// DefaultConfig returns sensible scanner defaults.
func DefaultConfig() Config {
	return Config{
		EnableTrivy:       true,
		EnableGrype:       true,
		EnableLinters:     true,
		TrivySeverity:     "HIGH,CRITICAL",
		GrypeFailOn:       "high",
		LinterMinSeverity: "warning",
		TimeoutSeconds:    120,
	}
}

// Finding is a normalized result from any external scanner.
type Finding struct {
	ID          string
	Source      string // trivy, grype, linter name
	Category    string
	Severity    string
	Title       string
	Description string
	File        string
	Line        int
	Code        string
	Confidence  float64
	Reference   string // CVE ID, rule code, etc.
}

// ToCandidateFinding converts a scanner finding to a pipeline candidate.
func (f Finding) ToCandidateFinding() models.CandidateFinding {
	confidence := f.Confidence
	if confidence <= 0 {
		confidence = 0.95
	}
	return models.CandidateFinding{
		ID:         f.ID,
		Hypothesis: f.Title,
		Evidence: models.Evidence{
			Code:      f.Code,
			CallChain: []string{f.File},
		},
		Severity:    f.Severity,
		Confidence:  confidence,
		AuditorType: f.Source,
		Category:    f.Category,
		File:        f.File,
		Line:        f.Line,
	}
}
