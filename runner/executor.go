package runner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/customrules"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/models"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

// ExecuteInput configures a deterministic workspace scan on a runner host.
type ExecuteInput struct {
	WorkspaceDir string
	ScannerCfg   scanners.Config
	HealthCfg    health.Config
	GraphCfg     graph.Config
	SkipPatterns []string
	Logger       *logrus.Logger
}

// ExecuteWorkspaceScan runs allowed deterministic tasks for a JobSpec on a local checkout.
func ExecuteWorkspaceScan(ctx context.Context, spec JobSpec, in ExecuteInput) (JobResult, error) {
	started := time.Now().UTC()
	result := JobResult{
		Version:   ContractVersion,
		JobID:     spec.JobID,
		ScanID:    spec.ScanID,
		Status:    JobStatusCompleted,
		StartedAt: started,
		WorkspaceMeta: models.WorkspaceMeta{
			ModeUsed: scanners.WorkspaceModeArchive,
			RefUsed:  spec.Ref,
		},
	}

	if spec.ForbiddenTasks == nil {
		spec.ForbiddenTasks = ForbiddenTasks
	}
	if in.WorkspaceDir == "" {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, "workspace directory required")
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}

	files, entries, err := collectWorkspaceFiles(in.WorkspaceDir, in.SkipPatterns, spec.Limits.MaxFiles)
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, err.Error())
		result.FinishedAt = time.Now().UTC()
		return result, nil
	}
	result.FilesAnalyzed = len(files)

	policy := spec.EffectiveSettings
	depth := policy.AnalysisDepth
	if depth <= 0 {
		depth = 2
	}

	var candidates []models.CandidateFinding

	if depth >= 1 && taskAllowed(spec, "scanners") {
		static := analyzers.RunStaticAnalysis(files, true, true)
		candidates = append(candidates, static...)
	}

	// Apply the same declarative repository rules used by core API scans.
	// Invalid policies block the job instead of producing false clean results.
	for _, f := range files {
		if f.Path != customrules.Filename {
			continue
		}
		inputs := make([]customrules.File, 0, len(files))
		for _, source := range files {
			inputs = append(inputs, customrules.File{Path: source.Path, Content: source.Content})
		}
		matches, ruleErr := customrules.Scan(ctx, []byte(f.Content), inputs)
		if ruleErr != nil {
			result.Status = JobStatusFailed
			result.Errors = append(result.Errors, "custom rules: "+ruleErr.Error())
			result.FinishedAt = time.Now().UTC()
			return result, nil
		}
		for _, match := range matches {
			candidates = append(candidates, models.CandidateFinding{
				ID: match.RuleID, Hypothesis: match.Title, Severity: match.Severity, Confidence: 0.80,
				AuditorType: "custom_rule", Category: match.Category, File: match.File, Line: match.Line,
				Evidence: models.Evidence{ASTNode: match.Description},
			})
		}
		customStatus := scanners.StatusClean
		if len(matches) > 0 {
			customStatus = scanners.StatusFound
		}
		result.ScannerResults = append(result.ScannerResults, ScannerResultDTO{Scanner: "custom_rules", Status: string(customStatus), FindingsCount: len(matches)})
		break
	}

	if depth >= 2 && taskAllowed(spec, "health") && in.HealthCfg.Enabled {
		healthFiles := make([]health.FileInput, 0, len(files))
		for _, f := range files {
			healthFiles = append(healthFiles, health.FileInput{Path: f.Path, Content: f.Content, Language: f.Language})
		}
		for _, hf := range health.Run(health.RunInput{Files: healthFiles}, in.HealthCfg, in.SkipPatterns) {
			candidates = append(candidates, models.CandidateFinding{
				ID: hf.RuleID, Hypothesis: hf.Title, Severity: hf.Severity, Confidence: hf.Confidence,
				AuditorType: hf.Source, Category: hf.Category, File: hf.File, Line: hf.Line,
				Evidence: models.Evidence{Code: hf.Evidence},
			})
		}
	}

	if depth >= 2 && taskAllowed(spec, "scanners") {
		summary := scanners.RunAll(ctx, in.Logger, in.WorkspaceDir, entries, in.ScannerCfg, true, true)
		for _, sr := range summary.Results {
			result.ScannerResults = append(result.ScannerResults, ScannerResultDTO{
				Scanner: sr.Scanner, Status: string(sr.Status),
				FindingsCount: len(sr.Findings), Detail: sr.Detail,
			})
		}
		for _, f := range summary.Candidates() {
			candidates = append(candidates, f.ToCandidateFinding())
		}
	}

	repoFullName := spec.Repository.FullName
	for _, c := range candidates {
		issue := candidateToFinding(repoFullName, c)
		result.Findings = append(result.Findings, issue)
	}

	if depth >= 2 && taskAllowed(spec, "graph") && in.GraphCfg.Enabled {
		graphFiles := make([]graph.FileInput, 0, len(files))
		allPaths := make([]string, 0, len(files))
		for _, f := range files {
			graphFiles = append(graphFiles, graph.FileInput{Path: f.Path, Content: f.Content, Language: f.Language})
			allPaths = append(allPaths, f.Path)
		}
		overlays := graphOverlaysFromFindings(result.Findings)
		g, graphFindings := graph.Build(ctx, graph.BuildInput{
			RepositoryID: spec.Repository.FullName,
			ScanID:       spec.ScanID,
			Files:        graphFiles,
			AllPaths:     allPaths,
			Findings:     overlays,
		}, in.GraphCfg, in.SkipPatterns)
		result.Graph = &g
		for _, gf := range graphFindings {
			result.Findings = append(result.Findings, FindingResult{
				Fingerprint: issues.ComputeFingerprint(issues.FingerprintInput{
					Repository: repoFullName, RuleID: gf.RuleID, File: gf.File, Line: gf.Line,
					Category: gf.Category, Source: gf.Source,
				}),
				Category: gf.Category, Severity: gf.Severity, Confidence: gf.Confidence,
				Source: gf.Source, RuleID: gf.RuleID, File: gf.File, Line: gf.Line,
				Title: gf.Title, Description: gf.Description, CodeSnippet: gf.Evidence,
			})
		}
	}

	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func taskAllowed(spec JobSpec, task string) bool {
	for _, t := range spec.AllowedTasks {
		if t == task {
			return true
		}
	}
	return false
}

