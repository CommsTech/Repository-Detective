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
	giteaClient   *gitea.Client
	logger        *logrus.Logger
	config        *Config
	semanticStore *SemanticStore
}

// Config holds issue manager configuration
type Config struct {
	AutoCreateIssues   bool
	GiteaBaseURL       string
	IssueLabels        []string
	IssueTemplate      string
	CommentTemplate    string
	MaxIssuesPerRun    int
	SkipLowSeverity    bool
	GroupSimilarIssues bool
	MinIssueConfidence float64
	IssueTitleTemplate string
	IssueBodyTemplate  string
}

// IssueCreationRequest represents a request to create issues
type IssueCreationRequest struct {
	Owner              string
	Repository         string
	AnalysisResult     *ai.CodeAnalysisResult
	Context            string
	Commit             string
	PullRequest        int
	ScanID             string
	UseSemanticDedup     bool
	MinIssueConfidence   float64
	ForceIssueCreation   bool
}

// ProcessedIssueRecord links a finding fingerprint to a forge issue action.
type ProcessedIssueRecord struct {
	Fingerprint string
	IssueNumber int
	IssueURL    string
	Action      string // created, updated
}

// IssueCreationResult represents the result of issue creation
type IssueCreationResult struct {
	IssuesCreated   int
	IssuesSkipped   int
	IssuesUpdated   int
	Errors          []string
	IssueURLs       []string
	ProcessedIssues []ProcessedIssueRecord
}

// NewManager creates a new issue manager
func NewManager(giteaClient *gitea.Client, config *Config, logger *logrus.Logger, semanticStore *SemanticStore) *Manager {
	return &Manager{
		giteaClient:   giteaClient,
		logger:        logger,
		config:        config,
		semanticStore: semanticStore,
	}
}

// CreateIssuesFromAnalysis creates or updates Gitea issues based on analysis results
func (m *Manager) CreateIssuesFromAnalysis(ctx context.Context, req *IssueCreationRequest) (*IssueCreationResult, error) {
	startTime := time.Now()
	m.logger.Infof("Starting issue creation for %s/%s", req.Owner, req.Repository)

	result := &IssueCreationResult{
		IssuesCreated: 0,
		IssuesSkipped: 0,
		IssuesUpdated: 0,
		Errors:        []string{},
		IssueURLs:     []string{},
	}

	if !m.config.AutoCreateIssues && !req.ForceIssueCreation {
		m.logger.Info("Auto issue creation is disabled, skipping")
		return result, nil
	}

	repository := fmt.Sprintf("%s/%s", req.Owner, req.Repository)
	seenFingerprints := make(map[string]struct{})

	if req.AnalysisResult != nil {
		for i := range req.AnalysisResult.Issues {
			issue := &req.AnalysisResult.Issues[i]
			if m.config.MaxIssuesPerRun > 0 && result.IssuesCreated >= m.config.MaxIssuesPerRun {
				m.logger.Infof("Reached maximum issues limit (%d), skipping remaining issues", m.config.MaxIssuesPerRun)
				result.IssuesSkipped += len(req.AnalysisResult.Issues) - i
				break
			}

			minConfidence := m.config.MinIssueConfidence
			if req.MinIssueConfidence > 0 {
				minConfidence = req.MinIssueConfidence
			}
			if minConfidence <= 0 {
				minConfidence = 0.5
			}
			if issue.Confidence > 0 && issue.Confidence < minConfidence {
				m.logger.Debugf("Skipping low-confidence issue: %s (%.2f)", issue.Title, issue.Confidence)
				result.IssuesSkipped++
				continue
			}

			if m.config.SkipLowSeverity && strings.EqualFold(issue.Severity, "low") {
				m.logger.Debugf("Skipping low severity issue: %s", issue.Title)
				result.IssuesSkipped++
				continue
			}

			EnrichIssue(repository, issue, req.ScanID)
			seenFingerprints[issue.Fingerprint] = struct{}{}

			action, err := m.createOrUpdateIssue(ctx, req, repository, issue, result)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to process issue for %s: %v", issue.Title, err)
				result.Errors = append(result.Errors, errorMsg)
				m.logger.Errorf(errorMsg)
				continue
			}
			if action == "updated" {
				result.IssuesUpdated++
			}
		}
	}

	if req.ScanID != "" && len(seenFingerprints) > 0 {
		if err := ReportNotReproduced(ctx, m.giteaClient, req.Owner, req.Repository, req.ScanID, seenFingerprints); err != nil {
			m.logger.Warnf("Failed to report not-reproduced findings: %v", err)
		}
	}

	if m.config.GroupSimilarIssues && req.AnalysisResult != nil && shouldCreateSummaryIssue(req.AnalysisResult.Issues) {
		if err := m.createSummaryIssue(ctx, req, result); err != nil {
			errorMsg := fmt.Sprintf("Failed to create summary issue: %v", err)
			result.Errors = append(result.Errors, errorMsg)
			m.logger.Errorf(errorMsg)
		}
	}

	m.logger.Infof("Issue creation completed in %v, created %d, updated %d, skipped %d",
		time.Since(startTime), result.IssuesCreated, result.IssuesUpdated, result.IssuesSkipped)

	return result, nil
}

