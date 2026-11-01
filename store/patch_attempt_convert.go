package store

import (
	"encoding/json"

	"git.commsnet.org/commstech/repository-detective/patcher"
)

// PatchAttemptFromDomain converts a domain attempt to a store record.
func PatchAttemptFromDomain(a patcher.PatchAttempt) PatchAttemptRecord {
	var findingID *int64
	if a.FindingID > 0 {
		v := a.FindingID
		findingID = &v
	}
	files, _ := json.Marshal(a.FilesChanged)
	tests, _ := json.Marshal(a.TestsRun)
	return PatchAttemptRecord{
		AttemptID:         a.ID,
		PlanID:            a.PlanID,
		RepositoryID:      a.RepositoryID,
		FindingID:         findingID,
		BranchName:        a.BranchName,
		BaseRef:           a.BaseRef,
		BaseCommitSHA:     a.CommitSHA,
		Status:            a.Status,
		DiffSummary:       a.DiffSummary,
		FilesChangedJSON:  files,
		TestsRunJSON:      tests,
		ValidationSummary: a.ValidationSummary,
		PullRequestNumber: a.PullRequestNumber,
		PullRequestURL:    a.PullRequestURL,
		Error:             a.Error,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

// PatchAttemptToDomain converts a store record to a domain attempt.
func PatchAttemptToDomain(rec PatchAttemptRecord) patcher.PatchAttempt {
	var files []string
	var tests []patcher.TestResult
	_ = json.Unmarshal(rec.FilesChangedJSON, &files)
	_ = json.Unmarshal(rec.TestsRunJSON, &tests)
	attempt := patcher.PatchAttempt{
		ID:                rec.AttemptID,
		PlanID:            rec.PlanID,
		RepositoryID:      rec.RepositoryID,
		BranchName:        rec.BranchName,
		BaseRef:           rec.BaseRef,
		CommitSHA:         rec.BaseCommitSHA,
		Status:            rec.Status,
		DiffSummary:       rec.DiffSummary,
		FilesChanged:      files,
		TestsRun:          tests,
		ValidationSummary: rec.ValidationSummary,
		PullRequestNumber: rec.PullRequestNumber,
		PullRequestURL:    rec.PullRequestURL,
		Error:             rec.Error,
		CreatedAt:         rec.CreatedAt,
		UpdatedAt:         rec.UpdatedAt,
	}
	if rec.FindingID != nil {
		attempt.FindingID = *rec.FindingID
	}
	return attempt
}
