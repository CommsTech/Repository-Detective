package scanners

import "git.commsnet.org/commstech/bugbot/models"

// Config controls external deterministic scanners.
type Config struct {
	EnableTrivy       bool
	EnableGrype       bool
	EnableLinters     bool
	TrivySeverity     string // comma-separated: UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL
	GrypeFailOn       string // negligible, low, medium, high, critical
	LinterMinSeverity string // error, warning, info
	TimeoutSeconds    int
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