func collectWorkspaceFiles(root string, skipPatterns []string, maxFiles int) ([]analyzers.FileContent, []scanners.FileEntry, error) {
	if maxFiles <= 0 {
		maxFiles = 5000
	}
	var files []analyzers.FileContent
	var entries []scanners.FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan coverage incomplete: walk failed: %w", err)
		}
		if d.IsDir() {
			if shouldSkipDir(path, root, skipPatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if shouldSkipPath(rel, skipPatterns) || skipGeneratedNonSource(rel) {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("scan coverage incomplete: symlink %s requires explicit review", rel)
		}
		if _, err := scanners.ValidateWorkspacePath(root, rel); err != nil {
			return fmt.Errorf("scan coverage incomplete: unsafe path %s: %w", rel, err)
		}
		if len(files) >= maxFiles {
			return fmt.Errorf("scan coverage incomplete: source file count exceeds %d", maxFiles)
		}
		info, statErr := d.Info()
		if statErr != nil {
			return fmt.Errorf("scan coverage incomplete: cannot stat %s: %w", rel, statErr)
		}
		if info.Size() > customrules.MaxFileBytes {
			return fmt.Errorf("scan coverage incomplete: %s exceeds %d-byte worker file budget", rel, customrules.MaxFileBytes)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("scan coverage incomplete: cannot read %s: %w", rel, err)
		}
		text := string(content) // share underlying string for both scanner representations
		files = append(files, analyzers.FileContent{
			Path: rel, Content: text, Language: detectLanguage(rel),
		})
		entries = append(entries, scanners.FileEntry{Path: rel, Content: text})
		return nil
	})
	return files, entries, err
}

// Generated SBOM outputs and binary media are not source to analyze as text.
// Dedicated SBOM, secret, and container scanners cover their respective formats.
func skipGeneratedNonSource(rel string) bool {
	rel = strings.ToLower(filepath.ToSlash(rel))
	if strings.HasPrefix(rel, "docs/release/sbom/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".pdf", ".woff", ".woff2", ".ttf", ".otf", ".eot", ".zip", ".gz", ".xz", ".7z", ".tar", ".mp4", ".mp3", ".wav", ".bin", ".wasm", ".exe", ".dll", ".so", ".a", ".o", ".class", ".jar", ".pyc":
		return true
	}
	return false
}

func shouldSkipDir(path, root string, patterns []string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return shouldSkipPath(rel, patterns) || rel == "vendor" || rel == "node_modules" || rel == ".git"
}

func shouldSkipPath(path string, patterns []string) bool {
	path = strings.ToLower(path)
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" && strings.Contains(path, p) {
			return true
		}
	}
	return false
}

func detectLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".js", ".jsx", ".ts", ".tsx":
		return "javascript"
	case ".py":
		return "python"
	default:
		return ""
	}
}

func candidateToFinding(repo string, c models.CandidateFinding) FindingResult {
	line := c.Line
	file := c.File
	title := c.Hypothesis
	if title == "" {
		title = c.ID
	}
	fp := issues.ComputeFingerprint(issues.FingerprintInput{
		Repository: repo, RuleID: c.ID, File: file, Line: line,
		Category: c.Category, Source: c.AuditorType,
	})
	return FindingResult{
		Fingerprint: fp, Category: c.Category, Severity: c.Severity, Confidence: c.Confidence,
		Source: c.AuditorType, RuleID: c.ID, File: file, Line: line,
		Title: title, CodeSnippet: c.Evidence.Code,
	}
}

func graphOverlaysFromFindings(findings []FindingResult) []graph.FindingOverlay {
	out := make([]graph.FindingOverlay, 0, len(findings))
	for _, f := range findings {
		out = append(out, graph.FindingOverlay{
			ID: f.RuleID, File: f.File, Line: f.Line, Severity: f.Severity,
			Category: f.Category, Source: f.Source, RuleID: f.RuleID, Title: f.Title,
			Confidence: f.Confidence,
		})
	}
	return out
}
