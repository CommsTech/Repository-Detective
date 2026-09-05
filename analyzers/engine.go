package analyzers

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/forge"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/internal/scanid"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/sbom"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CONFIG & RESULT TYPES (kept for compatibility)
// ============================================================================

// Config holds analyzer configuration
type Config struct {
	MaxFileSize       int64
	AnalysisDepth     int
	EnableSecurity    bool
	EnableQuality     bool
	EnableLLMAuditors bool
	SkipPatterns      []string
	LanguageMapping   map[string]string
	Scanners          scanners.Config
	Workspace         scanners.WorkspaceConfig
	Health            health.Config
	Graph             graph.Config
	Reporting         profile.ReportingConfig
	FalsePositive     profile.FalsePositiveReductionConfig
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
	CommitSHA      string
	ScanID         string
	AnalysisTime   time.Duration
	FilesAnalyzed  int
	IssuesFound    int
	Issues         []ai.CodeIssue
	ScannerResults []scanners.RunResult
	Suggestions    []CodeSuggestion
	OverallScore          float64
	ScoreComplete         bool
	ScoreIncompleteReason string
	ScoreExplanation      string
	Errors                []string
	PolicySnapshot    *PolicySnapshot
	WorkspaceModeUsed string
	Graph             *graph.Graph
	RepoProfile       profile.RepoProfile
	Sbom              *sbom.Result
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

// FinalReport is the complete Repository Detective report
type FinalReport struct {
	ScanID      string
	Repository  string
	Commit      string
	GeneratedAt time.Time
	TotalTimeMs int64
	Stages      []string // which stages completed

	Prepare        *PrepareReport
	Candidates     []CandidateFinding
	Validated      []ValidatedFinding
	Deduped        []DedupedFinding
	Proven         []ProvenFinding
	ScannerResults []scanners.RunResult
	Workspace      models.WorkspaceMeta
	Graph          *graph.Graph
	RepoProfile    profile.RepoProfile
	Sbom           *sbom.Result

	Stats ReportStats
}

// ReportStats are summary statistics
type ReportStats struct {
	FilesAnalyzed     int
	CandidatesFound   int
	ValidatedFindings int
	DedupedFindings   int
	DedupClusters     int
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
	giteaClient  *gitea.Client
	githubClient forge.RepoClient
	aiClient     *ai.Client
	logger       *logrus.Logger
	config       *Config
}

// NewEngine creates a new CAH-pipeline analysis engine.
func NewEngine(giteaClient *gitea.Client, githubClient forge.RepoClient, aiClient *ai.Client, config *Config, logger *logrus.Logger) *Engine {
	return &Engine{
		giteaClient:  giteaClient,
		githubClient: githubClient,
		aiClient:     aiClient,
		logger:       logger,
		config:       config,
	}
}

func (e *Engine) repoClient(ctx context.Context) forge.RepoClient {
	if ForgeTypeFrom(ctx) == store.ForgeTypeGitHub && e.githubClient != nil {
		return e.githubClient
	}
	if e.giteaClient != nil {
		return e.giteaClient.AsForgeClient()
	}
	return e.githubClient
}

// AnalysisOptions controls scoped vs full-repository analysis.
type AnalysisOptions struct {
	FilePaths    []string // empty = scan entire repository
	CommitPinned bool
}

// RunCAHPipeline runs the full 5-stage CAH pipeline on a repository ref.
func (e *Engine) RunCAHPipeline(ctx context.Context, owner, repo, ref string) (*FinalReport, error) {
	return e.RunCAHPipelineWithOptions(ctx, owner, repo, ref, nil)
}

// RunCAHPipelineWithOptions runs the CAH pipeline, optionally limited to filePaths.
func (e *Engine) RunCAHPipelineWithOptions(ctx context.Context, owner, repo, ref string, opts *AnalysisOptions) (*FinalReport, error) {
	var filePaths []string
	commitPinned := false
	if opts != nil {
		filePaths = opts.FilePaths
		commitPinned = opts.CommitPinned
	}
	if !commitPinned {
		commitPinned = looksLikeCommitSHA(ref)
	}

	id := scanid.From(ctx)
	if id == "" {
		id = scanid.New()
	}
	ctx = scanid.With(ctx, id)
	log := e.scanLogger(ctx)

	startTime := time.Now()
	report := &FinalReport{
		ScanID:      id,
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		Commit:      ref,
		GeneratedAt: time.Now(),
		Stages:      []string{},
	}

	if len(filePaths) > 0 {
		log.Infof("[CAH:PIPELINE] Scoped analysis on %d changed file(s)", len(filePaths))
	}
	log.Info("[CAH:PIPELINE] Starting CAH pipeline")

	// Stage 1: PREPARE
	log.Info("[CAH:PREPARE] Starting preparation phase...")
	pStart := time.Now()
	prepareReport, err := e.Prepare(ctx, owner, repo, ref, filePaths, commitPinned)
	if err != nil {
		log.Errorf("[CAH:PREPARE] Failed: %v", err)
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	prepareReport.ScanTime = time.Since(pStart)
	report.Prepare = prepareReport
	report.RepoProfile = detectRepoProfile(ctx, e, owner, repo, ref, filePaths, prepareReport)
	report.Stages = append(report.Stages, "prepare")
	log.Infof("[CAH:PREPARE] Done in %v — found %d entry points, %d attack surface entries",
		prepareReport.ScanTime, len(prepareReport.EntryPoints), len(prepareReport.AttackSurface))

	// Stage 2: SCAN
	log.Info("[CAH:SCAN] Starting scan phase...")
	sStart := time.Now()
	candidates, scanSummary, workspaceMeta, repoGraph, sbomResult, err := e.Scan(ctx, prepareReport)
	if err != nil {
		log.Errorf("[CAH:SCAN] Failed: %v", err)
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	report.Candidates = candidates
	report.ScannerResults = profile.AnnotateScannerResults(scanSummary.Results, report.RepoProfile)
	report.Workspace = workspaceMeta
	report.Graph = repoGraph
	report.Sbom = sbomResult
	report.Stages = append(report.Stages, "scan")
	scanSummary.LogResults(e.logger, id)
	scanners.LogMeta(workspaceMeta, id, log)
	log.Infof("[CAH:SCAN] Done in %v — found %d candidates", time.Since(sStart), len(candidates))

	// Stage 3: VALIDATE
	log.Info("[CAH:VALIDATE] Starting validation phase...")
	vStart := time.Now()
	validated, err := e.Validate(ctx, candidates)
	if err != nil {
		log.Errorf("[CAH:VALIDATE] Failed: %v", err)
		return nil, fmt.Errorf("validate failed: %w", err)
	}
	report.Validated = validated
	report.Stages = append(report.Stages, "validate")
	log.Infof("[CAH:VALIDATE] Done in %v — %d/%d validated", time.Since(vStart), len(validated), len(candidates))

	// Stage 4: DEDUP
	log.Info("[CAH:DEDUP] Starting deduplication phase...")
	deduped := e.Dedup(validated)
	report.Deduped = deduped
	report.Stages = append(report.Stages, "dedup")
	log.Infof("[CAH:DEDUP] Done — %d unique findings from %d candidates", len(deduped), len(candidates))

	// Stage 5: PROVE
	log.Info("[CAH:PROVE] Starting proof generation phase...")
	pStart = time.Now()
	proven, err := e.Prove(ctx, deduped)
	if err != nil {
		log.Warnf("[CAH:PROVE] Some proofs failed: %v", err)
	}
	report.Proven = proven
	report.Stages = append(report.Stages, "prove")
	log.Infof("[CAH:PROVE] Done in %v — generated %d proofs", time.Since(pStart), len(proven))

	// Compile stats
	report.Stats = e.compileStats(report)
	report.TotalTimeMs = time.Since(startTime).Milliseconds()

	log.Infof("[CAH:PIPELINE] Complete in %v — %d critical, %d high, %d medium, %d low",
		time.Since(startTime), report.Stats.CriticalCount, report.Stats.HighCount,
		report.Stats.MediumCount, report.Stats.LowCount)

	return report, nil
}

// ============================================================================
// STAGE 1: PREPARE
// ============================================================================

// Prepare maps the repository attack surface, optionally scoped to targetFiles.
func (e *Engine) Prepare(ctx context.Context, owner, repo, ref string, targetFiles []string, commitPinned bool) (*PrepareReport, error) {
	report := &PrepareReport{
		Repository:      fmt.Sprintf("%s/%s", owner, repo),
		Commit:          ref,
		CommitPinned:    commitPinned,
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

	// Use AI to identify entry points, attack surface, and trust boundaries when LLM stages are enabled.
	if e.llmEnabledFor(ctx) {
		log := e.scanLogger(ctx)
		log.Info("[CAH:PREPARE] Running attack surface analysis...")

		contextSummary := e.buildPrepareContext(files)

		surfaceFindings, err := e.aiClient.AnalyzeAttackSurface(ctx, &ai.AttackSurfaceRequest{
			RepositoryName: report.Repository,
			Files:          contextSummary,
		})
		if err != nil {
			log.Warnf("[CAH:PREPARE] Attack surface analysis failed: %v", err)
		} else {
			report.EntryPoints = surfaceFindings.EntryPoints
			report.AttackSurface = surfaceFindings.AttackSurface
			report.TrustBoundaries = surfaceFindings.TrustBoundaries
		}
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
func (e *Engine) Scan(ctx context.Context, prepare *PrepareReport) ([]CandidateFinding, scanners.RunSummary, models.WorkspaceMeta, *graph.Graph, *sbom.Result, error) {
	log := e.scanLogger(ctx)
	summary := scanners.RunSummary{}
	workspaceMeta := models.WorkspaceMeta{
		ModeUsed:     scanners.WorkspaceModeAPI,
		RefUsed:      prepare.Commit,
		CommitPinned: prepare.CommitPinned,
	}
	var repoGraph *graph.Graph
	var sbomResult *sbom.Result
	owner, repo := splitRepository(prepare.Repository)

	analyzableFiles, err := e.resolveAnalyzableFiles(ctx, owner, repo, prepare.Commit, prepare.TargetFiles)
	if err != nil {
		return nil, summary, workspaceMeta, nil, nil, fmt.Errorf("failed to resolve files: %w", err)
	}

	fileContents, err := e.fetchFileContents(ctx, owner, repo, prepare.Commit, analyzableFiles)
	if err != nil {
		return nil, summary, workspaceMeta, nil, nil, fmt.Errorf("failed to fetch file contents: %w", err)
	}

	cfg := e.configFor(ctx)
	var allCandidates []CandidateFinding
	depth := cfg.AnalysisDepth
	if depth <= 0 {
		depth = 3
	}

	// Stage 2a: deterministic static analysis (depth >= 1)
	if depth >= 1 && (cfg.EnableSecurity || cfg.EnableQuality) {
		staticPaths := make([]string, 0, len(analyzableFiles))
		for _, f := range analyzableFiles {
			staticPaths = append(staticPaths, f.Path)
		}
		repoProfile := profile.DetectProfile(staticPaths)
		staticFindings := RunStaticAnalysisWithProfile(fileContents, cfg.EnableSecurity, cfg.EnableQuality, repoProfile)
		for _, f := range staticFindings {
			allCandidates = append(allCandidates, CandidateFinding(f))
		}
		summary.Results = append(summary.Results, scanners.DeterministicRunResult("static", len(staticFindings)))
		e.logger.Infof("[CAH:SCAN] Static analysis found %d candidate(s)", len(staticFindings))
	}

	// Stage 2a-health: deterministic repository health checks (depth >= 2)
	if depth >= 2 && cfg.Health.Enabled {
		allPaths := make([]string, 0, len(analyzableFiles))
		for _, f := range analyzableFiles {
			allPaths = append(allPaths, f.Path)
		}
		healthInputs := make([]health.FileContent, 0, len(fileContents))
		for _, f := range fileContents {
			healthInputs = append(healthInputs, health.FileContent{Path: f.Path, Content: f.Content, Language: f.Language})
		}
		healthFindings := health.Run(health.RunInput{
			Files:    health.InputsFromFileContents(healthInputs),
			AllPaths: allPaths,
		}, cfg.Health, cfg.SkipPatterns)
		for _, f := range health.ToCandidateFindings(healthFindings) {
			allCandidates = append(allCandidates, CandidateFinding(f))
		}
		summary.Results = append(summary.Results, scanners.DeterministicRunResult("health", len(healthFindings)))
		e.logger.Infof("[CAH:SCAN] Health checks found %d candidate(s)", len(healthFindings))
	}

	// Stage 2a-graph: repository map (depth >= 2)
	if depth >= 2 && cfg.Graph.Enabled {
		allPaths := make([]string, 0, len(analyzableFiles))
		for _, f := range analyzableFiles {
			allPaths = append(allPaths, f.Path)
		}
		graphFiles := make([]graph.FileInput, 0, len(fileContents))
		for _, f := range fileContents {
			graphFiles = append(graphFiles, graph.FileInput{Path: f.Path, Content: f.Content, Language: f.Language})
		}
		overlays := graphOverlaysFromCandidates(allCandidates)
		repoProfile := profile.DetectProfile(allPaths)
		g, graphFindings := graph.Build(ctx, graph.BuildInput{
			ScanID:   scanid.From(ctx),
			Files:    graphFiles,
			AllPaths: allPaths,
			Findings: overlays,
			Repo: graph.RepoContext{
				FileCount:        repoProfile.FileCount,
				Layout:           repoProfile.Layout,
				PrimaryEcosystem: repoProfile.PrimaryEcosystem,
				HomelabInfra:     profile.IsHomelabInfra(repoProfile),
			},
		}, cfg.Graph, cfg.SkipPatterns)
		repoGraph = &g
		for _, f := range graph.ToCandidateFindings(graphFindings) {
			allCandidates = append(allCandidates, CandidateFinding(f))
		}
		summary.Results = append(summary.Results, scanners.DeterministicRunResult("graph", len(graphFindings)))
		e.logger.Infof("[CAH:SCAN] Code graph generated %d nodes, %d edges, %d graph finding(s)",
			g.Metrics.NodeCount, g.Metrics.EdgeCount, len(graphFindings))
	}

	// Stage 2a-b: external scanners (depth >= 2)
	if depth >= 2 && (cfg.Scanners.EnableTrivy || cfg.Scanners.EnableGrype || cfg.Scanners.EnableGitleaks || cfg.Scanners.EnableSemgrep || cfg.Scanners.EnableGovulncheck || cfg.Scanners.EnableGosec || cfg.Scanners.EnableStaticcheck || cfg.Scanners.EnableHadolint || cfg.Scanners.EnableCheckov || cfg.Scanners.EnableLinters) {
		apiEntries := toScannerEntries(fileContents)
		if cfg.Workspace.NormalizedMode() != scanners.WorkspaceModeArchive {
			manifestFiles, err := e.fetchManifestContents(ctx, owner, repo, prepare.Commit)
			if err != nil {
				log.Warnf("[CAH:SCAN] Failed to fetch dependency manifests: %v", err)
			}
			apiEntries = toScannerEntries(mergeFileContents(fileContents, manifestFiles))
		}

		prepared, err := scanners.PrepareWorkspace(
			ctx,
			cfg.Workspace,
			e.repoClient(ctx),
			owner,
			repo,
			prepare.Commit,
			prepare.CommitPinned,
			apiEntries,
		)
		if err != nil {
			log.Warnf("[CAH:SCAN] Failed to prepare scanner workspace: %v", err)
			workspaceMeta.WorkspaceError = err.Error()
		} else {
			defer prepared.Cleanup()
			workspaceMeta = prepared.Meta
			deterministicResults := append([]scanners.RunResult(nil), summary.Results...)
			externalSummary := scanners.RunAll(ctx, e.logger, prepared.Dir, prepared.Entries, cfg.Scanners, cfg.EnableSecurity, cfg.EnableQuality)
			summary = mergeScannerRunSummaries(deterministicResults, externalSummary)
			repoProfile := profile.DetectProfile(allPathsFromEntries(prepared.Entries))
			summary.Results = profile.CalibrateRuffResults(summary.Results, repoProfile)
			for _, finding := range summary.Candidates() {
				allCandidates = append(allCandidates, finding.ToCandidateFinding())
			}
			log.Infof("[CAH:SCAN] External scanners found %d candidate(s) (workspace_mode=%s)", len(summary.Candidates()), workspaceMeta.ModeUsed)
			outDir := filepath.Join(prepared.Dir, ".rd-sbom")
			// SBOM runs after scanners; use a deadline independent of the analysis timeout so
			// a long scanner phase does not leave zero budget for cyclonedx/syft.
			sbomCtx, sbomCancel := context.WithTimeout(context.WithoutCancel(ctx), sbom.DefaultTimeout())
			res, sbErr := sbom.GenerateAndCheck(sbomCtx, prepared.Dir, outDir)
			sbomCancel()
			if sbErr == nil {
				copy := res
				if copy.Status == sbom.StatusCheckFailed && strings.TrimSpace(copy.Detail) == "" {
					copy.Detail = "SBOM generation or vulnerability check failed (no detail from tool)"
				}
				sbomResult = &copy
				log.Infof("[CAH:SCAN] SBOM status=%s packages=%d vulns=%d", res.Status, res.PackageCount, res.VulnCount)
			} else {
				log.Warnf("[CAH:SCAN] SBOM generation failed: %v", sbErr)
			}

			scopedScan := len(prepare.TargetFiles) > 0
			secretModes := scanners.ResolveSecretScanModes(cfg.Scanners, scopedScan, depth)
			if cfg.EnableSecurity && (secretModes.GitHistory || secretModes.RecentCommits || secretModes.ChangedFiles) {
				historyCtx, historyCancel := context.WithTimeout(context.WithoutCancel(ctx), scanners.GitleaksHistoryBudget(cfg.Scanners))
				for _, hr := range e.runGitHistorySecretScans(historyCtx, owner, repo, prepare.Commit, prepared.Dir, secretModes, cfg.Scanners) {
					summary.Results = append(summary.Results, hr)
					for _, finding := range hr.Findings {
						allCandidates = append(allCandidates, finding.ToCandidateFinding())
					}
				}
				historyCancel()
			}
		}
	}

	// Stage 2c: LLM auditors (depth >= 3)
	if !e.llmEnabledFor(ctx) || !e.configFor(ctx).EnableSecurity {
		return allCandidates, summary, workspaceMeta, repoGraph, sbomResult, nil
	}

	llmTargets := e.selectLLMTargetFiles(fileContents, allCandidates)
	if len(llmTargets) == 0 {
		log.Info("[CAH:SCAN] No files selected for LLM audit")
		return allCandidates, summary, workspaceMeta, repoGraph, sbomResult, nil
	}

	log.Infof("[CAH:SCAN] Running LLM auditors on %d file(s)", len(llmTargets))

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
			log.Warnf("[CAH:SCAN] Auditor %s failed: %v", r.auditor, r.err)
			continue
		}
		allCandidates = append(allCandidates, r.findings...)
	}

	return allCandidates, summary, workspaceMeta, repoGraph, sbomResult, nil
}

// mergeScannerRunSummaries keeps in-process deterministic stage results when external scanners run.
func mergeScannerRunSummaries(deterministic []scanners.RunResult, external scanners.RunSummary) scanners.RunSummary {
	external.Results = append(deterministic, external.Results...)
	return external
}

// selectLLMTargetFiles limits LLM usage to files flagged by deterministic checks when possible.
func (e *Engine) selectLLMTargetFiles(allFiles []FileContent, candidates []CandidateFinding) []FileContent {
	if len(candidates) == 0 {
		return allFiles
	}

	flagged := make(map[string]bool, len(candidates))
	for _, c := range candidates {
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

func (e *Engine) fetchManifestContents(ctx context.Context, owner, repo, ref string) ([]FileContent, error) {
	var manifests []FileContent
	for _, path := range scanners.ManifestPaths() {
		content, err := e.repoClient(ctx).GetFileContent(ctx, owner, repo, ref, path)
		if err != nil || content == "" {
			continue
		}
		manifests = append(manifests, FileContent{
			Path:     path,
			Content:  content,
			Language: e.detectLanguage(path, content),
		})
	}
	return manifests, nil
}

func mergeFileContents(primary, extra []FileContent) []FileContent {
	merged := make([]FileContent, 0, len(primary)+len(extra))
	seen := make(map[string]bool)
	for _, file := range primary {
		if seen[file.Path] {
			continue
		}
		seen[file.Path] = true
		merged = append(merged, file)
	}
	for _, file := range extra {
		if seen[file.Path] {
			continue
		}
		seen[file.Path] = true
		merged = append(merged, file)
	}
	return merged
}

func toScannerEntries(files []FileContent) []scanners.FileEntry {
	entries := make([]scanners.FileEntry, 0, len(files))
	for _, file := range files {
		entries = append(entries, scanners.FileEntry{
			Path:    file.Path,
			Content: file.Content,
		})
	}
	return entries
}

func isDeterministicAuditor(auditorType string) bool {
	if auditorType == "static" {
		return true
	}
	return scanners.IsDeterministicSource(auditorType)
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

		content, err := e.repoClient(ctx).GetFileContent(ctx, owner, repo, ref, file.Path)
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
	// Deterministic findings skip LLM debate (scanners, static analysis, health checks).
	if isDeterministicAuditor(candidate.AuditorType) {
		return &ValidatedFinding{
			CandidateFinding: candidate,
			DebateResult: DebateResult{
				AdvocateConfidence: candidate.Confidence,
				CounselConfidence:  0.1,
				Outcome:            "validated",
			},
		}, nil
	}

	if !e.llmEnabledFor(ctx) {
		return &ValidatedFinding{
			CandidateFinding: candidate,
			DebateResult: DebateResult{
				AdvocateConfidence: candidate.Confidence,
				CounselConfidence:  0.0,
				Outcome:            "downgraded",
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

// Dedup collapses semantically equivalent findings and assigns stable cluster IDs.
func (e *Engine) Dedup(candidates []ValidatedFinding) []DedupedFinding {
	if len(candidates) == 0 {
		return []DedupedFinding{}
	}

	groups := make(map[string][]ValidatedFinding)
	for _, c := range candidates {
		category := strings.TrimSpace(strings.ToLower(c.Category))
		if category == "" {
			category = strings.TrimSpace(strings.ToLower(string(c.AuditorType)))
		}
		if category == "" {
			category = "security"
		}
		// Dedup within the same file, line block, and vulnerability class — never merge secrets with SQLi, etc.
		key := fmt.Sprintf("%s:%d:%s", c.File, c.Line/10*10, category)
		groups[key] = append(groups[key], c)
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var deduped []DedupedFinding
	for clusterIndex, key := range keys {
		group := groups[key]
		best := group[0]
		for i := 1; i < len(group); i++ {
			if group[i].Confidence > best.Confidence {
				best = group[i]
			}
		}

		var files []string
		var lines []int
		var related []string
		for _, c := range group {
			files = append(files, c.File)
			lines = append(lines, c.Line)
			if c.Hypothesis != "" && c.Hypothesis != best.Hypothesis {
				related = append(related, c.Hypothesis)
			}
		}

		clusterID := fmt.Sprintf("cluster-%03d", clusterIndex)
		description := best.Hypothesis
		if best.AuditorType == "graph" && strings.TrimSpace(best.Evidence.Code) != "" {
			description = best.Evidence.Code
		} else if len(group) > 1 {
			description = fmt.Sprintf("%s\n\n**Dedup cluster `%s`** — merged %d related finding(s):\n",
				best.Hypothesis, clusterID, len(group))
			for _, item := range group {
				description += fmt.Sprintf("- %s (`%s:%d`, confidence %.2f)\n", item.Hypothesis, item.File, item.Line, item.Confidence)
			}
		}

		deduped = append(deduped, DedupedFinding{
			ID:          best.ID,
			Severity:    best.Severity,
			Category:    firstNonEmptyCategory(best.Category, "security"),
			Title:       best.Hypothesis,
			Description: description,
			AuditorType: best.AuditorType,
			Files:       files,
			Lines:       lines,
			Evidence:    best.Evidence,
			Confidence:  best.Confidence,
			DedupGroup:  key,
			ClusterID:   clusterID,
			Related:     related,
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
		if isDeterministicAuditor(f.AuditorType) {
			proven = append(proven, ProvenFinding{
				DedupedFinding: f,
				ProofOfConcept: ProofOfConcept{
					Type:        "scanner",
					Command:     f.Evidence.Code,
					Explanation: fmt.Sprintf("Deterministic finding from %s — no LLM PoC required", f.AuditorType),
				},
			})
			continue
		}

		if !e.llmEnabledFor(ctx) {
			proven = append(proven, ProvenFinding{DedupedFinding: f})
			continue
		}

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

func firstNonEmptyRuleID(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func packageNameFromFinding(f ProvenFinding) string {
	code := strings.TrimSpace(f.Evidence.Code)
	if code == "" {
		return ""
	}
	if idx := strings.Index(code, "@"); idx > 0 {
		return code[:idx]
	}
	return ""
}

func firstNonEmptyCategory(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "security"
}

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
	skipDirs := []string{"node_modules", "vendor", ".git", ".venv", "venv", "__pycache__", "build", "dist", "target"}
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
		DedupClusters:     len(report.Deduped),
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
		commitSHA := ""
		if looksLikeCommitSHA(ref) {
			commitSHA = ref
		}
		return &AnalysisResult{
			Repository: fmt.Sprintf("%s/%s", owner, repo),
			Commit:     ref,
			CommitSHA:  commitSHA,
		}, nil
	}
	return e.analysisResultFromReport(ctx, owner, repo, ref, ref, &AnalysisOptions{
		FilePaths:    filePaths,
		CommitPinned: looksLikeCommitSHA(ref),
	})
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

	ref, commitPinned := PullRequestRef(pr)
	if !commitPinned {
		e.logger.Warnf("PR #%d: archive ref is branch %q (not commit-pinned) — workspace may drift if branch moves", prNumber, ref)
	} else {
		e.logger.Infof("PR #%d: using commit-pinned ref %s", prNumber, ref)
	}

	e.logger.Infof("PR #%d: analyzing %d changed file(s)", prNumber, len(changedFiles))

	return e.analysisResultFromReport(ctx, owner, repo, ref, fmt.Sprintf("PR #%d", prNumber), &AnalysisOptions{
		FilePaths:    changedFiles,
		CommitPinned: commitPinned,
	})
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
		Repository:     fmt.Sprintf("%s/%s", owner, repo),
		Commit:         commitLabel,
		ScanID:         report.ScanID,
		AnalysisTime:   time.Duration(report.TotalTimeMs) * time.Millisecond,
		FilesAnalyzed:  report.Stats.FilesAnalyzed,
		IssuesFound:    len(report.Proven),
		ScannerResults: report.ScannerResults,
		WorkspaceModeUsed: report.Workspace.ModeUsed,
	}
	if report.Sbom != nil {
		copy := *report.Sbom
		result.Sbom = &copy
	}
	if report.Graph != nil {
		result.Graph = report.Graph
	}
	if policy, ok := ScanPolicyFromContext(ctx); ok {
		s := SnapshotFromPolicy(policy)
		result.PolicySnapshot = &s
	}
	if looksLikeCommitSHA(ref) {
		result.CommitSHA = ref
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

		snippet := f.Evidence.Code
		detailJSON := f.Evidence.ASTNode
		if f.AuditorType == "graph" {
			snippet = ""
		}

		result.Issues = append(result.Issues, ai.CodeIssue{
			Severity:       f.Severity,
			Category:       f.Category,
			Title:          f.Title,
			Description:    description,
			File:           file,
			LineNumber:     line,
			CodeSnippet:    snippet,
			ProofOfConcept: poc,
			Confidence:     f.Confidence,
			ClusterID:      f.ClusterID,
			Source:         f.AuditorType,
			RuleID:         firstNonEmptyRuleID(f.ID, f.ClusterID),
			ScanID:         report.ScanID,
			PackageName:    packageNameFromFinding(f),
			Evidence:       detailJSON,
		})
	}

	knownPaths := buildKnownPathSet(report)
	result.RepoProfile = report.RepoProfile
	issues.EnrichIssues(result.Repository, result.ScanID, result.Issues)
	reporting := e.config.Reporting
	if policy, ok := ScanPolicyFromContext(ctx); ok {
		reporting = profile.ReportingForScanProfile(reporting, policy.ScanProfile)
	}
	result.Issues = profile.NormalizeIssues(result.Issues, profile.NormalizeInput{
		Repository:    result.Repository,
		CommitSHA:     result.CommitSHA,
		ScanID:        result.ScanID,
		Profile:       report.RepoProfile,
		Reporting:     reporting,
		FalsePositive: e.config.FalsePositive,
		KnownPaths:    knownPaths,
	})
	score := ComputeScoreResult(result.Issues, ScoreInput{ScannerResults: report.ScannerResults})
	result.ScoreComplete = score.Complete
	result.ScoreIncompleteReason = score.IncompleteReason
	result.ScoreExplanation = score.Explanation
	result.OverallScore = ComputeOverallScoreWithInput(result.Issues, ScoreInput{ScannerResults: report.ScannerResults})
	return result, nil
}

func buildKnownPathSet(report *FinalReport) map[string]struct{} {
	paths := map[string]struct{}{}
	if report == nil || report.Prepare == nil {
		return paths
	}
	for _, c := range report.Candidates {
		if c.File != "" {
			paths[profile.NormalizePath(c.File)] = struct{}{}
		}
	}
	for _, sr := range report.ScannerResults {
		for _, f := range sr.Findings {
			if f.File != "" {
				paths[profile.NormalizePath(f.File)] = struct{}{}
			}
		}
	}
	return paths
}

func detectRepoProfile(ctx context.Context, e *Engine, owner, repo, ref string, targetFiles []string, prepare *PrepareReport) profile.RepoProfile {
	var paths []string
	if len(targetFiles) > 0 {
		paths = append(paths, targetFiles...)
	} else if client := e.repoClient(ctx); client != nil {
		resolvedRef, err := client.ResolveRef(ctx, owner, repo, ref)
		if err == nil {
			ref = resolvedRef
		}
		allFiles, err := client.ListAllFiles(ctx, owner, repo, ref, "")
		if err == nil {
			for _, f := range allFiles {
				paths = append(paths, f.Path)
			}
		}
	}
	return profile.DetectProfile(paths)
}

func (e *Engine) resolveAnalyzableFiles(ctx context.Context, owner, repo, ref string, targetFiles []string) ([]gitea.RepositoryContent, error) {
	if len(targetFiles) > 0 {
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

	client := e.repoClient(ctx)
	if client == nil {
		return nil, fmt.Errorf("no repository client configured for forge %s", ForgeTypeFrom(ctx))
	}
	{
		resolvedRef, err := client.ResolveRef(ctx, owner, repo, ref)
		if err != nil {
			return nil, err
		}
		ref = resolvedRef
		allFiles, err := client.ListAllFiles(ctx, owner, repo, ref, "")
		if err != nil {
			if strings.Contains(err.Error(), "content not found") {
				return []gitea.RepositoryContent{}, nil
			}
			return nil, err
		}
		var filtered []forge.RepositoryContent
		for _, f := range allFiles {
			if e.shouldAnalyzeFile(f.Path) {
				filtered = append(filtered, f)
			}
		}
		return forgeToGiteaFiles(filtered), nil
	}
}

func splitRepository(fullName string) (owner, repo string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return fullName, ""
	}
	return parts[0], parts[1]
}

func (e *Engine) scanLogger(ctx context.Context) *logrus.Entry {
	if id := scanid.From(ctx); id != "" {
		return e.logger.WithField("scan_id", id)
	}
	return logrus.NewEntry(e.logger)
}

func forgeToGiteaFiles(files []forge.RepositoryContent) []gitea.RepositoryContent {
	out := make([]gitea.RepositoryContent, len(files))
	for i, f := range files {
		out[i] = gitea.RepositoryContent{
			Name: f.Name, Path: f.Path, SHA: f.SHA, Size: f.Size,
			URL: f.URL, HTMLURL: f.HTMLURL, GitURL: f.GitURL,
			DownloadURL: f.DownloadURL, Type: f.Type,
			Content: f.Content, Encoding: f.Encoding,
		}
	}
	return out
}

func allPathsFromEntries(entries []scanners.FileEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Path != "" {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

func graphOverlaysFromCandidates(candidates []CandidateFinding) []graph.FindingOverlay {
	out := make([]graph.FindingOverlay, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, graph.FindingOverlay{
			ID: c.ID, File: c.File, Line: c.Line,
			Severity: c.Severity, Category: c.Category,
			Source: c.AuditorType, RuleID: c.ID, Title: c.Hypothesis,
			Confidence: c.Confidence,
		})
	}
	return out
}

func (e *Engine) runGitHistorySecretScans(
	ctx context.Context,
	owner, repo, ref, currentTreeDir string,
	modes scanners.SecretScanModes,
	cfg scanners.Config,
) []scanners.RunResult {
	cloneURL, err := e.resolveCloneURL(ctx, owner, repo)
	if err != nil || strings.TrimSpace(cloneURL) == "" {
		detail := "clone URL unavailable for git history secret scan"
		if err != nil {
			detail = err.Error()
		}
		e.logger.Warnf("[CAH:SCAN] git history secret scan skipped: %s", detail)
		return []scanners.RunResult{{
			Scanner: scanners.HistoryScannerName,
			Status:  scanners.StatusFailed,
			Detail:  detail,
		}}
	}

	scope := scanners.SecretScopeGitHistory
	maxCommits := 0
	if modes.ChangedFiles {
		scope = scanners.SecretScopeChangedFiles
		maxCommits = cfg.SecretScanRecentCommitsMax
	} else if modes.RecentCommits {
		scope = scanners.SecretScopeRecentCommits
		maxCommits = cfg.SecretScanRecentCommitsMax
	}
	if maxCommits <= 0 && cfg.SecretScanHistoryMaxCommits > 0 {
		maxCommits = cfg.SecretScanHistoryMaxCommits
	}
	if (modes.ChangedFiles || modes.RecentCommits) && maxCommits <= 0 {
		maxCommits = 50
	}

	gitWS, err := scanners.PrepareGitHistoryWorkspace(ctx, cloneURL, e.cloneToken(ctx), ref, maxCommits, cfg.SecretScanHistoryTimeoutSeconds)
	if err != nil {
		e.logger.Warnf("[CAH:SCAN] git history workspace: %v", err)
		return []scanners.RunResult{{
			Scanner: scanners.HistoryScannerName,
			Status:  scanners.StatusFailed,
			Detail:  err.Error(),
		}}
	}
	defer gitWS.Cleanup()

	hr := scanners.RunGitleaksGitHistory(ctx, e.logger, gitWS.Dir, cfg, scope, currentTreeDir)
	return []scanners.RunResult{hr}
}

// cloneToken returns the forge credential for the active context so private
// repositories can be cloned for history secret scanning. It returns an empty
// string when the forge client cannot supply one, leaving the clone anonymous.
func (e *Engine) cloneToken(ctx context.Context) string {
	if ForgeTypeFrom(ctx) == store.ForgeTypeGitHub {
		if provider, ok := e.githubClient.(interface{ Token() string }); ok {
			return provider.Token()
		}
		return ""
	}
	return e.giteaClient.Token()
}

func (e *Engine) resolveCloneURL(ctx context.Context, owner, repo string) (string, error) {
	client := e.repoClient(ctx)
	if client == nil {
		return "", fmt.Errorf("no forge client configured")
	}
	r, err := client.GetRepository(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(r.CloneURL) == "" {
		return "", fmt.Errorf("repository has no clone URL")
	}
	return r.CloneURL, nil
}