// shouldCreateSummaryIssue avoids one extra rollup ticket for tiny scan runs (reduces board noise).
func shouldCreateSummaryIssue(issues []ai.CodeIssue) bool {
	return len(issues) >= 5
}

func (m *Manager) createOrUpdateIssue(ctx context.Context, req *IssueCreationRequest, repository string, issue *ai.CodeIssue, result *IssueCreationResult) (string, error) {
	if match, err := FindIssueByFingerprint(ctx, m.giteaClient, req.Owner, req.Repository, issue.Fingerprint); err != nil {
		m.logger.Warnf("Fingerprint lookup failed: %v", err)
	} else if match != nil {
		if err := m.updateExistingIssue(ctx, req, issue, match, result); err != nil {
			return "", err
		}
		return "updated", nil
	}

	if req.UseSemanticDedup && m.semanticStore != nil && m.semanticStore.Enabled() {
		dup, err := m.semanticStore.FindDuplicate(ctx, repository, issue)
		if err != nil {
			m.logger.Warnf("Semantic dedup lookup failed: %v", err)
		} else if dup != nil && dup.IssueNumber > 0 {
			comment := DuplicateCommentBody(issue, dup.Score)
			if err := m.giteaClient.CreateIssueComment(ctx, req.Owner, req.Repository, dup.IssueNumber, comment); err != nil {
				m.logger.Warnf("Failed to comment on duplicate issue #%d: %v", dup.IssueNumber, err)
			} else {
				m.logger.Infof("Updated existing issue #%d via semantic dedup (score %.2f)", dup.IssueNumber, dup.Score)
				result.IssuesSkipped++
				result.IssueURLs = append(result.IssueURLs, dup.IssueURL)
				return "updated", nil
			}
		}
	}

	if err := m.createIssueForProblem(ctx, req, issue, result); err != nil {
		return "", err
	}
	return "created", nil
}

func (m *Manager) updateExistingIssue(ctx context.Context, req *IssueCreationRequest, issue *ai.CodeIssue, match *ExistingIssueMatch, result *IssueCreationResult) error {
	var comment string
	var labels []any

	if ConfidenceNeedsHumanReview(issue.Confidence) {
		comment = NeedsHumanReviewCommentBody(issue, req.ScanID)
		labels = ExpandLifecycleLabel(LifecycleNeedsHumanReview)
	} else {
		comment = StillPresentCommentBody(issue, req.ScanID)
		labels = ExpandLifecycleLabel(LifecycleStillPresent)
	}

	if err := m.giteaClient.CreateIssueComment(ctx, req.Owner, req.Repository, match.IssueNumber, comment); err != nil {
		return fmt.Errorf("comment on existing issue #%d: %w", match.IssueNumber, err)
	}

	if _, err := m.giteaClient.AddIssueLabels(ctx, req.Owner, req.Repository, match.IssueNumber, labels); err != nil {
		m.logger.Warnf("Failed to attach lifecycle labels to issue #%d: %v", match.IssueNumber, err)
	}

	result.IssuesSkipped++
	result.IssueURLs = append(result.IssueURLs, match.IssueURL)
	result.ProcessedIssues = append(result.ProcessedIssues, ProcessedIssueRecord{
		Fingerprint: issue.Fingerprint,
		IssueNumber: match.IssueNumber,
		IssueURL:    match.IssueURL,
		Action:      "updated",
	})
	m.logger.Infof("Updated existing issue #%d for fingerprint %s", match.IssueNumber, issue.Fingerprint)
	return nil
}

