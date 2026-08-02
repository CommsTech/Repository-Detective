package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"github.com/sirupsen/logrus"
)

// scanBatchStore extends Store with batched scan persistence operations.
type scanBatchStore interface {
	Store
	PersistScanFindingsBatch(ctx context.Context, repositoryID int64, scanID string, codeIssues []ai.CodeIssue, now time.Time) (int, map[string]int64, error)
	RecordExternalIssuesBatch(ctx context.Context, scanID string, forgeType string, processed []issues.ProcessedIssueRecord, findingIDs map[string]int64, now time.Time) error
	CountFindingInstancesForScan(ctx context.Context, scanID string) (int, error)
	UpdateScanPipelineState(ctx context.Context, scanID string, status string, fields map[string]any) error
}

// ScanContext describes a scan for persistence.
type ScanContext struct {
	Owner         string
	Repo          string
	ForgeType     string
	CloneURL      string
	DefaultBranch string
	TriggerType   string
	Ref           string
	CommitSHA     string
	PRNumber      int
	ScanID        string
	ConnectedRepo bool
}

// Recorder persists scan metadata to the local store.
type Recorder struct {
	store  Store
	logger *logrus.Logger
}

// NewRecorder creates a scan recorder. store may be nil (no-op).
func NewRecorder(s Store, logger *logrus.Logger) *Recorder {
	return &Recorder{store: s, logger: logger}
}

// Enabled reports whether persistence is active.
func (r *Recorder) Enabled() bool {
	return r != nil && r.store != nil
}

func (r *Recorder) batchStore() scanBatchStore {
	bs, _ := r.store.(scanBatchStore)
	return bs
}

// BeginScan upserts the repository and creates a started scan row.
func (r *Recorder) BeginScan(ctx context.Context, scanCtx ScanContext) (Repository, error) {
	if !r.Enabled() {
		return Repository{}, nil
	}
	if scanCtx.ForgeType == "" {
		scanCtx.ForgeType = ForgeTypeGitea
	}
	fullName := scanCtx.Owner + "/" + scanCtx.Repo

	repo, err := r.store.UpsertRepository(ctx, Repository{
		ForgeType:     scanCtx.ForgeType,
		Owner:         scanCtx.Owner,
		Name:          scanCtx.Repo,
		FullName:      fullName,
		CloneURL:      scanCtx.CloneURL,
		DefaultBranch: scanCtx.DefaultBranch,
		ConnectedRepo: scanCtx.ConnectedRepo,
	})
	if err != nil {
		return Repository{}, err
	}

	_, err = r.store.CreateScan(ctx, Scan{
		ID:           scanCtx.ScanID,
		RepositoryID: repo.ID,
		TriggerType:  scanCtx.TriggerType,
		Ref:          scanCtx.Ref,
		CommitSHA:    scanCtx.CommitSHA,
		PRNumber:     scanCtx.PRNumber,
		Status:       ScanStatusStarted,
		StartedAt:    time.Now().UTC(),
	})
	if err != nil {
		return Repository{}, err
	}

	return repo, nil
}

