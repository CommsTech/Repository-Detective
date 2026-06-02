package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/issues"
	"git.commsnet.org/commstech/bugbot/scanners"
	"github.com/sirupsen/logrus"
)

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

// FinishScan records scan completion, scanner results, and optional summary.
func (r *Recorder) FinishScan(ctx context.Context, scanID string, data *ScanCompletion, analysisErr error) error {
	if !r.Enabled() || scanID == "" {
		return nil
	}

	status := ScanStatusCompleted
	errMsg := ""
	if analysisErr != nil {
		status = ScanStatusFailed
		errMsg = analysisErr.Error()
	}

	summary := map[string]any{
		"issues_found":     0,
		"files_analyzed":   0,
		"analysis_time_ms": 0,
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

	if data != nil {
		summary["issues_found"] = data.IssuesFound
		summary["files_analyzed"] = data.FilesAnalyzed
		summary["analysis_time_ms"] = data.AnalysisTime.Milliseconds()
		summary["overall_score"] = data.OverallScore
		commitSHA = data.CommitSHA
		if data.RepoProfile != nil {
			summary["repo_profile"] = data.RepoProfile
		}
		if data.GraphNodeCount > 0 {
			summary["graph_nodes"] = data.GraphNodeCount
			summary["graph_edges"] = data.GraphEdgeCount
		}
	}

	summaryJSON, _ := json.Marshal(summary)

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
			}
		}
	}

	if data == nil {
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

// RecordIssues persists findings, instances, external issue links, and lifecycle events.
func (r *Recorder) RecordIssues(ctx context.Context, repositoryID int64, scanID string, codeIssues []ai.CodeIssue, processed []issues.ProcessedIssueRecord) error {
	if !r.Enabled() || repositoryID == 0 || scanID == "" {
		return nil
	}

	processedByFingerprint := make(map[string]issues.ProcessedIssueRecord, len(processed))
	for _, item := range processed {
		if item.Fingerprint != "" {
			processedByFingerprint[item.Fingerprint] = item
		}
	}

	now := time.Now().UTC()
	for _, issue := range codeIssues {
		if issue.Fingerprint == "" {
			continue
		}

		stored, err := r.store.UpsertFinding(ctx, Finding{
			RepositoryID:    repositoryID,
			Fingerprint:     issue.Fingerprint,
			Category:        issue.Category,
			Severity:        issue.Severity,
			Confidence:      issue.Confidence,
			Source:          issue.Source,
			RuleID:          issue.RuleID,
			PackageName:     issue.PackageName,
			FilePath:        issue.File,
			Line:            issue.LineNumber,
			Title:           issue.Title,
			Status:          mapLifecycleToStatus(issue.LifecycleState),
			FirstSeenScanID: scanID,
			LastSeenScanID:  scanID,
			FirstSeenAt:     now,
			LastSeenAt:      now,
		})
		if err != nil {
			return fmt.Errorf("upsert finding %s: %w", issue.Fingerprint, err)
		}

		locationJSON, _ := json.Marshal(map[string]any{
			"file":         issue.File,
			"line":         issue.LineNumber,
			"column":       issue.ColumnNumber,
			"code_snippet": redactSnippet(issue.CodeSnippet),
		})
		metaJSON, _ := json.Marshal(map[string]any{
			"source":  issue.Source,
			"rule_id": issue.RuleID,
			"from_ai": issue.FromAI,
			"fixable": issue.Fixable,
			"scan_id": scanID,
		})

		if err := r.store.AddFindingInstance(ctx, FindingInstance{
			FindingID:        stored.ID,
			ScanID:           scanID,
			EvidenceRedacted: redactSnippet(issue.Description),
			LocationJSON:     locationJSON,
			RawMetadataJSON:  metaJSON,
			CreatedAt:        now,
		}); err != nil {
			return fmt.Errorf("add finding instance %s: %w", issue.Fingerprint, err)
		}

		if processedItem, ok := processedByFingerprint[issue.Fingerprint]; ok && processedItem.IssueNumber > 0 {
			if _, err := r.store.UpsertExternalIssue(ctx, ExternalIssue{
				FindingID:   stored.ID,
				ForgeType:   ForgeTypeGitea,
				IssueNumber: processedItem.IssueNumber,
				IssueURL:    processedItem.IssueURL,
				State:       "open",
			}); err != nil {
				return fmt.Errorf("upsert external issue: %w", err)
			}

			findingID := stored.ID
			eventType := "issue_created"
			if processedItem.Action == "updated" {
				eventType = "issue_updated"
			}
			if err := r.store.AddLifecycleEvent(ctx, LifecycleEvent{
				FindingID: &findingID,
				ScanID:    scanID,
				EventType: eventType,
				Message:   fmt.Sprintf("forge issue #%d (%s)", processedItem.IssueNumber, processedItem.Action),
				MetadataJSON: mustJSON(map[string]any{
					"issue_number": processedItem.IssueNumber,
					"issue_url":    processedItem.IssueURL,
					"action":       processedItem.Action,
				}),
				CreatedAt: now,
			}); err != nil {
				return fmt.Errorf("add lifecycle event: %w", err)
			}
		}
	}

	return nil
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

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
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
