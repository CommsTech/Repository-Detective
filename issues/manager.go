package issues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/gitea"
	"github.com/sirupsen/logrus"
)

// Manager handles issue creation and management
type Manager struct {
	giteaClient *gitea.Client
	logger      *logrus.Logger
	config      *Config
}

// Config holds issue manager configuration
type Config struct {
	AutoCreateIssues   bool
	IssueLabels        []string
	IssueTemplate      string
	CommentTemplate    string
	MaxIssuesPerRun    int
	SkipLowSeverity    bool
	GroupSimilarIssues bool
	IssueTitleTemplate string
	IssueBodyTemplate  string
}

// IssueCreationRequest represents a request to create issues
type IssueCreationRequest struct {
	Owner          string
	Repository     string
	AnalysisResult *ai.CodeAnalysisResult
	Context        string
	Commit         string
	PullRequest    int
}

// IssueCreationResult represents the result of issue creation
type IssueCreationResult struct {
	IssuesCreated int
	IssuesSkipped int
	Errors        []string
	IssueURLs     []string
}

// NewManager creates a new issue manager
func NewManager(giteaClient *gitea.Client, config *Config, logger *logrus.Logger) *Manager {
	return &Manager{
		giteaClient: giteaClient,
		logger:      logger,
		config:      config,
	}
}

// CreateIssuesFromAnalysis creates Gitea issues based on analysis results
func (m *Manager) CreateIssuesFromAnalysis(ctx context.Context, req *IssueCreationRequest) (*IssueCreationResult, error) {
	startTime := time.Now()
	m.logger.Infof("Starting issue creation for %s/%s", req.Owner, req.Repository)

	result := &IssueCreationResult{
		IssuesCreated: 0,
		IssuesSkipped: 0,
		Errors:        []string{},
		IssueURLs:     []string{},
	}

	// Check if we should create issues
	if !m.config.AutoCreateIssues {
		m.logger.Info("Auto issue creation is disabled, skipping")
		return result, nil
	}

	// Process each issue from the analysis
	for i, issue := range req.AnalysisResult.Issues {
		// Check if we've reached the maximum issues limit
		if m.config.MaxIssuesPerRun > 0 && result.IssuesCreated >= m.config.MaxIssuesPerRun {
			m.logger.Infof("Reached maximum issues limit (%d), skipping remaining issues", m.config.MaxIssuesPerRun)
			result.IssuesSkipped += len(req.AnalysisResult.Issues) - i
			break
		}

		// Skip low severity issues if configured
		if m.config.SkipLowSeverity && issue.Severity == "low" {
			m.logger.Debugf("Skipping low severity issue: %s", issue.Title)
			result.IssuesSkipped++
			continue
		}

		// Create issue for this problem
		if err := m.createIssueForProblem(ctx, req, &issue, result); err != nil {
			errorMsg := fmt.Sprintf("Failed to create issue for %s: %v", issue.Title, err)
			result.Errors = append(result.Errors, errorMsg)
			m.logger.Errorf(errorMsg)
		}
	}

	// Create summary issue if multiple issues were found
	if m.config.GroupSimilarIssues && len(req.AnalysisResult.Issues) > 1 {
		if err := m.createSummaryIssue(ctx, req, result); err != nil {
			errorMsg := fmt.Sprintf("Failed to create summary issue: %v", err)
			result.Errors = append(result.Errors, errorMsg)
			m.logger.Errorf(errorMsg)
		}
	}

	m.logger.Infof("Issue creation completed in %v, created %d issues, skipped %d",
		time.Since(startTime), result.IssuesCreated, result.IssuesSkipped)

	return result, nil
}

// createIssueForProblem creates a Gitea issue for a specific code problem
func (m *Manager) createIssueForProblem(ctx context.Context, req *IssueCreationRequest, issue *ai.CodeIssue, result *IssueCreationResult) error {
	title := m.createIssueTitle(issue, req)
	body := m.createIssueBody(issue, req)

	labelIDs, err := m.giteaClient.ResolveLabelIDs(ctx, req.Owner, req.Repository, m.config.IssueLabels)
	if err != nil {
		m.logger.Warnf("Failed to resolve labels: %v", err)
	}

	issueReq := &gitea.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labelIDs,
	}

	// Create the issue
	createdIssue, err := m.giteaClient.CreateIssue(ctx, req.Owner, req.Repository, issueReq)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	result.IssuesCreated++
	result.IssueURLs = append(result.IssueURLs, createdIssue.HTMLURL)

	m.logger.Infof("Created issue #%d: %s", createdIssue.Number, title)

	return nil
}

