package analyzers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"yourusername/gitea-bugbot/ai"
	"yourusername/gitea-bugbot/gitea"
)

// ============================================================================
// CONFIG & RESULT TYPES (kept for compatibility)
// ============================================================================

// Config holds analyzer configuration
type Config struct {
	MaxFileSize           int64
	AnalysisDepth         int
	EnableSecurity        bool
	EnableQuality         bool
	SkipPatterns          []string
	LanguageMapping       map[string]string
}

// CodeSuggestion represents a code improvement suggestion
type CodeSuggestion struct {
	Type        string
	File        string
	Line        int
	Suggestion  string
	Explanation string
}

// AnalysisResult represents the complete result of analyzing a repository
type AnalysisResult struct {
	Repository     string
	Commit         string
	AnalysisTime   time.Duration
	FilesAnalyzed  int
	IssuesFound    int
	Issues         []ai.CodeIssue
	Suggestions    []CodeSuggestion
	OverallScore   float64
	Errors         []string
}

// ============================================================================
// STAGE RESULT TYPES
// ============================================================================

// PrepareReport is the output of the PREPARE stage
type PrepareReport struct {
	Repository    string
	Commit       string
	ScanTime     time.Duration
	FilesFound   int
	FilesIndexed int
	Languages    map[string]int // language -> count
	EntryPoints  []EntryPoint // public APIs, endpoints, handlers
	AttackSurface []AttackSurfaceEntry
	TrustBoundaries []TrustBoundary
	RecentVulns  []VulnContext // past vulnerability patterns from git history
}

// EntryPoint represents a public-facing function/endpoint
type EntryPoint struct {
	File         string
	Line         int
	FunctionName string
	Type         string // "http_handler", "api", "cli", "service"
	AuthRequired bool
}

// AttackSurfaceEntry is an I/O boundary or data entry point
type AttackSurfaceEntry struct {
	File         string
	Line         int
	Type         string // "user_input", "file_read", "network", "env"
	DataFlow     string // how data moves from entry to sink
}

// TrustBoundary represents a transition between trusted/untrusted zones
type TrustBoundary struct {
	File         string
	Line         int
	FromZone     string // "external", "internal", "privileged"
	ToZone       string
	Operation    string // "auth_check", "data_parse", "system_call"
}

// VulnContext is a vulnerability pattern from git history
type VulnContext struct {
	Commit     string
	Message   string
	Files     []string
	Severity  string
}

// CandidateFinding is a vulnerability candidate from the SCAN stage
type CandidateFinding struct {
	ID            string
	Hypothesis    string   // "This code is vulnerable to X because Y"
	Evidence      Evidence
	Reachability  Reachability
	Severity      string   // "critical", "high", "medium", "low"
	Confidence    float64  // 0.0-1.0
	AuditorType  string   // which auditor found it
	File          string
	Line          int
}

// Evidence contains the proof of a finding
type Evidence struct {
	Code        string   // vulnerable code snippet
	CallChain   []string // function call chain
	ASTNode     string   // optional AST context
}

// Reachability describes how exploitable a finding is
type Reachability struct {
	FromEntryPoint bool     // can it be reached from an entry point?
	EntryPointRef string   // which entry point
	Exploitable   bool     // can it be triggered externally?
	AttackVector  string   // HTTP, CLI, local, etc.
}

// ValidatedFinding is a finding that survived the VALIDATE stage
type ValidatedFinding struct {
	CandidateFinding
	DebateResult DebateResult
}

// DebateResult is the output of the VALIDATE stage
type DebateResult struct {
	AdvocateConfidence float64
	CounselConfidence  float64
	AdvocateArgs       string
	CounselArgs        string
	Outcome            string // "validated", "downgraded", "dismissed"
}

// DedupedFinding is a finding after DEDUP stage
type DedupedFinding struct {
	ID           string
	Severity     string
	Category     string
	Title        string
	Description  string
	Files        []string // all files with this same root cause
	Lines        []int
	Evidence     Evidence
	Confidence   float64
	DedupGroup   string // root cause identifier
}

// ProvenFinding includes a PoC for the finding
type ProvenFinding struct {
	DedupedFinding
	ProofOfConcept ProofOfConcept
}

