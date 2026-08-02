package issues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/github"
	"git.commsnet.org/commstech/repository-detective/profile"
	"github.com/sirupsen/logrus"
)

// IssueMappingLookup resolves local fingerprint → forge issue links.
type IssueMappingLookup interface {
	MappedIssueNumber(ctx context.Context, repositoryID int64, forgeType, fingerprint string) (issueNumber int, issueURL string, ok bool)
	LinkForgeIssue(ctx context.Context, repositoryID int64, forgeType, fingerprint, scanID string, issueNumber int, issueURL string)
}

// BackfillRunner repairs missing local mappings from open forge issues.
type BackfillRunner interface {
	BackfillMissingMappings(ctx context.Context, req *IssueCreationRequest) (BackfillOutcome, error)
}

// BackfillOutcome summarizes mapping repair.
type BackfillOutcome struct {
	Examined   int
	Backfilled int
	Skipped    int
}

// Manager handles issue creation and management on Gitea and GitHub.
type Manager struct {
	giteaForge     IssueForge
	githubForge    IssueForge
	logger         *logrus.Logger
	config         *Config
	mappingLookup  IssueMappingLookup
	backfillRunner BackfillRunner
}

// Config holds issue manager configuration
type Config struct {
	AutoCreateIssues   bool
	Reporting          profile.ReportingConfig
	BacklogControl     BacklogControlConfig
	GiteaBaseURL       string
	GitHubBaseURL      string
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
	ForgeType          string // gitea (default) or github
	Owner              string
	Repository         string
	RepositoryID       int64
	AnalysisResult     *ai.CodeAnalysisResult
	Context            string
	Commit             string
	PullRequest        int
	ScanID               string
	MinIssueConfidence   float64
	ForceIssueCreation   bool
}

// ProcessedIssueRecord links a finding fingerprint to a forge issue action.
type ProcessedIssueRecord struct {
	Fingerprint string
	ForgeType   string
	IssueNumber int
	IssueURL    string
	Action      string // created, updated
}

// IssueCreationResult represents the result of issue creation
type IssueCreationResult struct {
	IssuesCreated          int
	IssuesSkipped          int
	IssuesUpdated          int
	BacklogControlBlocked  int
	BacklogControlActive   bool
	BacklogControlNote     string
	Errors                 []string
	IssueURLs              []string
	ProcessedIssues        []ProcessedIssueRecord
}

// NewManager creates a new issue manager.
func NewManager(giteaClient *gitea.Client, githubClient *github.Client, config *Config, logger *logrus.Logger) *Manager {
	if config != nil && config.GitHubBaseURL == "" {
		config.GitHubBaseURL = "https://github.com"
	}
	m := &Manager{
		logger: logger,
		config: config,
	}
	if giteaClient != nil {
		m.giteaForge = &GiteaForge{Client: giteaClient}
	}
	if githubClient != nil {
		m.githubForge = &GitHubForge{Client: githubClient}
	}
	return m
}

// SetIssueMappingLookup enables local fingerprint → external issue lookup for idempotent filing.
func (m *Manager) SetIssueMappingLookup(lookup IssueMappingLookup) {
	m.mappingLookup = lookup
}

// SetBackfillRunner enables pre-filing mapping repair from open forge issues.
func (m *Manager) SetBackfillRunner(runner BackfillRunner) {
	m.backfillRunner = runner
}

// BackfillMissingMappings repairs local external_issues rows from open forge issues.
func (m *Manager) BackfillMissingMappings(ctx context.Context, req *IssueCreationRequest) (BackfillOutcome, error) {
	if m == nil || req == nil || m.backfillRunner == nil {
		return BackfillOutcome{}, nil
	}
	return m.backfillRunner.BackfillMissingMappings(ctx, req)
}

func (m *Manager) forgeFor(forgeType string) IssueForge {
	forgeType = strings.ToLower(strings.TrimSpace(forgeType))
	if forgeType == "github" && m.githubForge != nil {
		return m.githubForge
	}
	return m.giteaForge
}

// ForgeFor returns the issue forge adapter for a platform.
func (m *Manager) ForgeFor(forgeType string) IssueForge {
	return m.forgeFor(forgeType)
}

