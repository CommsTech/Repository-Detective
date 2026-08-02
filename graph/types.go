package graph

import (
	"time"

	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func init() {
	scanners.RegisterDeterministicSource("graph")
}

// Config controls repository map generation.
type Config struct {
	Enabled          bool
	MaxNodes         int
	MaxEdges         int
	TimeoutSeconds   int
	IncludeFunctions bool
	IncludeFindings  bool
}

// DefaultConfig returns Phase 11 defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		MaxNodes:         5000,
		MaxEdges:         15000,
		TimeoutSeconds:   120,
		IncludeFunctions: true,
		IncludeFindings:  true,
	}
}

func (c Config) normalized() Config {
	out := c
	if out.MaxNodes <= 0 {
		out.MaxNodes = 5000
	}
	if out.MaxEdges <= 0 {
		out.MaxEdges = 15000
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = 120
	}
	return out
}

// FileInput is a source file for graph analysis.
type FileInput struct {
	Path     string
	Content  string
	Language string
}

// FindingOverlay links scan findings to graph nodes.
type FindingOverlay struct {
	ID         string
	File       string
	Line       int
	Severity   string
	Category   string
	Source     string
	RuleID     string
	Title      string
	Confidence float64
}

// Node is a graph vertex.
type Node struct {
	ID          string             `json:"id"`
	Type        string             `json:"type"`
	Label       string             `json:"label"`
	Path        string             `json:"path,omitempty"`
	Language    string             `json:"language,omitempty"`
	PackageName string             `json:"package_name,omitempty"`
	Severity    string             `json:"severity,omitempty"`
	Category    string             `json:"category,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Disconnected bool              `json:"disconnected,omitempty"`
	Entrypoint  bool               `json:"entrypoint,omitempty"`
}

// Edge is a graph relationship.
type Edge struct {
	ID     string  `json:"id"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight,omitempty"`
}

// GraphMetrics summarizes graph statistics.
type GraphMetrics struct {
	NodeCount          int            `json:"node_count"`
	EdgeCount          int            `json:"edge_count"`
	OrphanFiles        int            `json:"orphan_files"`
	OrphanFunctions    int            `json:"orphan_functions"`
	DisconnectedPkgs   int            `json:"disconnected_packages"`
	SuspiciousIslands  int            `json:"suspicious_islands"`
	EntrypointCount    int            `json:"entrypoint_count"`
	FindingsOverlay    int            `json:"findings_overlay"`
	Truncated          bool           `json:"truncated,omitempty"`
	AggregationMode    string         `json:"aggregation_mode,omitempty"`
	ByType             map[string]int `json:"by_type,omitempty"`
	GeneratedAt        time.Time      `json:"generated_at"`
}

// Graph is the full repository map.
type Graph struct {
	RepositoryID string       `json:"repository_id,omitempty"`
	ScanID       string       `json:"scan_id,omitempty"`
	AuditID      string       `json:"audit_id,omitempty"`
	Nodes        []Node       `json:"nodes"`
	Edges        []Edge       `json:"edges"`
	Metrics      GraphMetrics `json:"metrics"`
}

// BuildInput is workspace context for graph generation.
type BuildInput struct {
	RepositoryID string
	ScanID       string
	AuditID      string
	Files        []FileInput
	AllPaths     []string
	Findings     []FindingOverlay
	Repo         RepoContext
}

// GraphFinding is a disconnected-code finding from graph analysis.
type GraphFinding struct {
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
	Detail      FindingDetail
}

// ToCandidateFindings converts graph findings to pipeline candidates.
func ToCandidateFindings(findings []GraphFinding) []models.CandidateFinding {
	out := make([]models.CandidateFinding, 0, len(findings))
	for _, f := range findings {
		detailJSON := f.Detail.JSON()
		dashboardText := f.Description
		if dashboardText == "" {
			dashboardText = f.Title
		}
		if len(dashboardText) > 6000 {
			dashboardText = dashboardText[:6000] + "…"
		}
		out = append(out, models.CandidateFinding{
			ID:         f.RuleID,
			Hypothesis: f.Title,
			Evidence: models.Evidence{
				Code:      dashboardText,
				CallChain: []string{f.File},
				ASTNode:   detailJSON,
			},
			Severity:    f.Severity,
			Confidence:  f.Confidence,
			AuditorType: f.Source,
			Category:    f.Category,
			File:        f.File,
			Line:        f.Line,
		})
	}
	return out
}