// ProofOfConcept is a triggering input that demonstrates the vulnerability
type ProofOfConcept struct {
	Type        string // "curl", "script", "asan"
	Command     string // the actual PoC to run
	Language    string
	Explanation string
}

// FinalReport is the complete Bugbot report
type FinalReport struct {
	Repository      string
	Commit         string
	GeneratedAt    time.Time
	TotalTimeMs   int64
	Stages        []string // which stages completed

	Prepare   *PrepareReport
	Candidates []CandidateFinding
	Validated  []ValidatedFinding
	Deduped   []DedupedFinding
	Proven    []ProvenFinding

	Stats ReportStats
}

// ReportStats are summary statistics
type ReportStats struct {
	FilesAnalyzed     int
	CandidatesFound   int
	ValidatedFindings int
	DedupedFindings  int
	ProvenFindings   int
	CriticalCount     int
	HighCount        int
	MediumCount      int
	LowCount         int
}

// ============================================================================
// ENGINE - CAH PIPELINE ORCHESTRATOR
// ============================================================================

// Engine coordinates the CAH multi-stage analysis pipeline
type Engine struct {
	giteaClient *gitea.Client
	aiClient    *ai.OpenWebUIClient
	logger      *logrus.Logger
	config      *Config
}

// NewEngine creates a new CAH-pipeline analysis engine
func NewEngine(giteaClient *gitea.Client, aiClient *ai.OpenWebUIClient, config *Config, logger *logrus.Logger) *Engine {
	return &Engine{
		giteaClient: giteaClient,
		aiClient:    aiClient,
		logger:      logger,
		config:      config,
	}
}

