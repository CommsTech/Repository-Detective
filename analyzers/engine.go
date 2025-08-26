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

// Engine coordinates the code analysis process
type Engine struct {
	giteaClient    *gitea.Client
	aiClient       *ai.OpenWebUIClient
	logger         *logrus.Logger
	config         *Config
}

// Config holds analysis engine configuration
type Config struct {
	MaxFileSize     int64
	AnalysisDepth   int
	EnableSecurity  bool
	EnableQuality   bool
	SkipPatterns    []string
	LanguageMapping map[string]string
}

// AnalysisResult represents the complete result of analyzing a repository
type AnalysisResult struct {
	Repository     string
	Commit         string
	AnalysisTime   time.Duration
	FilesAnalyzed  int
	IssuesFound    int
	Issues         []ai.CodeIssue
	Suggestions    []ai.CodeSuggestion
	OverallScore   float64
	Errors         []string
}

// NewEngine creates a new analysis engine
func NewEngine(giteaClient *gitea.Client, aiClient *ai.OpenWebUIClient, config *Config, logger *logrus.Logger) *Engine {
	return &Engine{
		giteaClient: giteaClient,
		aiClient:    aiClient,
		logger:      logger,
		config:      config,
	}
}

// AnalyzeRepository analyzes an entire repository for issues
func (e *Engine) AnalyzeRepository(ctx context.Context, owner, repo, ref string) (*AnalysisResult, error) {
	startTime := time.Now()
	e.logger.Infof("Starting repository analysis for %s/%s at ref %s", owner, repo, ref)

	result := &AnalysisResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Commit:       ref,
		AnalysisTime: 0,
		FilesAnalyzed: 0,
		IssuesFound:  0,
		Issues:       []ai.CodeIssue{},
		Suggestions:  []ai.CodeSuggestion{},
		OverallScore: 0.0,
		Errors:       []string{},
	}

	// Get repository information
	repository, err := e.giteaClient.GetRepository(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Analyze repository contents
	if err := e.analyzeRepositoryContents(ctx, owner, repo, ref, "", result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Analysis failed: %v", err))
		e.logger.Errorf("Repository analysis failed: %v", err)
	}

	result.AnalysisTime = time.Since(startTime)
	e.logger.Infof("Repository analysis completed in %v, analyzed %d files, found %d issues", 
		result.AnalysisTime, result.FilesAnalyzed, result.IssuesFound)

	return result, nil
}

// AnalyzePullRequest analyzes a pull request for issues
func (e *Engine) AnalyzePullRequest(ctx context.Context, owner, repo string, prNumber int) (*AnalysisResult, error) {
	startTime := time.Now()
	e.logger.Infof("Starting pull request analysis for %s/%s PR #%d", owner, repo, prNumber)

	result := &AnalysisResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Commit:       fmt.Sprintf("PR #%d", prNumber),
		AnalysisTime: 0,
		FilesAnalyzed: 0,
		IssuesFound:  0,
		Issues:       []ai.CodeIssue{},
		Suggestions:  []ai.CodeSuggestion{},
		OverallScore: 0.0,
		Errors:       []string{},
	}

	// Get pull request information
	pr, err := e.giteaClient.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Get changed files
	changedFiles, err := e.giteaClient.GetChangedFiles(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	e.logger.Infof("Analyzing %d changed files in pull request", len(changedFiles))

	// Analyze each changed file
	for _, filePath := range changedFiles {
		if err := e.analyzeFile(ctx, owner, repo, pr.HeadBranch, filePath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to analyze %s: %v", filePath, err))
			e.logger.Errorf("Failed to analyze file %s: %v", filePath, err)
		}
	}

	result.AnalysisTime = time.Since(startTime)
	e.logger.Infof("Pull request analysis completed in %v, analyzed %d files, found %d issues", 
		result.AnalysisTime, result.FilesAnalyzed, result.IssuesFound)

	return result, nil
}

// analyzeRepositoryContents recursively analyzes repository contents
func (e *Engine) analyzeRepositoryContents(ctx context.Context, owner, repo, ref, path string, result *AnalysisResult) error {
	e.logger.Debugf("Analyzing path: %s", path)

	// Get contents of current path
	contents, err := e.giteaClient.ListRepositoryContents(ctx, owner, repo, ref, path)
	if err != nil {
		return fmt.Errorf("failed to list contents of %s: %w", path, err)
	}

	// Analyze each item
	for _, content := range contents {
		fullPath := filepath.Join(path, content.Name)

		if content.Type == "dir" {
			// Recursively analyze subdirectories
			if e.shouldAnalyzeDirectory(fullPath) {
				if err := e.analyzeRepositoryContents(ctx, owner, repo, ref, fullPath, result); err != nil {
					e.logger.Warnf("Failed to analyze directory %s: %v", fullPath, err)
				}
			}
		} else if content.Type == "file" {
			// Analyze file
			if e.shouldAnalyzeFile(fullPath) {
				if err := e.analyzeFile(ctx, owner, repo, ref, fullPath, result); err != nil {
					e.logger.Warnf("Failed to analyze file %s: %v", fullPath, err)
				}
			}
		}
	}

	return nil
}