func (m *Manager) normalizeForgeType(forgeType string) string {
	forgeType = strings.ToLower(strings.TrimSpace(forgeType))
	if forgeType == "github" {
		return "github"
	}
	return "gitea"
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

			if issue.ReportingAction != "" && !profile.IsForgeAction(issue.ReportingAction, m.config.Reporting) {
				m.logger.Debugf("Skipping non-auto issue action=%s: %s", issue.ReportingAction, issue.Title)
				result.IssuesSkipped++
				continue
			}

			EnrichIssue(repository, issue, req.ScanID)
			seenFingerprints[issue.Fingerprint] = struct{}{}

			action, err := m.createOrUpdateIssue(ctx, req, issue, result)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to process issue for %s: %v", issue.Title, err)
				result.Errors = append(result.Errors, errorMsg)
				m.logger.Error(errorMsg)
				continue
			}
			if action == "updated" {
				result.IssuesUpdated++
			}
		}
	}

	if req.ScanID != "" && len(seenFingerprints) > 0 {
		forge := m.forgeFor(req.ForgeType)
		if err := ReportNotReproduced(ctx, forge, req.Owner, req.Repository, req.ScanID, seenFingerprints); err != nil {
			m.logger.Warnf("Failed to report not-reproduced findings: %v", err)
		}
	}

	if m.config.GroupSimilarIssues && req.AnalysisResult != nil && shouldCreateSummaryIssue(req.AnalysisResult.Issues) {
		if m.config.BacklogControl.ShouldBlockSummaryIssue() {
			m.logger.Infof("%s Skipping summary issue creation.", BacklogControlNote)
			result.BacklogControlActive = true
			result.BacklogControlNote = BacklogControlNote
			result.BacklogControlBlocked++
		} else if err := m.createSummaryIssue(ctx, req, result); err != nil {
			errorMsg := fmt.Sprintf("Failed to create summary issue: %v", err)
			result.Errors = append(result.Errors, errorMsg)
			m.logger.Error(errorMsg)
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

func (m *Manager) createOrUpdateIssue(ctx context.Context, req *IssueCreationRequest, issue *ai.CodeIssue, result *IssueCreationResult) (string, error) {
	forge := m.forgeFor(req.ForgeType)
	if forge == nil {
		return "", fmt.Errorf("no issue forge configured for %s", m.normalizeForgeType(req.ForgeType))
	}

	forgeType := m.normalizeForgeType(req.ForgeType)
	if m.mappingLookup != nil && req.RepositoryID > 0 && issue.Fingerprint != "" {
		if num, url, ok := m.mappingLookup.MappedIssueNumber(ctx, req.RepositoryID, forgeType, issue.Fingerprint); ok {
			match := &ExistingIssueMatch{IssueNumber: num, IssueURL: url}
			if err := m.updateExistingIssue(ctx, forge, req, issue, match, result); err != nil {
				return "", err
			}
			return "updated", nil
		}
	}

	if match, err := FindIssueByFingerprint(ctx, forge, req.Owner, req.Repository, issue.Fingerprint); err != nil {
		m.logger.Warnf("Fingerprint lookup failed: %v", err)
	} else if match != nil {
		if m.mappingLookup != nil && req.RepositoryID > 0 {
			m.mappingLookup.LinkForgeIssue(ctx, req.RepositoryID, forgeType, issue.Fingerprint, req.ScanID, match.IssueNumber, match.IssueURL)
		}
		if err := m.updateExistingIssue(ctx, forge, req, issue, match, result); err != nil {
			return "", err
		}
		return "updated", nil
	}

	if err := m.createIssueForProblem(ctx, forge, req, issue, result); err != nil {
		return "", err
	}
	return "created", nil
}

func (m *Manager) updateExistingIssue(ctx context.Context, forge IssueForge, req *IssueCreationRequest, issue *ai.CodeIssue, match *ExistingIssueMatch, result *IssueCreationResult) error {
	var comment string
	var labels []string

	if ConfidenceNeedsHumanReview(issue.Confidence) {
		comment = NeedsHumanReviewCommentBody(issue, req.ScanID)
		labels = ExpandLifecycleLabels(LifecycleNeedsHumanReview)
	} else {
		comment = StillPresentCommentBody(issue, req.ScanID)
		labels = ExpandLifecycleLabels(LifecycleStillPresent)
	}

	if err := forge.CreateIssueComment(ctx, req.Owner, req.Repository, match.IssueNumber, comment); err != nil {
		return fmt.Errorf("comment on existing issue #%d: %w", match.IssueNumber, err)
	}

	if err := forge.AddIssueLabels(ctx, req.Owner, req.Repository, match.IssueNumber, labels); err != nil {
		m.logger.Warnf("Failed to attach lifecycle labels to issue #%d: %v", match.IssueNumber, err)
	}

	result.IssuesSkipped++
	result.IssueURLs = append(result.IssueURLs, match.IssueURL)
	result.ProcessedIssues = append(result.ProcessedIssues, ProcessedIssueRecord{
		Fingerprint: issue.Fingerprint,
		ForgeType:   m.normalizeForgeType(req.ForgeType),
		IssueNumber: match.IssueNumber,
		IssueURL:    match.IssueURL,
		Action:      "updated",
	})
	m.logger.Infof("Updated existing issue #%d for fingerprint %s", match.IssueNumber, issue.Fingerprint)
	return nil
}

func (m *Manager) createIssueForProblem(ctx context.Context, forge IssueForge, req *IssueCreationRequest, issue *ai.CodeIssue, result *IssueCreationResult) error {
	if m.config.BacklogControl.Enabled {
		result.BacklogControlActive = true
		result.BacklogControlNote = BacklogControlNote
		if blocked, reason := m.config.BacklogControl.ShouldBlockNewIssue(issue, 0); blocked {
			m.logger.Infof("%s Skipping new issue for %q: %s", BacklogControlNote, issue.Title, reason)
			result.BacklogControlBlocked++
			result.IssuesSkipped++
			return nil
		}
	}

	title := m.createIssueTitle(issue, req)
	body := m.createIssueBody(issue, req)
	labelNames := BuildLabels(m.config.IssueLabels, issue)

	createdIssue, err := forge.CreateIssue(ctx, req.Owner, req.Repository, title, body, labelNames)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	result.IssuesCreated++
	result.IssueURLs = append(result.IssueURLs, createdIssue.HTMLURL)
	result.ProcessedIssues = append(result.ProcessedIssues, ProcessedIssueRecord{
		Fingerprint: issue.Fingerprint,
		ForgeType:   m.normalizeForgeType(req.ForgeType),
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
	forge := m.forgeFor(req.ForgeType)
	if forge == nil {
		return fmt.Errorf("no issue forge configured for %s", m.normalizeForgeType(req.ForgeType))
	}
	title := fmt.Sprintf("Code Review Summary - %d Issues Found", len(req.AnalysisResult.Issues))
	body := m.createSummaryIssueBody(req)

	createdIssue, err := forge.CreateIssue(ctx, req.Owner, req.Repository, title, body, m.config.IssueLabels)
	if err != nil {
		return fmt.Errorf("failed to create summary issue: %w", err)
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

	webBase := m.config.GiteaBaseURL
	if m.normalizeForgeType(req.ForgeType) == "github" {
		webBase = m.config.GitHubBaseURL
	}
	return RenderIssueBody(IssueRenderInput{
		Issue:        issue,
		Repository:   repository,
		Owner:        req.Owner,
		RepoName:     req.Repository,
		GiteaBaseURL: webBase,
		Context:      req.Context,
		Commit:       req.Commit,
		Ref:          req.Commit,
		PullRequest:  req.PullRequest,
		ScanID:       req.ScanID,
		Provider:     m.normalizeForgeType(req.ForgeType),
		ReportOnly:   !req.ForceIssueCreation,
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

func formatAnalysisOverallScore(result *ai.CodeAnalysisResult) string {
	if result == nil {
		return "incomplete"
	}
	complete := result.ScoreComplete
	score := result.OverallScore
	if !complete && score >= 0 && score <= 1 {
		complete = true
	}
	if !complete || score < 0 {
		if strings.TrimSpace(result.ScoreIncompleteReason) != "" {
			return "incomplete (" + result.ScoreIncompleteReason + ")"
		}
		return "incomplete"
	}
	line := fmt.Sprintf("%.2f%%", score*100)
	if strings.TrimSpace(result.ScoreExplanation) != "" {
		line += " — " + result.ScoreExplanation
	}
	return line
}

func (m *Manager) createSummaryIssueBody(req *IssueCreationRequest) string {
	var body strings.Builder

	body.WriteString("## Code Review Summary\n\n")
	body.WriteString(fmt.Sprintf("**Total Issues Found:** %d\n", len(req.AnalysisResult.Issues)))
	body.WriteString(fmt.Sprintf("**Overall Score:** %s\n", formatAnalysisOverallScore(req.AnalysisResult)))
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
	body.WriteString("*This summary was automatically generated by Repository Detective*\n")

	return body.String()
}

func capitalizeWord(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

// AnnotateCalibration adds lifecycle labels and a comment to an existing forge issue.
// Existing issues are not closed or deleted.
func (m *Manager) AnnotateCalibration(ctx context.Context, forgeType, owner, repo string, issueNumber int, falsePositive bool, reason string) error {
	if m == nil || issueNumber <= 0 {
		return nil
	}
	forge := m.forgeFor(forgeType)
	if forge == nil {
		return nil
	}
	var labels []string
	var marker string
	if falsePositive {
		labels = ExpandLifecycleLabels(LifecycleFalsePositive)
		marker = "false positive"
	} else {
		labels = ExpandLifecycleLabels(LifecycleSuppressed)
		marker = "suppressed"
	}
	if err := forge.AddIssueLabels(ctx, owner, repo, issueNumber, labels); err != nil {
		return err
	}
	body := "Repository Detective marked this finding as **" + marker + "**."
	if strings.TrimSpace(reason) != "" {
		body += "\n\n**Reason:** " + strings.TrimSpace(reason)
	}
	body += "\n\nThe issue remains open for audit history."
	return forge.CreateIssueComment(ctx, owner, repo, issueNumber, body)
}

func GetDefaultConfig() *Config {
	return &Config{
		AutoCreateIssues:   true,
		Reporting:          profile.DefaultReportingConfig(),
		BacklogControl:     DefaultBacklogControlConfig(),
		IssueLabels:        DefaultIssueBaseLabels(),
		MaxIssuesPerRun:    50,
		SkipLowSeverity:    false,
		GroupSimilarIssues: true,
		MinIssueConfidence: 0.5,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "",
	}
}