func (m *Manager) createIssueForProblem(ctx context.Context, req *IssueCreationRequest, issue *ai.CodeIssue, result *IssueCreationResult) error {
	repository := fmt.Sprintf("%s/%s", req.Owner, req.Repository)

	title := m.createIssueTitle(issue, req)
	body := m.createIssueBody(issue, req)

	labelNames := BuildLabels(m.config.IssueLabels, issue)
	labelIDs, err := m.giteaClient.ResolveLabelIDs(ctx, req.Owner, req.Repository, labelNames)
	if err != nil {
		m.logger.Warnf("Failed to resolve labels: %v", err)
	}

	issueReq := &gitea.CreateIssueRequest{
		Title:  title,
		Body:   body,
		Labels: labelIDs,
	}

	createdIssue, err := m.giteaClient.CreateIssue(ctx, req.Owner, req.Repository, issueReq)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Labels are set via CreateIssueRequest; only backfill when Gitea ignored label IDs.
	if len(labelIDs) == 0 && len(labelNames) > 0 {
		labelPayload := make([]any, 0, len(labelNames))
		for _, name := range labelNames {
			labelPayload = append(labelPayload, name)
		}
		if _, err := m.giteaClient.AddIssueLabels(ctx, req.Owner, req.Repository, createdIssue.Number, labelPayload); err != nil {
			m.logger.Warnf("Failed to attach labels to issue #%d: %v", createdIssue.Number, err)
		}
	}

	if m.semanticStore != nil && m.semanticStore.Enabled() {
		if err := m.semanticStore.Remember(ctx, repository, issue, createdIssue.HTMLURL, createdIssue.Number, issue.ClusterID); err != nil {
			m.logger.Warnf("Failed to store finding in Qdrant: %v", err)
		}
	}

	result.IssuesCreated++
	result.IssueURLs = append(result.IssueURLs, createdIssue.HTMLURL)
	result.ProcessedIssues = append(result.ProcessedIssues, ProcessedIssueRecord{
		Fingerprint: issue.Fingerprint,
		IssueNumber: createdIssue.Number,
		IssueURL:    createdIssue.HTMLURL,
		Action:      "created",
	})
	m.logger.Infof("Created issue #%d: %s", createdIssue.Number, title)
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	var out []string
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

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

	if len(labelIDs) > 0 {
		payload := make([]any, 0, len(labelIDs))
		for _, id := range labelIDs {
			payload = append(payload, id)
		}
		if _, err := m.giteaClient.AddIssueLabels(ctx, req.Owner, req.Repository, createdIssue.Number, payload); err != nil {
			m.logger.Warnf("Failed to attach summary labels: %v", err)
		}
	}

	result.IssuesCreated++
	result.IssueURLs = append(result.IssueURLs, createdIssue.HTMLURL)
	m.logger.Infof("Created summary issue #%d", createdIssue.Number)
	return nil
}

func (m *Manager) createIssueTitle(issue *ai.CodeIssue, req *IssueCreationRequest) string {
	if m.config.IssueTitleTemplate != "" {
		title := m.config.IssueTitleTemplate
		title = strings.ReplaceAll(title, "{{severity}}", strings.ToUpper(issue.Severity))
		title = strings.ReplaceAll(title, "{{category}}", issue.Category)
		title = strings.ReplaceAll(title, "{{title}}", issue.Title)
		title = strings.ReplaceAll(title, "{{file}}", issue.File)
		if issue.LineNumber > 0 {
			title = strings.ReplaceAll(title, "{{line}}", fmt.Sprintf("%d", issue.LineNumber))
		}
		return title
	}

	severity := strings.ToUpper(issue.Severity)
	title := issue.Title
	if loc := locationRef(issue); loc != "" && !strings.Contains(title, loc) {
		title = fmt.Sprintf("%s — %s", title, loc)
	}
	return fmt.Sprintf("[%s] %s", severity, title)
}

