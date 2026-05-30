package analyzers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/gitea"
	"git.commsnet.org/commstech/bugbot/models"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CONFIG & RESULT TYPES (kept for compatibility)
// ============================================================================

// Config holds analyzer configuration
type Config struct {
	MaxFileSize     int64
	AnalysisDepth   int
	EnableSecurity  bool
	EnableQuality   bool
	SkipPatterns    []string
	LanguageMapping map[string]string
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
	Repository    string
	Commit        string
	AnalysisTime  time.Duration
	FilesAnalyzed int
	IssuesFound   int
	Issues        []ai.CodeIssue
	Suggestions   []CodeSuggestion
	OverallScore  float64
	Errors        []string
}

// ============================================================================
// STAGE RESULT TYPES (aliases for models package)
// ============================================================================

type PrepareReport = models.PrepareReport
type EntryPoint = models.EntryPoint
type AttackSurfaceEntry = models.AttackSurfaceEntry
type TrustBoundary = models.TrustBoundary
type VulnContext = models.VulnContext
type CandidateFinding = models.CandidateFinding
type Evidence = models.Evidence
type Reachability = models.Reachability
type ValidatedFinding = models.ValidatedFinding
type DebateResult = models.DebateResult
type DedupedFinding = models.DedupedFinding
type ProvenFinding = models.ProvenFinding
type ProofOfConcept = models.ProofOfConcept

// FinalReport is the complete Bugbot report
type FinalReport struct {
	Repository  string
	Commit      string
	GeneratedAt time.Time
	TotalTimeMs int64
	Stages      []string // which stages completed

	Prepare    *PrepareReport
	Candidates []CandidateFinding
	Validated  []ValidatedFinding
	Deduped    []DedupedFinding
	Proven     []ProvenFinding

	Stats ReportStats
}

// ReportStats are summary statistics
type ReportStats struct {
	FilesAnalyzed     int
	CandidatesFound   int
	ValidatedFindings int
	DedupedFindings   int
	ProvenFindings    int
	CriticalCount     int
	HighCount         int
	MediumCount       int
	LowCount          int
}

// ============================================================================
// ENGINE - CAH PIPELINE ORCHESTRATOR
// ============================================================================

// Engine coordinates the CAH multi-stage analysis pipeline
type Engine struct {
	giteaClient *gitea.Client
	aiClient    *ai.Client
	logger      *logrus.Logger
	config      *Config
}

// NewEngine creates a new CAH-pipeline analysis engine
func NewEngine(giteaClient *gitea.Client, aiClient *ai.Client, config *Config, logger *logrus.Logger) *Engine {
	return &Engine{
		giteaClient: giteaClient,
		aiClient:    aiClient,
		logger:      logger,
		config:      config,
	}
}

// AnalysisOptions controls scoped vs full-repository analysis.
type AnalysisOptions struct {
	FilePaths []string // empty = scan entire repository
}

// RunCAHPipeline runs the full 5-stage CAH pipeline on a repository ref.
func (e *Engine) RunCAHPipeline(ctx context.Context, owner, repo, ref string) (*FinalReport, error) {
	return e.RunCAHPipelineWithOptions(ctx, owner, repo, ref, nil)
}

