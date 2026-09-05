package models

import "time"

// PrepareReport is the output of the PREPARE stage.
type PrepareReport struct {
	Repository      string
	Commit          string
	ScanTime        time.Duration
	FilesFound      int
	FilesIndexed    int
	Languages       map[string]int
	EntryPoints     []EntryPoint
	AttackSurface   []AttackSurfaceEntry
	TrustBoundaries []TrustBoundary
	RecentVulns     []VulnContext
	TargetFiles     []string // empty = full repository scan
	CommitPinned    bool
}

// EntryPoint represents a public-facing function or endpoint.
type EntryPoint struct {
	File         string
	Line         int
	FunctionName string
	Type         string
	AuthRequired bool
}

// AttackSurfaceEntry is an I/O boundary or data entry point.
type AttackSurfaceEntry struct {
	File     string
	Line     int
	Type     string
	DataFlow string
}

// TrustBoundary represents a transition between trusted and untrusted zones.
type TrustBoundary struct {
	File      string
	Line      int
	FromZone  string
	ToZone    string
	Operation string
}

// VulnContext is a vulnerability pattern from git history.
type VulnContext struct {
	Commit   string
	Message  string
	Files    []string
	Severity string
}

// CandidateFinding is a vulnerability candidate from the SCAN stage.
type CandidateFinding struct {
	ID           string
	Hypothesis   string
	Evidence     Evidence
	Reachability Reachability
	Severity     string
	Confidence   float64
	AuditorType  string
	Category     string
	File         string
	Line         int
}

// Evidence contains supporting proof for a finding.
type Evidence struct {
	Code      string
	CallChain []string
	ASTNode   string
}

// Reachability describes how exploitable a finding is.
type Reachability struct {
	FromEntryPoint bool
	EntryPointRef  string
	Exploitable    bool
	AttackVector   string
}

// ValidatedFinding is a finding that survived the VALIDATE stage.
type ValidatedFinding struct {
	CandidateFinding
	DebateResult DebateResult
}

// DebateResult is the output of the VALIDATE stage.
type DebateResult struct {
	AdvocateConfidence float64
	CounselConfidence  float64
	AdvocateArgs       string
	CounselArgs        string
	Outcome            string
}

// DedupedFinding is a finding after the DEDUP stage.
type DedupedFinding struct {
	ID          string
	Severity    string
	Category    string
	Title       string
	Description string
	AuditorType string
	Files       []string
	Lines       []int
	Evidence    Evidence
	Confidence  float64
	DedupGroup  string
	ClusterID   string
	Related     []string
}

// ProvenFinding includes a proof-of-concept for the finding.
type ProvenFinding struct {
	DedupedFinding
	ProofOfConcept ProofOfConcept
}

// ProofOfConcept demonstrates a vulnerability.
type ProofOfConcept struct {
	Type        string
	Command     string
	Language    string
	Explanation string
}
