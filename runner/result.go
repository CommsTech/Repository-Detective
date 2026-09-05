package runner

import (
	"encoding/json"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

// FindingResult is a deterministic finding from a runner scan.
type FindingResult struct {
	Fingerprint string  `json:"fingerprint"`
	Category    string  `json:"category"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
	Source      string  `json:"source"`
	RuleID      string  `json:"rule_id"`
	PackageName string  `json:"package_name,omitempty"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	CodeSnippet string  `json:"code_snippet,omitempty"`
}

// ScannerResultDTO mirrors scanner run output for transport.
type ScannerResultDTO struct {
	Scanner       string `json:"scanner"`
	Status        string `json:"status"`
	FindingsCount int    `json:"findings_count"`
	Detail        string `json:"detail,omitempty"`
	Error         string `json:"error,omitempty"`
}

// JobResult is the signed payload returned by a runner.
type JobResult struct {
	Version         int                  `json:"version"`
	JobID           string               `json:"job_id"`
	ScanID          string               `json:"scan_id"`
	Status          string               `json:"status"`
	StartedAt       time.Time            `json:"started_at"`
	FinishedAt      time.Time            `json:"finished_at"`
	ScannerResults  []ScannerResultDTO   `json:"scanner_results"`
	Findings        []FindingResult      `json:"findings"`
	Graph           *graph.Graph         `json:"graph,omitempty"`
	WorkspaceMeta   models.WorkspaceMeta `json:"workspace_meta,omitempty"`
	FilesAnalyzed   int                  `json:"files_analyzed"`
	Errors          []string             `json:"errors,omitempty"`
	Warnings        []string             `json:"warnings,omitempty"`
	ForbiddenAction string               `json:"forbidden_action,omitempty"`
	ContainerScan   *ContainerScanDTO    `json:"container_scan,omitempty"`
}

// ContainerScanDTO transports container scan metadata in job results.
type ContainerScanDTO struct {
	Image      string            `json:"image"`
	Digest     string            `json:"digest,omitempty"`
	VulnCount  int               `json:"vuln_count"`
	SBOMPath   string            `json:"sbom_path,omitempty"`
	SBOMFormat string            `json:"sbom_format,omitempty"`
	Coverage   map[string]string `json:"coverage"`
	Warnings   []string          `json:"warnings,omitempty"`
}

// EncodedSize returns the JSON byte length of the result.
func (r JobResult) EncodedSize() (int64, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return 0, err
	}
	return int64(len(b)), nil
}

// ToAnalysisResult converts a validated runner result for core issue/status handling.
func (r JobResult) ToAnalysisResult(repoFullName, ref string, policy *analyzers.PolicySnapshot) *analyzers.AnalysisResult {
	out := &analyzers.AnalysisResult{
		Repository:        repoFullName,
		Commit:            ref,
		ScanID:            r.ScanID,
		AnalysisTime:      r.FinishedAt.Sub(r.StartedAt),
		FilesAnalyzed:     r.FilesAnalyzed,
		IssuesFound:       len(r.Findings),
		PolicySnapshot:    policy,
		WorkspaceModeUsed: r.WorkspaceMeta.ModeUsed,
		Graph:             r.Graph,
	}
	if r.WorkspaceMeta.RefUsed != "" {
		out.Commit = r.WorkspaceMeta.RefUsed
	}
	for _, sr := range r.ScannerResults {
		out.ScannerResults = append(out.ScannerResults, scanners.RunResult{
			Scanner: sr.Scanner,
			Status:  scanners.Status(sr.Status),
			Detail:  sr.Detail,
		})
	}
	for _, f := range r.Findings {
		out.Issues = append(out.Issues, ai.CodeIssue{
			Fingerprint: f.Fingerprint,
			Category:    f.Category,
			Severity:    f.Severity,
			Confidence:  f.Confidence,
			Source:      f.Source,
			RuleID:      f.RuleID,
			PackageName: f.PackageName,
			Title:       f.Title,
			Description: f.Description,
			File:        f.File,
			LineNumber:  f.Line,
			CodeSnippet: f.CodeSnippet,
		})
	}
	return out
}

// SummaryJSON builds a compact summary for runner_jobs.result_summary_json.
func (r JobResult) SummaryJSON() json.RawMessage {
	summary := map[string]any{
		"status":         r.Status,
		"findings_count": len(r.Findings),
		"files_analyzed": r.FilesAnalyzed,
		"scanner_count":  len(r.ScannerResults),
		"warnings":       len(r.Warnings),
		"errors":         len(r.Errors),
	}
	if r.Graph != nil {
		summary["graph_nodes"] = r.Graph.Metrics.NodeCount
		summary["graph_edges"] = r.Graph.Metrics.EdgeCount
	}
	b, _ := json.Marshal(summary)
	return b
}