// RunCAHPipeline runs the full 5-stage CAH pipeline
func (e *Engine) RunCAHPipeline(ctx context.Context, owner, repo, ref string) (*FinalReport, error) {
	startTime := time.Now()
	report := &FinalReport{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Commit:      ref,
		GeneratedAt: time.Now(),
		Stages:      []string{},
	}

	// Stage 1: PREPARE
	e.logger.Info("[CAH:PREPARE] Starting preparation phase...")
	pStart := time.Now()
	prepareReport, err := e.Prepare(ctx, owner, repo, ref)
	if err != nil {
		e.logger.Errorf("[CAH:PREPARE] Failed: %v", err)
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	prepareReport.ScanTime = time.Since(pStart)
	report.Prepare = prepareReport
	report.Stages = append(report.Stages, "prepare")
	e.logger.Infof("[CAH:PREPARE] Done in %v — found %d entry points, %d attack surface entries",
		prepareReport.ScanTime, len(prepareReport.EntryPoints), len(prepareReport.AttackSurface))

	// Stage 2: SCAN
	e.logger.Info("[CAH:SCAN] Starting scan phase...")
	sStart := time.Now()
	candidates, err := e.Scan(ctx, prepareReport)
	if err != nil {
		e.logger.Errorf("[CAH:SCAN] Failed: %v", err)
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	report.Candidates = candidates
	report.Stages = append(report.Stages, "scan")
	e.logger.Infof("[CAH:SCAN] Done in %v — found %d candidates", time.Since(sStart), len(candidates))

	// Stage 3: VALIDATE
	e.logger.Info("[CAH:VALIDATE] Starting validation phase...")
	vStart := time.Now()
	validated, err := e.Validate(ctx, candidates)
	if err != nil {
		e.logger.Errorf("[CAH:VALIDATE] Failed: %v", err)
		return nil, fmt.Errorf("validate failed: %w", err)
	}
	report.Validated = validated
	report.Stages = append(report.Stages, "validate")
	e.logger.Infof("[CAH:VALIDATE] Done in %v — %d/%d validated", time.Since(vStart), len(validated), len(candidates))

	// Stage 4: DEDUP
	e.logger.Info("[CAH:DEDUP] Starting deduplication phase...")
	deduped := e.Dedup(validated)
	report.Deduped = deduped
	report.Stages = append(report.Stages, "dedup")
	e.logger.Infof("[CAH:DEDUP] Done — %d unique findings from %d candidates", len(deduped), len(candidates))

	// Stage 5: PROVE
	e.logger.Info("[CAH:PROVE] Starting proof generation phase...")
	pStart = time.Now()
	proven, err := e.Prove(ctx, deduped)
	if err != nil {
		e.logger.Warnf("[CAH:PROVE] Some proofs failed: %v", err)
	}
	report.Proven = proven
	report.Stages = append(report.Stages, "prove")
	e.logger.Infof("[CAH:PROVE] Done in %v — generated %d proofs", time.Since(pStart), len(proven))

	// Compile stats
	report.Stats = e.compileStats(report)
	report.TotalTimeMs = time.Since(startTime).Milliseconds()

	e.logger.Infof("[CAH:PIPELINE] Complete in %v — %d critical, %d high, %d medium, %d low",
		time.Since(startTime), report.Stats.CriticalCount, report.Stats.HighCount,
		report.Stats.MediumCount, report.Stats.LowCount)

	return report, nil
}

// ============================================================================
// STAGE 1: PREPARE
// ============================================================================

// Prepare maps the repository attack surface
func (e *Engine) Prepare(ctx context.Context, owner, repo, ref string) (*PrepareReport, error) {
	report := &PrepareReport{
		Repository:     fmt.Sprintf("%s/%s", owner, repo),
		Commit:         ref,
		Languages:      make(map[string]int),
		EntryPoints:    []EntryPoint{},
		AttackSurface:  []AttackSurfaceEntry{},
		TrustBoundaries: []TrustBoundary{},
		RecentVulns:    []VulnContext{},
	}

	// Fetch all files
	files, err := e.giteaClient.ListAllFiles(ctx, owner, repo, ref, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	report.FilesFound = len(files)

	// Build language map and identify files to analyze
	var analyzableFiles []string
	for _, f := range files {
		if e.shouldAnalyzeFile(f.Path) {
			report.FilesIndexed++
			analyzableFiles = append(analyzableFiles, f.Path)
			lang := e.detectLanguage(f.Path, "")
			report.Languages[lang]++
		}
	}

	// Use AI to identify entry points, attack surface, and trust boundaries
	// by analyzing all files together (Prepare stage gets the full picture)
	e.logger.Info("[CAH:PREPARE] Running attack surface analysis...")

	// Build a context summary for the AI
	contextSummary := e.buildPrepareContext(files)

	// Call AI to identify entry points and attack surface
	surfaceFindings, err := e.aiClient.AnalyzeAttackSurface(ctx, &ai.AttackSurfaceRequest{
		RepositoryName: report.Repository,
		Files:         contextSummary,
	})
	if err != nil {
		e.logger.Warnf("[CAH:PREPARE] Attack surface analysis failed: %v", err)
	} else {
		report.EntryPoints = surfaceFindings.EntryPoints
		report.AttackSurface = surfaceFindings.AttackSurface
		report.TrustBoundaries = surfaceFindings.TrustBoundaries
	}

	return report, nil
}

// buildPrepareContext creates a context summary for the Prepare stage
func (e *Engine) buildPrepareContext(files []gitea.RepositoryContent) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Repository has %d files\n\n", len(files)))

	// Group by directory
	dirs := make(map[string][]string)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if dir == "." {
			dir = "/"
		}
		dirs[dir] = append(dirs[dir], f.Name)
	}

	// Write top-level structure
	sb.WriteString("Top-level structure:\n")
	for dir, files := range dirs {
		if len(files) > 20 {
			sb.WriteString(fmt.Sprintf("  %s/ (%d files)\n", dir, len(files)))
		} else {
			sb.WriteString(fmt.Sprintf("  %s/\n", dir))
			for _, f := range files {
				sb.WriteString(fmt.Sprintf("    - %s\n", f))
			}
		}
	}

	return sb.String()
}

// ============================================================================
// STAGE 2: SCAN
// ============================================================================

// AuditorType defines the type of security auditor
type AuditorType string

const (
	AuditorSQL      AuditorType = "sql"
	AuditorXSS     AuditorType = "xss"
	AuditorAuth    AuditorType = "auth"
	AuditorInject  AuditorType = "injection"
	AuditorCrypto  AuditorType = "crypto"
	AuditorRace    AuditorType = "race"
	AuditorMemory  AuditorType = "memory"
	AuditorConfig  AuditorType = "config"
)

// Scan runs all auditor agents in parallel and collects candidates
func (e *Engine) Scan(ctx context.Context, prepare *PrepareReport) ([]CandidateFinding, error) {
	// Get files to analyze
	files, err := e.giteaClient.ListAllFiles(ctx,
		strings.Split(prepare.Repository, "/")[0],
		strings.Split(prepare.Repository, "/")[1],
		prepare.Commit, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var analyzableFiles []gitea.RepositoryContent
	for _, f := range files {
		if e.shouldAnalyzeFile(f.Path) {
			analyzableFiles = append(analyzableFiles, f)
		}
	}

	// Run all auditors in parallel using goroutines
	type result struct {
		findings []CandidateFinding
		err      error
	}

	// Define auditors to run
	auditors := []struct {
		auditorType AuditorType
		promptType  string
	}{
		{AuditorSQL, "sql_injection"},
		{AuditorXSS, "xss"},
		{AuditorAuth, "auth_bypass"},
		{AuditorInject, "command_injection"},
		{AuditorCrypto, "hardcoded_secrets"},
		{AuditorConfig, "misconfiguration"},
	}

	// Channel to collect results
	results := make(chan result, len(auditors))

	for _, auditor := range auditors {
		go func(a AuditorType, prompt string) {
			findings, err := e.runAuditor(ctx, a, prompt, prepare, analyzableFiles)
			results <- result{findings: findings, err: err}
		}(auditor.auditorType, auditor.promptType)
	}

	// Collect all findings
	var allCandidates []CandidateFinding
	for i := 0; i < len(auditors); i++ {
		r := <-results
		if r.err != nil {
			e.logger.Warnf("[CAH:SCAN] Auditor %s failed: %v", auditors[i].auditorType, r.err)
			continue
		}
		allCandidates = append(allCandidates, r.findings...)
	}

	return allCandidates, nil
}

// runAuditor runs a single auditor agent across all files
func (e *Engine) runAuditor(ctx context.Context, auditorType AuditorType, vulnClass string,
	prepare *PrepareReport, files []gitea.RepositoryContent) ([]CandidateFinding, error) {

	e.logger.Infof("[CAH:SCAN] Running %s auditor on %d files", auditorType, len(files))

	var candidates []CandidateFinding

	// Process files in batches to avoid overwhelming the AI
	batchSize := 10
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[i:end]

		// Build auditor request
		req := &ai.AuditorRequest{
			RepositoryName: prepare.Repository,
			VulnerabilityClass: string(vulnClass),
			Files:       batch,
			AttackSurface: prepare.AttackSurface,
			AuditorType: string(auditorType),
		}

		// Call auditor
		resp, err := e.aiClient.RunAuditor(ctx, req)
		if err != nil {
			e.logger.Warnf("[CAH:SCAN] %s auditor batch failed: %v", auditorType, err)
			continue
		}

		// Convert to CandidateFinding
		for _, f := range resp.Findings {
			candidates = append(candidates, CandidateFinding{
				ID:           fmt.Sprintf("%s-%s-%d", auditorType, vulnClass, len(candidates)+1),
				Hypothesis:   f.Hypothesis,
				Evidence:     Evidence{Code: f.CodeSnippet, CallChain: f.CallChain},
				Reachability: Reachability{FromEntryPoint: true},
				Severity:     f.Severity,
				Confidence:   f.Confidence,
				AuditorType: string(auditorType),
				File:         f.File,
				Line:         f.Line,
			})
		}
	}

	e.logger.Infof("[CAH:SCAN] %s auditor found %d candidates", auditorType, len(candidates))
	return candidates, nil
}

// ============================================================================
// STAGE 3: VALIDATE
// ============================================================================

// Validate runs debater agents on each candidate
func (e *Engine) Validate(ctx context.Context, candidates []CandidateFinding) ([]ValidatedFinding, error) {
	if len(candidates) == 0 {
		return []ValidatedFinding{}, nil
	}

	// Process in batches for efficiency
	batchSize := 5
	var validated []ValidatedFinding

	for i := 0; i < len(candidates); i += batchSize {
		end := i + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]

		// Run validation for each candidate in parallel
		type vResult struct {
			candidate CandidateFinding
			validated *ValidatedFinding
		}
		vResults := make(chan vResult, len(batch))

		for _, c := range batch {
			go func(cand CandidateFinding) {
				v, err := e.validateOne(ctx, cand)
				vResults <- vResult{candidate: cand, validated: v}
			}(c)
		}

		// Collect results
		for j := 0; j < len(batch); j++ {
			vr := <-vResults
			if vr.validated != nil && vr.validated.DebateResult.Outcome == "validated" {
				validated = append(validated, *vr.validated)
			}
		}
	}

	return validated, nil
}