// FinishScan records scanner completion and marks analysis complete; findings persist separately.
func (r *Recorder) FinishScan(ctx context.Context, scanID string, data *ScanCompletion, analysisErr error) error {
	if !r.Enabled() || scanID == "" {
		return nil
	}

	status := ScanStatusAnalysisComplete
	errMsg := ""
	if analysisErr != nil {
		status = ScanStatusFailed
		errMsg = analysisErr.Error()
	}

	summary := map[string]any{
		"issues_found":        0,
		"files_analyzed":      0,
		"analysis_time_ms":    0,
		"persistence_status":  PersistenceStatusPending,
		"issue_sync_status":   IssueSyncStatusPending,
	}
	workspaceMode := ""
	commitPinned := false
	commitSHA := ""

	if data != nil && data.PolicySnapshot != nil {
		summary["effective_settings"] = data.PolicySnapshot
	}
	if data != nil && data.WorkspaceModeUsed != "" {
		workspaceMode = data.WorkspaceModeUsed
	}

	expectedCount := 0
	if data != nil {
		expectedCount = data.IssuesFound
		summary["issues_found"] = data.IssuesFound
		summary["files_analyzed"] = data.FilesAnalyzed
		summary["analysis_time_ms"] = data.AnalysisTime.Milliseconds()
		summary["overall_score"] = data.OverallScore
		summary["score_complete"] = data.ScoreComplete
		if data.ScoreIncompleteReason != "" {
			summary["score_incomplete_reason"] = data.ScoreIncompleteReason
		}
		if data.ScoreExplanation != "" {
			summary["score_explanation"] = data.ScoreExplanation
		}
		commitSHA = data.CommitSHA
		if data.RepoProfile != nil {
			summary["repo_profile"] = data.RepoProfile
		}
		summary["graph_enabled"] = data.GraphEnabled
		summary["graph_state"] = data.GraphState
		if data.GraphNodeCount > 0 {
			summary["graph_nodes"] = data.GraphNodeCount
			summary["graph_edges"] = data.GraphEdgeCount
		}
		if data.GraphTruncated {
			summary["graph_truncated"] = true
		}
		if data.GraphError != "" {
			summary["graph_error"] = data.GraphError
		}
		summary["persistence_expected_count"] = expectedCount
	}

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal scan summary: %w", err)
	}

	if err := r.store.FinishScan(ctx, scanID, ScanResult{
		Status:            status,
		FinishedAt:        time.Now().UTC(),
		SummaryJSON:       summaryJSON,
		Error:             errMsg,
		WorkspaceModeUsed: workspaceMode,
		CommitPinned:      commitPinned,
		CommitSHA:         commitSHA,
	}); err != nil {
		return err
	}

	if data != nil && len(data.GraphJSON) > 0 {
		scan, err := r.store.GetScan(ctx, scanID)
		if err == nil {
			if err := r.store.SaveScanGraph(ctx, ScanGraphRecord{
				ScanID: scanID, RepositoryID: scan.RepositoryID,
				GraphJSON: data.GraphJSON, NodeCount: data.GraphNodeCount,
				EdgeCount: data.GraphEdgeCount, GeneratedAt: time.Now().UTC(),
			}); err != nil && r.logger != nil {
				r.logger.Warnf("Failed to save scan graph: %v", err)
				if data.GraphError == "" {
					data.GraphError = "failed to persist graph: " + err.Error()
				}
			}
		}
	}

	if data == nil || analysisErr != nil {
		return nil
	}

	scannerRecords := make([]ScannerResultRecord, 0, len(data.ScannerResults))
	for _, sr := range data.ScannerResults {
		scannerRecords = append(scannerRecords, ScannerResultRecord{
			ScanID:        scanID,
			ScannerName:   sr.Scanner,
			Status:        sr.Status,
			FindingsCount: sr.FindingsCount,
			Detail:        sr.Detail,
		})
	}
	return r.store.AddScannerResults(ctx, scannerRecords)
}

// RecordFindings persists findings and instances before forge issue filing.
func (r *Recorder) RecordFindings(ctx context.Context, repositoryID int64, scanID string, codeIssues []ai.CodeIssue) (map[string]int64, error) {
	if !r.Enabled() || repositoryID == 0 || scanID == "" {
		return nil, nil
	}
	bs := r.batchStore()
	if bs == nil {
		return nil, fmt.Errorf("batch persistence not supported by store")
	}

	now := time.Now().UTC()
	expected := len(codeIssues)
	persisted, byFingerprint, err := bs.PersistScanFindingsBatch(ctx, repositoryID, scanID, codeIssues, now)
	if err != nil {
		_ = bs.UpdateScanPipelineState(ctx, scanID, ScanStatusPersistenceIncomplete, map[string]any{
			"persistence_status":          PersistenceStatusFailed,
			"persistence_expected_count":    expected,
			"persistence_persisted_count":   persisted,
			"persistence_error":           err.Error(),
			"issue_sync_status":             IssueSyncStatusSkipped,
		})
		return nil, err
	}

	fields := map[string]any{
		"persistence_status":          PersistenceStatusComplete,
		"persistence_expected_count":  expected,
		"persistence_persisted_count": persisted,
		"persistence_error":         "",
	}
	if err := bs.UpdateScanPipelineState(ctx, scanID, ScanStatusCompleted, fields); err != nil {
		return byFingerprint, fmt.Errorf("mark persistence complete: %w", err)
	}
	return byFingerprint, nil
}