func (m *Manager) createIssueBody(issue *ai.CodeIssue, req *IssueCreationRequest) string {
	repository := fmt.Sprintf("%s/%s", req.Owner, req.Repository)
	if m.config.IssueBodyTemplate != "" {
		body := m.applyIssueBodyTemplate(m.config.IssueBodyTemplate, issue, req, repository)
		return body
	}

	return RenderIssueBody(IssueRenderInput{
		Issue:        issue,
		Repository:   repository,
		Owner:        req.Owner,
		RepoName:     req.Repository,
		GiteaBaseURL: m.config.GiteaBaseURL,
		Context:      req.Context,
		Commit:       req.Commit,
		Ref:          req.Commit,
		PullRequest:  req.PullRequest,
		ScanID:       req.ScanID,
	})
}

func (m *Manager) applyIssueBodyTemplate(tmpl string, issue *ai.CodeIssue, req *IssueCreationRequest, repository string) string {
	body := tmpl
	body = strings.ReplaceAll(body, "{{description}}", issue.Description)
	body = strings.ReplaceAll(body, "{{title}}", issue.Title)
	body = strings.ReplaceAll(body, "{{severity}}", issue.Severity)
	body = strings.ReplaceAll(body, "{{category}}", issue.Category)
	body = strings.ReplaceAll(body, "{{confidence}}", fmt.Sprintf("%.2f", issue.Confidence))
	body = strings.ReplaceAll(body, "{{context}}", req.Context)
	body = strings.ReplaceAll(body, "{{commit}}", req.Commit)
	body = strings.ReplaceAll(body, "{{repository}}", repository)
	body = strings.ReplaceAll(body, "{{fingerprint}}", issue.Fingerprint)
	body = strings.ReplaceAll(body, "{{scan_id}}", req.ScanID)
	body = strings.ReplaceAll(body, "{{file}}", issue.File)
	body = strings.ReplaceAll(body, "{{line}}", fmt.Sprintf("%d", issue.LineNumber))
	body = strings.ReplaceAll(body, "{{source}}", issue.Source)
	body = strings.ReplaceAll(body, "{{rule_id}}", issue.RuleID)
	body = strings.ReplaceAll(body, "{{evidence}}", SanitizeSecretEvidence(issue.CodeSnippet))
	body = strings.ReplaceAll(body, "{{recommended_fix}}", recommendedFix(issue))
	if req.PullRequest > 0 {
		body = strings.ReplaceAll(body, "{{pull_request}}", fmt.Sprintf("#%d", req.PullRequest))
	}
	return body
}

func (m *Manager) createSummaryIssueBody(req *IssueCreationRequest) string {
	var body strings.Builder

	body.WriteString("## Code Review Summary\n\n")
	body.WriteString(fmt.Sprintf("**Total Issues Found:** %d\n", len(req.AnalysisResult.Issues)))
	body.WriteString(fmt.Sprintf("**Overall Score:** %.2f%%\n", req.AnalysisResult.OverallScore*100))
	body.WriteString(fmt.Sprintf("**Analysis Time:** %v\n", req.AnalysisResult.AnalysisTime))
	if req.ScanID != "" {
		body.WriteString(fmt.Sprintf("**Scan ID:** %s\n", req.ScanID))
	}

	severityCounts := make(map[string]int)
	for _, issue := range req.AnalysisResult.Issues {
		severityCounts[issue.Severity]++
	}

	body.WriteString("\n## Issue Breakdown\n\n")
	for severity, count := range severityCounts {
		body.WriteString(fmt.Sprintf("- **%s:** %d issues\n", capitalizeWord(severity), count))
	}

	categoryCounts := make(map[string]int)
	for _, issue := range req.AnalysisResult.Issues {
		categoryCounts[NormalizeCategory(issue.Category, issue.Source)]++
	}

	body.WriteString("\n## Category Breakdown\n\n")
	for category, count := range categoryCounts {
		body.WriteString(fmt.Sprintf("- **%s:** %d issues\n", capitalizeWord(category), count))
	}

	body.WriteString("\n## Top Issues\n\n")

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

func capitalizeWord(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func GetDefaultConfig() *Config {
	return &Config{
		AutoCreateIssues:   true,
		IssueLabels:        []string{"repository-detective", "automated-review"},
		MaxIssuesPerRun:    50,
		SkipLowSeverity:    false,
		GroupSimilarIssues: true,
		MinIssueConfidence: 0.5,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "",
	}
}