// validateOne runs a single candidate through debaters
func (e *Engine) validateOne(ctx context.Context, candidate CandidateFinding) (*ValidatedFinding, error) {
	// Run advocate and counselor in parallel
	type dResult struct {
		confidence float64
		args       string
	}

	advocateCh := make(chan dResult, 1)
	counselCh := make(chan dResult, 1)

	// Advocate argues FOR exploitation
	go func() {
		resp, err := e.aiClient.RunDebater(ctx, &ai.DebaterRequest{
			Finding:       candidate,
			Role:          "advocate",
		})
		if err != nil {
			advocateCh <- dResult{0, ""}
			return
		}
		advocateCh <- dResult{resp.Confidence, resp.Arguments}
	}()

	// Counselor argues AGAINST exploitation
	go func() {
		resp, err := e.aiClient.RunDebater(ctx, &ai.DebaterRequest{
			Finding:       candidate,
			Role:          "counsel",
		})
		if err != nil {
			counselCh <- dResult{0, ""}
			return
		}
		counselCh <- dResult{resp.Confidence, resp.Arguments}
	}()

	advocate := <-advocateCh
	counsel := <-counselCh

	// Determine outcome
	outcome := "downgraded"
	if advocate.confidence > counsel.confidence+0.2 {
		outcome = "validated"
	} else if counsel.confidence > advocate.confidence+0.3 {
		outcome = "dismissed"
	}

	return &ValidatedFinding{
		CandidateFinding: candidate,
		DebateResult: DebateResult{
			AdvocateConfidence: advocate.confidence,
			CounselConfidence:  counsel.confidence,
			AdvocateArgs:       advocate.args,
			CounselArgs:        counsel.args,
			Outcome:            outcome,
		},
	}, nil
}