// RunCAHPipelineWithOptions runs the CAH pipeline, optionally limited to filePaths.
func (e *Engine) RunCAHPipelineWithOptions(ctx context.Context, owner, repo, ref string, opts *AnalysisOptions) (*FinalReport, error) {
	var filePaths []string
	if opts != nil {
		filePaths = opts.FilePaths
	}

	startTime := time.Now()
	report := &FinalReport{
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		Commit:      ref,
		GeneratedAt: time.Now(),
		Stages:      []string{},
	}

	if len(filePaths) > 0 {
		e.logger.Infof("[CAH:PIPELINE] Scoped analysis on %d changed file(s)", len(filePaths))
	}

	// Stage 1: PREPARE
	e.logger.Info("[CAH:PREPARE] Starting preparation phase...")
	pStart := time.Now()
	prepareReport, err := e.Prepare(ctx, owner, repo, ref, filePaths)
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

// Prepare maps the repository attack surface, optionally scoped to targetFiles.
func (e *Engine) Prepare(ctx context.Context, owner, repo, ref string, targetFiles []string) (*PrepareReport, error) {
	report := &PrepareReport{
		Repository:      fmt.Sprintf("%s/%s", owner, repo),
		Commit:          ref,
		Languages:       make(map[string]int),
		EntryPoints:     []EntryPoint{},
		AttackSurface:   []AttackSurfaceEntry{},
		TrustBoundaries: []TrustBoundary{},
		RecentVulns:     []VulnContext{},
		TargetFiles:     targetFiles,
	}

	files, err := e.resolveAnalyzableFiles(ctx, owner, repo, ref, targetFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve files: %w", err)
	}

	report.FilesFound = len(files)

	for _, f := range files {
		report.FilesIndexed++
		lang := e.detectLanguage(f.Path, "")
		report.Languages[lang]++
	}

	// Use AI to identify entry points, attack surface, and trust boundaries
	// by analyzing all files together (Prepare stage gets the full picture)
	e.logger.Info("[CAH:PREPARE] Running attack surface analysis...")

	// Build a context summary for the AI
	contextSummary := e.buildPrepareContext(files)

	// Call AI to identify entry points and attack surface
	surfaceFindings, err := e.aiClient.AnalyzeAttackSurface(ctx, &ai.AttackSurfaceRequest{
		RepositoryName: report.Repository,
		Files:          contextSummary,
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
	AuditorSQL    AuditorType = "sql"
	AuditorXSS    AuditorType = "xss"
	AuditorAuth   AuditorType = "auth"
	AuditorInject AuditorType = "injection"
	AuditorCrypto AuditorType = "crypto"
	AuditorRace   AuditorType = "race"
	AuditorMemory AuditorType = "memory"
	AuditorConfig AuditorType = "config"
)

// Scan runs deterministic checks first, then LLM auditors on flagged files.
func (e *Engine) Scan(ctx context.Context, prepare *PrepareReport) ([]CandidateFinding, error) {
	owner, repo := splitRepository(prepare.Repository)

	analyzableFiles, err := e.resolveAnalyzableFiles(ctx, owner, repo, prepare.Commit, prepare.TargetFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve files: %w", err)
	}

	fileContents, err := e.fetchFileContents(ctx, owner, repo, prepare.Commit, analyzableFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file contents: %w", err)
	}

	var allCandidates []CandidateFinding

	// Stage 2a: deterministic static analysis (no LLM tokens)
	if e.config.EnableSecurity || e.config.EnableQuality {
		staticFindings := RunStaticAnalysis(fileContents, e.config.EnableSecurity, e.config.EnableQuality)
		for _, f := range staticFindings {
			allCandidates = append(allCandidates, CandidateFinding(f))
		}
		e.logger.Infof("[CAH:SCAN] Static analysis found %d candidate(s)", len(staticFindings))
	}

	// Stage 2b: LLM auditors — only when security scanning is enabled
	if !e.config.EnableSecurity {
		return allCandidates, nil
	}

	llmTargets := e.selectLLMTargetFiles(fileContents, allCandidates)
	if len(llmTargets) == 0 {
		e.logger.Info("[CAH:SCAN] No files selected for LLM audit")
		return allCandidates, nil
	}

	e.logger.Infof("[CAH:SCAN] Running LLM auditors on %d file(s)", len(llmTargets))

	type result struct {
		findings []CandidateFinding
		err      error
		auditor  AuditorType
	}

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

	results := make(chan result, len(auditors))

	for _, auditor := range auditors {
		go func(a AuditorType, prompt string) {
			findings, err := e.runAuditor(ctx, a, prompt, prepare, llmTargets)
			results <- result{findings: findings, err: err, auditor: a}
		}(auditor.auditorType, auditor.promptType)
	}

	for i := 0; i < len(auditors); i++ {
		r := <-results
		if r.err != nil {
			e.logger.Warnf("[CAH:SCAN] Auditor %s failed: %v", r.auditor, r.err)
			continue
		}
		allCandidates = append(allCandidates, r.findings...)
	}

	return allCandidates, nil
}

// selectLLMTargetFiles limits LLM usage to files flagged by static analysis when possible.
func (e *Engine) selectLLMTargetFiles(allFiles []FileContent, staticCandidates []CandidateFinding) []FileContent {
	if len(staticCandidates) == 0 {
		return allFiles
	}

	flagged := make(map[string]bool, len(staticCandidates))
	for _, c := range staticCandidates {
		if c.File != "" {
			flagged[c.File] = true
		}
	}

	var targets []FileContent
	for _, file := range allFiles {
		if flagged[file.Path] {
			targets = append(targets, file)
		}
	}
	return targets
}

func (e *Engine) fetchFileContents(ctx context.Context, owner, repo, ref string, files []gitea.RepositoryContent) ([]FileContent, error) {
	var contents []FileContent

	for _, file := range files {
		if file.Type != "" && file.Type != "file" {
			continue
		}
		if file.Size > 0 && file.Size > e.config.MaxFileSize {
			e.logger.Debugf("Skipping oversized file %s (%d bytes)", file.Path, file.Size)
			continue
		}

		content, err := e.giteaClient.GetFileContent(ctx, owner, repo, ref, file.Path)
		if err != nil {
			e.logger.Warnf("Failed to fetch %s: %v", file.Path, err)
			continue
		}
		if int64(len(content)) > e.config.MaxFileSize {
			e.logger.Debugf("Skipping oversized content for %s", file.Path)
			continue
		}

		contents = append(contents, FileContent{
			Path:     file.Path,
			Content:  content,
			Language: e.detectLanguage(file.Path, content),
		})
	}

	return contents, nil
}

// runAuditor runs a single auditor agent across target files with content.
func (e *Engine) runAuditor(ctx context.Context, auditorType AuditorType, vulnClass string,
	prepare *PrepareReport, files []FileContent) ([]CandidateFinding, error) {

	e.logger.Infof("[CAH:SCAN] Running %s auditor on %d files", auditorType, len(files))

	var candidates []CandidateFinding

	batchSize := 5
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[i:end]

		repoFiles := make([]gitea.RepositoryContent, len(batch))
		aiFiles := make([]ai.FileContent, len(batch))
		for j, f := range batch {
			repoFiles[j] = gitea.RepositoryContent{
				Name: filepath.Base(f.Path),
				Path: f.Path,
				Type: "file",
			}
			aiFiles[j] = ai.FileContent{
				Path:     f.Path,
				Content:  f.Content,
				Language: f.Language,
			}
		}

		req := &ai.AuditorRequest{
			RepositoryName:     prepare.Repository,
			VulnerabilityClass: vulnClass,
			Files:              repoFiles,
			FileContents:       aiFiles,
			AttackSurface:      prepare.AttackSurface,
			AuditorType:        string(auditorType),
		}

		resp, err := e.aiClient.RunAuditor(ctx, req)
		if err != nil {
			e.logger.Warnf("[CAH:SCAN] %s auditor batch failed: %v", auditorType, err)
			continue
		}

		for _, f := range resp.Findings {
			candidates = append(candidates, CandidateFinding{
				ID:           fmt.Sprintf("%s-%s-%d", auditorType, vulnClass, len(candidates)+1),
				Hypothesis:   f.Hypothesis,
				Evidence:     Evidence{Code: f.CodeSnippet, CallChain: f.CallChain},
				Reachability: Reachability{FromEntryPoint: true},
				Severity:     f.Severity,
				Confidence:   f.Confidence,
				AuditorType:  string(auditorType),
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
				v, _ := e.validateOne(ctx, cand)
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
	// High-confidence static findings skip LLM debate (token savings)
	if candidate.AuditorType == "static" && candidate.Confidence >= 0.9 {
		return &ValidatedFinding{
			CandidateFinding: candidate,
			DebateResult: DebateResult{
				AdvocateConfidence: candidate.Confidence,
				CounselConfidence:  0.1,
				Outcome:            "validated",
			},
		}, nil
	}

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
			Finding: candidate,
			Role:    "advocate",
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
			Finding: candidate,
			Role:    "counsel",
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
			ID:          best.ID,
			Severity:    best.Severity,
			Category:    "security",
			Title:       best.Hypothesis,
			Description: best.Hypothesis,
			Files:       files,
			Lines:       lines,
			Evidence:    best.Evidence,
			Confidence:  best.Confidence,
			DedupGroup:  fmt.Sprintf("%s:%d", best.File, best.Line/10*10),
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
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".sh", ".bash":
		return "bash"
	case ".ps1":
		return "powershell"
	case ".sql":
		return "sql"
	case ".html", ".htm":
		return "html"
	case ".css", ".scss", ".sass":
		return "css"
	default:
		return "unknown"
	}
}

func (e *Engine) compileStats(report *FinalReport) ReportStats {
	stats := ReportStats{
		FilesAnalyzed:     report.Prepare.FilesIndexed,
		CandidatesFound:   len(report.Candidates),
		ValidatedFindings: len(report.Validated),
		DedupedFindings:   len(report.Deduped),
		ProvenFindings:    len(report.Proven),
	}
	for _, f := range report.Deduped {
		switch strings.ToLower(f.Severity) {
		case "critical":
			stats.CriticalCount++
		case "high":
			stats.HighCount++
		case "medium":
			stats.MediumCount++
		case "low":
			stats.LowCount++
		}
	}
	return stats
}

// ============================================================================
// COMPATIBILITY WRAPPERS
// ============================================================================

// AnalyzeRepository runs a full-repository CAH scan (manual/API use).
func (e *Engine) AnalyzeRepository(ctx context.Context, owner, repo, ref string) (*AnalysisResult, error) {
	return e.analysisResultFromReport(ctx, owner, repo, ref, "", nil)
}

// AnalyzeChangedFiles runs CAH on a specific set of paths (push webhooks).
func (e *Engine) AnalyzeChangedFiles(ctx context.Context, owner, repo, ref string, filePaths []string) (*AnalysisResult, error) {
	if len(filePaths) == 0 {
		e.logger.Info("No changed files to analyze")
		return &AnalysisResult{
			Repository: fmt.Sprintf("%s/%s", owner, repo),
			Commit:     ref,
		}, nil
	}
	return e.analysisResultFromReport(ctx, owner, repo, ref, ref, &AnalysisOptions{FilePaths: filePaths})
}

// AnalyzePullRequest analyzes only files changed in a pull request.
func (e *Engine) AnalyzePullRequest(ctx context.Context, owner, repo string, prNumber int) (*AnalysisResult, error) {
	pr, err := e.giteaClient.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	changedFiles, err := e.giteaClient.GetChangedFiles(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	e.logger.Infof("PR #%d: analyzing %d changed file(s) on branch %s", prNumber, len(changedFiles), pr.HeadBranch)

	return e.analysisResultFromReport(ctx, owner, repo, pr.HeadBranch, fmt.Sprintf("PR #%d", prNumber), &AnalysisOptions{FilePaths: changedFiles})
}

func (e *Engine) analysisResultFromReport(ctx context.Context, owner, repo, ref, commitLabel string, opts *AnalysisOptions) (*AnalysisResult, error) {
	report, err := e.RunCAHPipelineWithOptions(ctx, owner, repo, ref, opts)
	if err != nil {
		return nil, err
	}

	if commitLabel == "" {
		commitLabel = ref
	}

	result := &AnalysisResult{
		Repository:    fmt.Sprintf("%s/%s", owner, repo),
		Commit:        commitLabel,
		AnalysisTime:  time.Duration(report.TotalTimeMs) * time.Millisecond,
		FilesAnalyzed: report.Stats.FilesAnalyzed,
		IssuesFound:   len(report.Proven),
	}

	// Prefer proven findings (include PoC); fall back to deduped if prove stage empty
	findings := report.Proven
	if len(findings) == 0 {
		for _, f := range report.Deduped {
			findings = append(findings, ProvenFinding{DedupedFinding: f})
		}
	}

	for _, f := range findings {
		description := f.Description
		if description == "" {
			description = f.Title
		}

		poc := ""
		if f.ProofOfConcept.Command != "" {
			poc = f.ProofOfConcept.Command
			if f.ProofOfConcept.Explanation != "" {
				poc += "\n\n" + f.ProofOfConcept.Explanation
			}
		}

		line := 0
		if len(f.Lines) > 0 {
			line = f.Lines[0]
		}
		file := ""
		if len(f.Files) > 0 {
			file = f.Files[0]
		}

		result.Issues = append(result.Issues, ai.CodeIssue{
			Severity:       f.Severity,
			Category:       f.Category,
			Title:          f.Title,
			Description:    description,
			File:           file,
			LineNumber:     line,
			CodeSnippet:    f.Evidence.Code,
			ProofOfConcept: poc,
			Confidence:     f.Confidence,
		})
	}
	return result, nil
}

func (e *Engine) resolveAnalyzableFiles(ctx context.Context, owner, repo, ref string, targetFiles []string) ([]gitea.RepositoryContent, error) {
	if len(targetFiles) == 0 {
		allFiles, err := e.giteaClient.ListAllFiles(ctx, owner, repo, ref, "")
		if err != nil {
			return nil, err
		}
		var filtered []gitea.RepositoryContent
		for _, f := range allFiles {
			if e.shouldAnalyzeFile(f.Path) {
				filtered = append(filtered, f)
			}
		}
		return filtered, nil
	}

	var scoped []gitea.RepositoryContent
	for _, path := range targetFiles {
		if !e.shouldAnalyzeFile(path) {
			continue
		}
		scoped = append(scoped, gitea.RepositoryContent{
			Name: filepath.Base(path),
			Path: path,
			Type: "file",
		})
	}
	return scoped, nil
}

func splitRepository(fullName string) (owner, repo string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return fullName, ""
	}
	return parts[0], parts[1]
}
