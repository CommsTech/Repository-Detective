package store

import "time"

// ScanCompletion carries scan finish data without importing analyzers.
type ScanCompletion struct {
	IssuesFound           int
	FilesAnalyzed         int
	AnalysisTime          time.Duration
	OverallScore          float64
	ScoreComplete         bool
	ScoreIncompleteReason string
	ScoreExplanation      string
	CommitSHA             string
	WorkspaceModeUsed     string
	PolicySnapshot        any
	ScannerResults        []ScanCompletionScanner
	GraphEnabled          bool
	GraphState            string
	GraphNodeCount        int
	GraphEdgeCount        int
	GraphTruncated        bool
	GraphError            string
	GraphJSON             []byte
	RepoProfile           any
}

// ScanCompletionScanner is a scanner outcome for persistence.
type ScanCompletionScanner struct {
	Scanner             string
	Status              string
	FindingsCount       int
	Detail              string
	ApplicabilityReason string
}