// ============================================================================
// STAGE 4: DEDUP
// ============================================================================

// Dedup collapses semantically equivalent findings
func (e *Engine) Dedup(candidates []ValidatedFinding) []DedupedFinding {
	if len(candidates) == 0 {
		return []DedupedFinding{}
	}

	// Group by root cause (simplified: group by file + similar line range)
	groups := make(map[string][]ValidatedFinding)
	for _, c := range candidates {
		// Use file + line as dedup key
		key := fmt.Sprintf("%s:%d", c.File, c.Line/10*10) // group by 10-line blocks
		groups[key] = append(groups[key], c)
	}

	var deduped []DedupedFinding
	for _, group := range groups {
		// Keep the highest-confidence finding
		best := group[0]
		for i := 1; i < len(group); i++ {
			if group[i].Confidence > best.Confidence {
				best = group[i]
			}
		}

		// Collect all affected files
		var files []string
		var lines []int
		for _, c := range group {
			files = append(files, c.File)
			lines = append(lines, c.Line)
		}

		deduped = append(deduped, DedupedFinding{
			ID::       best.ID,
			Severity:  best.Severity,
			Title:     best.Hypothesis,
			Files:     files,
			Lines:     lines,
			Evidence:  best.Evidence,
			Confidence: best.Confidence,
			DedupGroup: fmt.Sprintf("%s:%d", best.File, best.Line/10*10),
		})
	}

	return deduped
}

// ============================================================================
// STAGE 5: PROVE
// ============================================================================