// createSummaryIssue creates a summary issue when multiple issues are found
func (m *Manager) createSummaryIssue(ctx context.Context, req *IssueCreationRequest, result *IssueCreationResult) error {
	title := fmt.Sprintf("Code Review Summary - %d Issues Found", len(req.AnalysisResult.Issues))

	body := m.createSummaryIssueBody(req)

	labelIDs, err := m.giteaClient.ResolveLabelIDs(ctx, req.Owner, req.Repository, m.config.IssueLabels)
	if err != nil {
		m.logger.Warnf("Failed to resolve labels for summary issue: %v", err)
	}

	issueReq := &gitea.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labelIDs,
	}

	createdIssue, err := m.giteaClient.CreateIssue(ctx, req.Owner, req.Repository, issueReq)
	if err != nil {
		return fmt.Errorf("failed to create summary issue: %w", err)
	}

	result.IssuesCreated++
	result.IssueURLs = append(result.IssueURLs, createdIssue.HTMLURL)

	m.logger.Infof("Created summary issue #%d", createdIssue.Number)

	return nil
}

// createIssueTitle creates a title for an issue
func (m *Manager) createIssueTitle(issue *ai.CodeIssue, req *IssueCreationRequest) string {
	if m.config.IssueTitleTemplate != "" {
		// Use custom template
		title := m.config.IssueTitleTemplate
		title = strings.ReplaceAll(title, "{{severity}}", issue.Severity)
		title = strings.ReplaceAll(title, "{{category}}", issue.Category)
		title = strings.ReplaceAll(title, "{{title}}", issue.Title)
		title = strings.ReplaceAll(title, "{{file}}", issue.CodeSnippet)
		return title
	}

	// Default title format
	severity := strings.ToUpper(issue.Severity)
	return fmt.Sprintf("[%s] %s", severity, issue.Title)
}

// createIssueBody creates the body content for an issue
func (m *Manager) createIssueBody(issue *ai.CodeIssue, req *IssueCreationRequest) string {
	if m.config.IssueBodyTemplate != "" {
		// Use custom template
		body := m.config.IssueBodyTemplate
		body = strings.ReplaceAll(body, "{{description}}", issue.Description)
		body = strings.ReplaceAll(body, "{{severity}}", issue.Severity)
		body = strings.ReplaceAll(body, "{{category}}", issue.Category)
		body = strings.ReplaceAll(body, "{{confidence}}", fmt.Sprintf("%.2f", issue.Confidence))
		body = strings.ReplaceAll(body, "{{context}}", req.Context)
		body = strings.ReplaceAll(body, "{{commit}}", req.Commit)
		if req.PullRequest > 0 {
			body = strings.ReplaceAll(body, "{{pull_request}}", fmt.Sprintf("#%d", req.PullRequest))
		}
		return body
	}

	// Default body format
	var body strings.Builder

	body.WriteString(fmt.Sprintf("## Issue Details\n\n"))
	body.WriteString(fmt.Sprintf("**Severity:** %s\n", issue.Severity))
	body.WriteString(fmt.Sprintf("**Category:** %s\n", issue.Category))
	body.WriteString(fmt.Sprintf("**Confidence:** %.2f%%\n", issue.Confidence*100))

	if issue.File != "" {
		body.WriteString(fmt.Sprintf("**File:** `%s`\n", issue.File))
	}

	if issue.LineNumber > 0 {
		body.WriteString(fmt.Sprintf("**Line:** %d\n", issue.LineNumber))
	}

	body.WriteString(fmt.Sprintf("\n## Description\n\n%s\n\n", issue.Description))

	if issue.CodeSnippet != "" {
		body.WriteString("## Code Snippet\n\n")
		body.WriteString(fmt.Sprintf("```\n%s\n```\n\n", issue.CodeSnippet))
	}

	if issue.ProofOfConcept != "" {
		body.WriteString("## Proof of Concept\n\n")
		body.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", issue.ProofOfConcept))
	}

	body.WriteString("## Context\n\n")
	body.WriteString(fmt.Sprintf("- **Repository:** %s\n", req.Repository))
	body.WriteString(fmt.Sprintf("- **Context:** %s\n", req.Context))

	if req.Commit != "" {
		body.WriteString(fmt.Sprintf("- **Commit:** %s\n", req.Commit))
	}

	if req.PullRequest > 0 {
		body.WriteString(fmt.Sprintf("- **Pull Request:** #%d\n", req.PullRequest))
	}

	body.WriteString(fmt.Sprintf("- **Detected:** %s\n", time.Now().Format(time.RFC3339)))

	body.WriteString("\n---\n")
	body.WriteString("*This issue was automatically generated by the Gitea Bugbot*\n")

	return body.String()
}

