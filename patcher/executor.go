package patcher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/remediation"
)

// GiteaAPI abstracts Gitea operations used by the executor.
type GiteaAPI interface {
	CreatePullRequest(ctx context.Context, owner, repo string, req *gitea.CreatePullRequestRequest) (*gitea.PullRequest, error)
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error
}

// Executor runs safe remediation PR attempts.
type Executor struct {
	Config     Config
	Gitea      GiteaAPI
	GiteaToken string
	CloneURL   string
	IssueNumber int
}

// NewAttemptID generates a unique patch attempt identifier.
func NewAttemptID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return AttemptIDPrefix + fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return AttemptIDPrefix + hex.EncodeToString(buf)
}

// Run executes a patch attempt synchronously.
func (e *Executor) Run(ctx context.Context, input AttemptInput) (PatchAttempt, error) {
	now := time.Now().UTC()
	attempt := PatchAttempt{
		ID:           NewAttemptID(),
		PlanID:       input.Plan.ID,
		RepositoryID: input.Plan.RepositoryID,
		FindingID:    input.Plan.FindingID,
		Status:       StatusRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	eligibility := CheckPREligibility(input.Plan, input.Repo, e.Config)
	if !eligibility.Eligible {
		attempt.Status = StatusFailed
		attempt.Error = "not eligible: " + strings.Join(eligibility.BlockedReasons, "; ")
		attempt.UpdatedAt = time.Now().UTC()
		return attempt, fmt.Errorf("%s", attempt.Error)
	}

	baseRef := input.Repo.DefaultBranch
	if baseRef == "" {
		baseRef = "main"
	}
	attempt.BaseRef = baseRef

	branch := BranchName(e.Config.BranchPrefix, input.Plan.Fingerprint)
	if err := EnsureDefaultBranchNotTarget(branch, baseRef); err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}
	attempt.BranchName = branch

	cloneURL := e.CloneURL
	if cloneURL == "" {
		cloneURL = input.Repo.CloneURL
	}
	if cloneURL == "" {
		attempt.Status = StatusFailed
		attempt.Error = "clone URL unavailable"
		return attempt, fmt.Errorf("clone URL unavailable")
	}

	ws, err := PrepareWorkspace(ctx, cloneURL, e.GiteaToken, baseRef)
	if err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}
	defer ws.Cleanup()
	attempt.CommitSHA = ws.BaseSHA

	if err := CreateBranch(ctx, ws.Dir, branch); err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}

	maxFiles := e.Config.MaxFilesChanged
	if maxFiles <= 0 {
		maxFiles = 3
	}
	maxLines := e.Config.MaxDiffLines
	if maxLines <= 0 {
		maxLines = 100
	}

	patchResult, err := ApplyPatch(input.Plan, ws.Dir, maxFiles, maxLines)
	if err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}
	attempt.FilesChanged = patchResult.FilesChanged
	attempt.DiffSummary = patchResult.Summary

	timeout := time.Duration(e.Config.ValidationTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	var tests []TestResult
	allPassed := true
	for _, cmd := range input.Plan.ValidationCommands {
		if _, perr := ParseAllowedCommand(cmd); perr != nil {
			continue
		}
		result := RunAllowedCommand(cmd, ws.Dir, timeout)
		tests = append(tests, result)
		if result.Status != "passed" {
			allPassed = false
		}
	}
	attempt.TestsRun = tests
	attempt.ValidationSummary = SummarizeValidationResults(tests)
	if !allPassed || len(tests) == 0 {
		attempt.Status = StatusFailed
		if len(tests) == 0 {
			attempt.Error = "no allowlisted validation command passed"
		} else {
			attempt.Error = "validation failed"
		}
		return attempt, fmt.Errorf("%s", attempt.Error)
	}

	commitMsg := fmt.Sprintf("fix(repository-detective): %s (%s)", input.Plan.RuleID, input.Plan.Source)
	commitSHA, err := CommitAll(ctx, ws.Dir, commitMsg)
	if err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}
	attempt.CommitSHA = commitSHA

	if err := PushBranch(ctx, cloneURL, e.GiteaToken, ws.Dir, branch); err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}

	if e.Gitea == nil {
		attempt.Status = StatusFailed
		attempt.Error = "gitea client unavailable"
		return attempt, fmt.Errorf("gitea client unavailable")
	}

	owner, name, err := splitRepoFullName(input.Repo.FullName)
	if err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}

	prTitle := "Repository Detective: " + input.Plan.Title
	if prTitle == "Repository Detective: " {
		prTitle = "Repository Detective remediation"
	}
	prBody := RenderPRBody(input.Plan.Summary, patchResult.Summary, attempt.ValidationSummary)
	pr, err := e.Gitea.CreatePullRequest(ctx, owner, name, &gitea.CreatePullRequestRequest{
		Title: prTitle,
		Body:  prBody,
		Head:  branch,
		Base:  baseRef,
	})
	if err != nil {
		attempt.Status = StatusFailed
		attempt.Error = err.Error()
		return attempt, err
	}

	num := pr.Number
	attempt.PullRequestNumber = &num
	attempt.PullRequestURL = pr.HTMLURL
	attempt.Status = StatusPROpened
	attempt.UpdatedAt = time.Now().UTC()

	if e.IssueNumber > 0 {
		comment := RenderIssuePRComment(branch, pr.HTMLURL, attempt.ValidationSummary)
		if err := e.Gitea.CreateIssueComment(ctx, owner, name, e.IssueNumber, comment); err != nil {
			attempt.ValidationSummary = strings.TrimSpace(attempt.ValidationSummary + "; issue comment failed: " + err.Error())
		}
	}

	return attempt, nil
}

func splitRepoFullName(full string) (owner, name string, err error) {
	parts := strings.SplitN(full, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository full name")
	}
	return parts[0], parts[1], nil
}

// EligiblePlan is a convenience helper for tests.
func EligiblePlan() remediation.Plan {
	return remediation.Plan{
		ID:                  "rp-test",
		Fingerprint:         "fp-staticcheck-1",
		Status:              remediation.StatusApproved,
		Category:            "code_quality",
		Source:              "staticcheck",
		RuleID:              "S1039",
		Title:               "unnecessary fmt.Sprintf",
		Summary:             "replace fmt.Sprintf with string literal",
		AffectedFiles:       []string{"main.go"},
		ValidationCommands:  []string{"go vet ./..."},
		RegressionRisk:      remediation.RiskLow,
		FixComplexity:       remediation.ComplexitySmall,
		SafeForAutoPR:       true,
		RequiresHumanReview: false,
		Severity:            "medium",
	}
}