// Prove generates PoCs for validated findings
func (e *Engine) Prove(ctx context.Context, findings []DedupedFinding) ([]ProvenFinding, error) {
	var proven []ProvenFinding

	for _, f := range findings {
		poc, err := e.aiClient.GeneratePoC(ctx, &ai.PoCRequest{
			Finding: f,
		})
		if err != nil {
			e.logger.Warnf("[CAH:PROVE] Failed to generate PoC for %s: %v", f.ID, err)
			continue
		}

		proven = append(proven, ProvenFinding{
			DedupedFinding: f,
			ProofOfConcept: ProofOfConcept{
				Type:        poc.Type,
				Command:     poc.Command,
				Language:    poc.Language,
				Explanation: poc.Explanation,
			},
		})
	}

	return proven, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (e *Engine) shouldAnalyzeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	skipExtensions := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".ico": true, ".svg": true, ".woff": true, ".ttf": true,
	}
	if skipExtensions[ext] {
		return false
	}
	for _, pattern := range e.config.SkipPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}
	skipDirs := []string{"node_modules", "vendor", ".git", "build", "dist", "target"}
	for _, dir := range skipDirs {
		if strings.Contains(path, dir) {
			return false
		}
	}
	return true
}

func (e *Engine) detectLanguage(path, content string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go": return "go"
	case ".py": return "python"
	case ".js", ".jsx": return "javascript"
	case ".ts", ".tsx": return "typescript"
	case ".java": return "java"
	case ".cpp", ".cc", ".cxx": return "cpp"
	case ".c": return "c"
	case ".cs": return "csharp"
	case ".php": return "php"
	case ".rb": return "ruby"
	case ".rs": return "rust"
	case ".swift": return "swift"
	case ".kt": return "kotlin"
	case ".scala": return "scala"
	case ".sh", ".bash": return "bash"
	case ".ps1": return "powershell"
	case ".sql": return "sql"
	case ".html", ".htm": return "html"
	case ".css", ".scss", ".sass": return "css"
	default: return "unknown"
	}
}

func (e *Engine) compileStats(report *FinalReport) ReportStats {
	stats := ReportStats{
		FilesAnalyzed: report.Prepare.FilesIndexed,
		CandidatesFound: len(report.Candidates),
		ValidatedFindings: len(report.Validated),
		DedupedFindings: len(report.Deduped),
		ProvenFindings: len(report.Proven),
	}
	for _, f := range report.Deduped {
		switch strings.ToLower(f.Severity) {
		case "critical": stats.CriticalCount++
		case "high": stats.HighCount++
		case "medium": stats.MediumCount++
		case "low": stats.LowCount++
		}
	}
	return stats
}

// ============================================================================
// COMPATIBILITY WRAPPERS
// ============================================================================

// AnalyzeRepository is the old single-pass entry point (kept for compatibility)
func (e *Engine) AnalyzeRepository(ctx context.Context, owner, repo, ref string) (*AnalysisResult, error) {
	report, err := e.RunCAHPipeline(ctx, owner, repo, ref)
	if err != nil {
		return nil, err
	}

	// Convert to old format
	result := &AnalysisResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Commit:       ref,
		AnalysisTime: time.Duration(report.TotalTimeMs) * time.Millisecond,
		FilesAnalyzed: report.Stats.FilesAnalyzed,
		IssuesFound:  len(report.Deduped),
	}
	for _, f := range report.Deduped {
		result.Issues = append(result.Issues, ai.CodeIssue{
			Severity:   f.Severity,
			Category:   "security",
			Title:      f.Title,
			Description: f.Description,
			Confidence: f.Confidence,
		})
	}
	return result, nil
}

// AnalyzePullRequest analyzes a single PR using CAH pipeline
func (e *Engine) AnalyzePullRequest(ctx context.Context, owner, repo string, prNumber int) (*AnalysisResult, error) {
	pr, err := e.giteaClient.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	report, err := e.RunCAHPipeline(ctx, owner, repo, pr.HeadBranch)
	if err != nil {
		return nil, err
	}

	result := &AnalysisResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Commit:       fmt.Sprintf("PR #%d", prNumber),
		AnalysisTime: time.Duration(report.TotalTimeMs) * time.Millisecond,
		FilesAnalyzed: report.Stats.FilesAnalyzed,
		IssuesFound:  len(report.Deduped),
	}
	for _, f := range report.Deduped {
		result.Issues = append(result.Issues, ai.CodeIssue{
			Severity:   f.Severity,
			Category:   "security",
			Title:      f.Title,
			Description: f.Description,
			Confidence: f.Confidence,
		})
	}
	return result, nil
}