// createSummaryIssueBody creates the body for a summary issue
func (m *Manager) createSummaryIssueBody(req *IssueCreationRequest) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("## Code Review Summary\n\n"))
	body.WriteString(fmt.Sprintf("**Total Issues Found:** %d\n", len(req.AnalysisResult.Issues)))
	body.WriteString(fmt.Sprintf("**Overall Score:** %.2f%%\n", req.AnalysisResult.OverallScore*100))
	body.WriteString(fmt.Sprintf("**Analysis Time:** %v\n", req.AnalysisResult.AnalysisTime))

	// Group issues by severity
	severityCounts := make(map[string]int)
	for _, issue := range req.AnalysisResult.Issues {
		severityCounts[issue.Severity]++
	}

	body.WriteString("\n## Issue Breakdown\n\n")
	for severity, count := range severityCounts {
		body.WriteString(fmt.Sprintf("- **%s:** %d issues\n", strings.Title(severity), count))
	}

	// Group issues by category
	categoryCounts := make(map[string]int)
	for _, issue := range req.AnalysisResult.Issues {
		categoryCounts[issue.Category]++
	}

	body.WriteString("\n## Category Breakdown\n\n")
	for category, count := range categoryCounts {
		body.WriteString(fmt.Sprintf("- **%s:** %d issues\n", strings.Title(category), count))
	}

	body.WriteString("\n## Top Issues\n\n")

	// Show top 5 most critical issues
	topIssues := 5
	if len(req.AnalysisResult.Issues) < topIssues {
		topIssues = len(req.AnalysisResult.Issues)
	}

	for i := 0; i < topIssues; i++ {
		issue := req.AnalysisResult.Issues[i]
		body.WriteString(fmt.Sprintf("### %d. %s\n", i+1, issue.Title))
		body.WriteString(fmt.Sprintf("- **Severity:** %s\n", issue.Severity))
		body.WriteString(fmt.Sprintf("- **Category:** %s\n", issue.Category))
		body.WriteString(fmt.Sprintf("- **Description:** %s\n\n", issue.Description))
	}

	body.WriteString("## Context\n\n")
	body.WriteString(fmt.Sprintf("- **Repository:** %s\n", req.Repository))
	body.WriteString(fmt.Sprintf("- **Context:** %s\n", req.Context))

	if req.Commit != "" {
		body.WriteString(fmt.Sprintf("- **Commit:** %s\n", req.Commit))
	}

	if req.PullRequest > 0 {
		body.WriteString(fmt.Sprintf("- **Pull Request:** #%d\n", req.PullRequest))
	}

	body.WriteString(fmt.Sprintf("- **Analysis Completed:** %s\n", time.Now().Format(time.RFC3339)))

	body.WriteString("\n---\n")
	body.WriteString("*This summary was automatically generated by the Gitea Bugbot*\n")

	return body.String()
}

// GetDefaultConfig returns default configuration for the issue manager
func GetDefaultConfig() *Config {
	return &Config{
		AutoCreateIssues:   true,
		IssueLabels:        []string{"bugbot", "automated-review"},
		MaxIssuesPerRun:    50,
		SkipLowSeverity:    false,
		GroupSimilarIssues: true,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "## Issue Details\n\n**Severity:** {{severity}}\n**Category:** {{category}}\n**Confidence:** {{confidence}}\n\n## Description\n\n{{description}}\n\n## Context\n\n- **Repository:** {{repository}}\n- **Context:** {{context}}\n- **Commit:** {{commit}}\n\n---\n*This issue was automatically generated by the Gitea Bugbot*",
	}
}