// analyzeFile analyzes a single file for issues
func (e *Engine) analyzeFile(ctx context.Context, owner, repo, ref, filePath string, result *AnalysisResult) error {
	// Check if file should be skipped
	if !e.shouldAnalyzeFile(filePath) {
		return nil
	}

	// Check file size
	if e.config.MaxFileSize > 0 {
		content, err := e.giteaClient.GetRepositoryContent(ctx, owner, repo, ref, filePath)
		if err != nil {
			return fmt.Errorf("failed to get file content: %w", err)
		}

		if content.Size > e.config.MaxFileSize {
			e.logger.Debugf("Skipping large file %s (size: %d bytes)", filePath, content.Size)
			return nil
		}
	}

	// Get file content
	content, err := e.giteaClient.GetFileContent(ctx, owner, repo, ref, filePath)
	if err != nil {
		return fmt.Errorf("failed to get file content: %w", err)
	}

	// Determine language
	language := e.detectLanguage(filePath, content)

	// Create analysis request
	analysisReq := &ai.CodeAnalysisRequest{
		RepositoryName: fmt.Sprintf("%s/%s", owner, repo),
		FilePath:       filePath,
		CodeContent:    content,
		Language:       language,
		Context:        fmt.Sprintf("Repository: %s/%s, Branch: %s", owner, repo, ref),
		AnalysisType:   "all",
	}

	// Perform AI analysis
	analysisResult, err := e.aiClient.AnalyzeCode(ctx, analysisReq)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Update result
	result.FilesAnalyzed++
	result.IssuesFound += len(analysisResult.Issues)
	result.Issues = append(result.Issues, analysisResult.Issues...)
	result.Suggestions = append(result.Suggestions, analysisResult.Suggestions...)

	// Update overall score (weighted average)
	if result.OverallScore == 0.0 {
		result.OverallScore = analysisResult.OverallScore
	} else {
		result.OverallScore = (result.OverallScore + analysisResult.OverallScore) / 2.0
	}

	e.logger.Debugf("Analyzed file %s, found %d issues", filePath, len(analysisResult.Issues))

	return nil
}

// shouldAnalyzeFile determines if a file should be analyzed
func (e *Engine) shouldAnalyzeFile(filePath string) bool {
	// Skip binary files and common non-code files
	ext := strings.ToLower(filepath.Ext(filePath))
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

	// Check skip patterns
	for _, pattern := range e.config.SkipPatterns {
		if strings.Contains(filePath, pattern) {
			return false
		}
	}

	// Skip common non-code directories
	skipDirs := []string{"node_modules", "vendor", ".git", "build", "dist", "target"}
	for _, dir := range skipDirs {
		if strings.Contains(filePath, dir) {
			return false
		}
	}

	return true
}

// shouldAnalyzeDirectory determines if a directory should be analyzed
func (e *Engine) shouldAnalyzeDirectory(dirPath string) bool {
	// Skip common non-code directories
	skipDirs := []string{"node_modules", "vendor", ".git", "build", "dist", "target", "bin", "obj"}
	for _, dir := range skipDirs {
		if strings.Contains(dirPath, dir) {
			return false
		}
	}

	return true
}

// detectLanguage detects the programming language based on file extension and content
func (e *Engine) detectLanguage(filePath, content string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Check language mapping first
	if lang, ok := e.config.LanguageMapping[ext]; ok {
		return lang
	}

	// Default language detection based on extension
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
	case ".xml", ".yaml", ".yml", ".json":
		return "data"
	case ".md", ".txt":
		return "markdown"
	default:
		// Try to detect from content
		if strings.Contains(content, "package main") || strings.Contains(content, "import (") {
			return "go"
		}
		if strings.Contains(content, "def ") || strings.Contains(content, "import ") {
			return "python"
		}
		if strings.Contains(content, "function ") || strings.Contains(content, "var ") {
			return "javascript"
		}
		if strings.Contains(content, "public class") || strings.Contains(content, "import java.") {
			return "java"
		}
		return "unknown"
	}
}

// GetDefaultConfig returns default configuration for the analysis engine
func GetDefaultConfig() *Config {
	return &Config{
		MaxFileSize:   1024 * 1024, // 1MB
		AnalysisDepth: 3,
		EnableSecurity: true,
		EnableQuality:  true,
		SkipPatterns: []string{
			"node_modules", "vendor", ".git", "build", "dist", "target",
			"bin", "obj", "coverage", "logs", "tmp", "temp",
		},
		LanguageMapping: map[string]string{
			".go":   "go",
			".py":   "python",
			".js":   "javascript",
			".jsx":  "javascript",
			".ts":   "typescript",
			".tsx":  "typescript",
			".java": "java",
			".cpp":  "cpp",
			".c":    "c",
			".cs":   "csharp",
			".php":  "php",
			".rb":   "ruby",
			".rs":   "rust",
		},
	}
}