// MarkPersistenceFailed records a failed persistence attempt without filing issues.
func (r *Recorder) MarkPersistenceFailed(ctx context.Context, scanID string, expected, persisted int, persistErr error) {
	bs := r.batchStore()
	if bs == nil || scanID == "" {
		return
	}
	msg := ""
	if persistErr != nil {
		msg = persistErr.Error()
	}
	_ = bs.UpdateScanPipelineState(ctx, scanID, ScanStatusPersistenceIncomplete, map[string]any{
		"persistence_status":          PersistenceStatusFailed,
		"persistence_expected_count":  expected,
		"persistence_persisted_count": persisted,
		"persistence_error":           msg,
		"issue_sync_status":             IssueSyncStatusSkipped,
	})
}

// IsPersistenceComplete checks whether finding persistence finished for a scan.
func (r *Recorder) IsPersistenceComplete(ctx context.Context, scanID string) bool {
	if !r.Enabled() || scanID == "" {
		return false
	}
	bs := r.batchStore()
	if bs == nil {
		return true
	}
	scan, err := r.store.GetScan(ctx, scanID)
	if err != nil {
		return false
	}
	pipeline := PipelineStateFromSummary(scan.SummaryJSON)
	count, err := bs.CountFindingInstancesForScan(ctx, scanID)
	if err != nil {
		return false
	}
	return pipeline.IsReconcilable(count)
}

// RecordExternalIssues links forge issues after successful persistence and filing.
func (r *Recorder) RecordExternalIssues(ctx context.Context, scanID string, forgeType string, processed []issues.ProcessedIssueRecord, findingIDs map[string]int64) error {
	if !r.Enabled() || scanID == "" || len(processed) == 0 {
		return nil
	}
	bs := r.batchStore()
	if bs == nil {
		return fmt.Errorf("batch persistence not supported by store")
	}
	if err := bs.RecordExternalIssuesBatch(ctx, scanID, forgeType, processed, findingIDs, time.Now().UTC()); err != nil {
		return err
	}
	return bs.UpdateScanPipelineState(ctx, scanID, ScanStatusCompleted, map[string]any{
		"issue_sync_status": IssueSyncStatusComplete,
	})
}

// MarkIssueSyncComplete records that the forge issue filing phase finished (including zero new links).
func (r *Recorder) MarkIssueSyncComplete(ctx context.Context, scanID string) {
	bs := r.batchStore()
	if bs == nil || scanID == "" {
		return
	}
	_ = bs.UpdateScanPipelineState(ctx, scanID, ScanStatusCompleted, map[string]any{
		"issue_sync_status": IssueSyncStatusComplete,
	})
}

// MarkIssueSyncSkipped records that forge issue filing was intentionally skipped.
func (r *Recorder) MarkIssueSyncSkipped(ctx context.Context, scanID string) {
	bs := r.batchStore()
	if bs == nil || scanID == "" {
		return
	}
	_ = bs.UpdateScanPipelineState(ctx, scanID, ScanStatusCompleted, map[string]any{
		"issue_sync_status": IssueSyncStatusSkipped,
	})
}

// RecordIssues persists findings then external issue links (legacy entry — prefer RecordFindings + RecordExternalIssues).
func (r *Recorder) RecordIssues(ctx context.Context, repositoryID int64, scanID string, forgeType string, codeIssues []ai.CodeIssue, processed []issues.ProcessedIssueRecord) error {
	findingIDs, err := r.RecordFindings(ctx, repositoryID, scanID, codeIssues)
	if err != nil {
		return err
	}
	if len(processed) == 0 {
		r.MarkIssueSyncSkipped(ctx, scanID)
		return nil
	}
	return r.RecordExternalIssues(ctx, scanID, forgeType, processed, findingIDs)
}

func mapLifecycleToStatus(lifecycle string) string {
	switch strings.ToLower(strings.TrimSpace(lifecycle)) {
	case "fixed", "closed":
		return "closed"
	default:
		return FindingStatusOpen
	}
}

func redactSnippet(value string) string {
	value = issues.SanitizeSecretEvidence(value)
	if len(value) > 2000 {
		return value[:2000] + "…"
	}
	return value
}

// ScannerResultsFromRun converts scanner run results for tests/integration.
func ScannerResultsFromRun(results []scanners.RunResult) []ScannerResultRecord {
	out := make([]ScannerResultRecord, 0, len(results))
	for _, sr := range results {
		out = append(out, ScannerResultRecord{
			ScannerName:   sr.Scanner,
			Status:        string(sr.Status),
			FindingsCount: len(sr.Findings),
			Detail:        sr.Detail,
		})
	}
	return out
}
