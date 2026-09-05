package health

import (
	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func init() {
	for _, name := range []string{
		"health", "tech_debt", "reliability", "maintainability",
		"test_gap", "performance", "ai_generated_risk",
	} {
		scanners.RegisterDeterministicSource(name)
	}
}

// Config controls deterministic repository health checks.
type Config struct {
	Enabled                 bool
	EnableTechDebt          bool
	EnableReliability       bool
	EnableMaintainability   bool
	EnableTestGap           bool
	EnablePerformance       bool
	EnableAIRisk            bool
	MaxFindings             int
	LargeFileLines          int
	LargeFunctionLines      int
	MaxNestingDepth         int
	MaxFunctionParams       int
}

// DefaultConfig returns Phase 10 defaults (AI risk off by default).
func DefaultConfig() Config {
	return Config{
		Enabled:               true,
		EnableTechDebt:        true,
		EnableReliability:     true,
		EnableMaintainability: true,
		EnableTestGap:         true,
		EnablePerformance:     true,
		EnableAIRisk:          false,
		MaxFindings:           100,
		LargeFileLines:        1000,
		LargeFunctionLines:    150,
		MaxNestingDepth:       5,
		MaxFunctionParams:     7,
	}
}

func (c Config) normalized() Config {
	out := c
	if out.MaxFindings <= 0 {
		out.MaxFindings = 100
	}
	if out.LargeFileLines <= 0 {
		out.LargeFileLines = 1000
	}
	if out.LargeFunctionLines <= 0 {
		out.LargeFunctionLines = 150
	}
	if out.MaxNestingDepth <= 0 {
		out.MaxNestingDepth = 5
	}
	if out.MaxFunctionParams <= 0 {
		out.MaxFunctionParams = 7
	}
	return out
}

// FileInput is a source file for health analysis.
type FileInput struct {
	Path     string
	Content  string
	Language string
}

// RunInput is the workspace context for health checks.
type RunInput struct {
	Files    []FileInput
	AllPaths []string
}

// Finding is a normalized health check result.
type Finding struct {
	Category    string
	Source      string
	RuleID      string
	Severity    string
	Confidence  float64
	Title       string
	Description string
	File        string
	Line        int
	Evidence    string
}

// ToCandidateFindings converts health findings to pipeline candidates.
func ToCandidateFindings(findings []Finding) []models.CandidateFinding {
	out := make([]models.CandidateFinding, 0, len(findings))
	for i, f := range findings {
		id := f.RuleID
		if id == "" {
			id = f.Source + "-" + f.Category
		}
		out = append(out, models.CandidateFinding{
			ID:         id,
			Hypothesis: f.Title,
			Evidence: models.Evidence{
				Code:      f.Evidence,
				CallChain: []string{f.File},
			},
			Severity:    f.Severity,
			Confidence:  f.Confidence,
			AuditorType: f.Source,
			Category:    f.Category,
			File:        f.File,
			Line:        f.Line,
		})
		_ = i
	}
	return out
}
