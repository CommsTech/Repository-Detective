package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/findinglearn"
	"git.commsnet.org/commstech/repository-detective/issues"
)

// FindingPersistRow is one finding + instance row for batch persistence.
type FindingPersistRow struct {
	Finding  Finding
	Evidence string
	Location json.RawMessage
	Meta     json.RawMessage
}

// CountFindingInstancesForScan returns persisted instance rows for a scan.
func (s *SQLiteStore) CountFindingInstancesForScan(ctx context.Context, scanID string) (int, error) {
	if scanID == "" {
		return 0, nil
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM finding_instances WHERE scan_id = ?`, scanID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count finding instances: %w", err)
	}
	return count, nil
}

// PersistScanFindingsBatch upserts findings and inserts instances in one transaction.
// Returns persisted instance count and fingerprint→finding_id map for issue linking.
func (s *SQLiteStore) PersistScanFindingsBatch(ctx context.Context, repositoryID int64, scanID string, codeIssues []ai.CodeIssue, now time.Time) (int, map[string]int64, error) {
	if repositoryID == 0 || scanID == "" {
		return 0, nil, fmt.Errorf("repository_id and scan_id are required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	storeRules, _ := s.ListRepoCalibrationRules(ctx, repositoryID, true)
	repoRules := make([]findinglearn.RepoCalibrationRule, 0, len(storeRules))
	for _, r := range storeRules {
		repoRules = append(repoRules, findinglearn.RepoCalibrationRule{
			Source: r.Source, RuleID: r.RuleID, PathPattern: r.PathPattern,
			Action: r.Action, Reason: r.Reason, Active: r.Active, ExpiresAt: r.ExpiresAt,
		})
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("begin findings batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	persisted := 0
	byFingerprint := make(map[string]int64, len(codeIssues))
	structuralGroups := make([]struct {
		hash      string
		findingID int64
	}, 0)
	for _, issue := range codeIssues {
		if issue.Fingerprint == "" {
			continue
		}

		pathInput := findinglearn.ClassifyPath(issue.File)
		severity, confidence, reachNote := findinglearn.ActionabilityAdjust(issue.Severity, issue.Confidence, pathInput)
		if len(repoRules) > 0 {
			if sev, conf, calNote := findinglearn.ApplyRepoRules(severity, confidence, issue.Source, issue.RuleID, issue.File, repoRules); calNote != "" {
				severity, confidence = sev, conf
				if reachNote != "" {
					reachNote = reachNote + " " + calNote
				} else {
					reachNote = calNote
				}
			}
		}

		finding := Finding{
			RepositoryID:    repositoryID,
			Fingerprint:     issue.Fingerprint,
			Category:        issue.Category,
			Severity:        severity,
			Confidence:      confidence,
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
		}
		if reachNote != "" {
			finding.CalibrationNote = reachNote
		}

		findingID, err := upsertFindingTx(ctx, tx, finding)
		if err != nil {
			return persisted, byFingerprint, fmt.Errorf("upsert finding %s: %w", issue.Fingerprint, err)
		}
		byFingerprint[issue.Fingerprint] = findingID

		snippet := issue.CodeSnippet
		if snippet == "" {
			snippet = issue.Description
		}
		if hash := findinglearn.StructuralHash(issue.RuleID, issue.Category, snippet); hash != "" {
			structuralGroups = append(structuralGroups, struct {
				hash      string
				findingID int64
			}{hash: hash, findingID: findingID})
		}

		locationJSON, err := json.Marshal(map[string]any{
			"file":         issue.File,
			"line":         issue.LineNumber,
			"column":       issue.ColumnNumber,
			"code_snippet": redactSnippet(issue.CodeSnippet),
		})
		if err != nil {
			return persisted, byFingerprint, fmt.Errorf("marshal location %s: %w", issue.Fingerprint, err)
		}
		metaJSON, err := json.Marshal(map[string]any{
			"source":  issue.Source,
			"rule_id": issue.RuleID,
			"from_ai": issue.FromAI,
			"fixable": issue.Fixable,
			"scan_id": scanID,
		})
		if err != nil {
			return persisted, byFingerprint, fmt.Errorf("marshal meta %s: %w", issue.Fingerprint, err)
		}
		if issue.Source == "graph" && issue.Evidence != "" {
			var meta map[string]any
			if err := json.Unmarshal(metaJSON, &meta); err != nil {
				return persisted, byFingerprint, fmt.Errorf("unmarshal meta %s: %w", issue.Fingerprint, err)
			}
			meta["graph_detail"] = json.RawMessage(issue.Evidence)
			metaJSON, err = json.Marshal(meta)
			if err != nil {
				return persisted, byFingerprint, fmt.Errorf("marshal graph meta %s: %w", issue.Fingerprint, err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO finding_instances (finding_id, scan_id, evidence_redacted, location_json, raw_metadata_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, findingID, scanID, redactSnippet(issue.Description), string(locationJSON), string(metaJSON), formatTime(now)); err != nil {
			return persisted, byFingerprint, fmt.Errorf("insert instance %s: %w", issue.Fingerprint, err)
		}
		persisted++
	}

	if err := tx.Commit(); err != nil {
		return persisted, byFingerprint, fmt.Errorf("commit findings batch: %w", err)
	}
	for _, g := range structuralGroups {
		_ = s.AssignStructuralGroup(ctx, repositoryID, g.hash, g.findingID)
	}
	return persisted, byFingerprint, nil
}

func upsertFindingTx(ctx context.Context, tx *sql.Tx, finding Finding) (int64, error) {
	var existingID int64
	var firstSeen, firstSeenScan string
	err := tx.QueryRowContext(ctx, `
		SELECT id, first_seen_at, first_seen_scan_id FROM findings
		WHERE repository_id = ? AND fingerprint = ?
	`, finding.RepositoryID, finding.Fingerprint).Scan(&existingID, &firstSeen, &firstSeenScan)
	if err == nil {
		finding.ID = existingID
		finding.FirstSeenAt = parseTime(firstSeen)
		finding.FirstSeenScanID = firstSeenScan
	} else if !isNoRows(err) {
		return 0, err
	}
	if finding.FirstSeenScanID == "" {
		finding.FirstSeenScanID = finding.LastSeenScanID
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO findings (
			repository_id, fingerprint, category, severity, confidence, source, rule_id,
			package_name, file_path, line, title, status,
			first_seen_scan_id, last_seen_scan_id, first_seen_at, last_seen_at, calibration_note
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id, fingerprint) DO UPDATE SET
			category = excluded.category,
			severity = excluded.severity,
			confidence = excluded.confidence,
			source = excluded.source,
			rule_id = excluded.rule_id,
			package_name = excluded.package_name,
			file_path = excluded.file_path,
			line = excluded.line,
			title = excluded.title,
			status = excluded.status,
			last_seen_scan_id = excluded.last_seen_scan_id,
			last_seen_at = excluded.last_seen_at,
			calibration_note = CASE
				WHEN excluded.calibration_note != '' THEN excluded.calibration_note
				ELSE findings.calibration_note
			END
	`, finding.RepositoryID, finding.Fingerprint, finding.Category, finding.Severity, finding.Confidence,
		finding.Source, finding.RuleID, finding.PackageName, finding.FilePath, finding.Line, finding.Title, finding.Status,
		finding.FirstSeenScanID, finding.LastSeenScanID, formatTime(finding.FirstSeenAt), formatTime(finding.LastSeenAt), finding.CalibrationNote)
	if err != nil {
		return 0, err
	}

	if finding.ID != 0 {
		return finding.ID, nil
	}
	var id int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM findings WHERE repository_id = ? AND fingerprint = ?
	`, finding.RepositoryID, finding.Fingerprint).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// RecordExternalIssuesBatch links forge issues after persistence and optional issue filing.
func (s *SQLiteStore) RecordExternalIssuesBatch(ctx context.Context, scanID string, forgeType string, processed []issues.ProcessedIssueRecord, findingIDs map[string]int64, now time.Time) error {
	if scanID == "" || len(processed) == 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ft := forgeType
	if ft == "" {
		ft = ForgeTypeGitea
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin external issues batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range processed {
		if item.IssueNumber <= 0 || item.Fingerprint == "" {
			continue
		}
		findingID, ok := findingIDs[item.Fingerprint]
		if !ok || findingID == 0 {
			continue
		}
		itemForge := ft
		if item.ForgeType != "" {
			itemForge = item.ForgeType
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO external_issues (finding_id, forge_type, issue_number, issue_url, state, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(finding_id, forge_type, issue_number) DO UPDATE SET
				issue_url = excluded.issue_url,
				state = excluded.state,
				updated_at = excluded.updated_at
		`, findingID, itemForge, item.IssueNumber, item.IssueURL, "open", formatTime(now), formatTime(now)); err != nil {
			return fmt.Errorf("upsert external issue #%d: %w", item.IssueNumber, err)
		}

		eventType := "issue_created"
		if item.Action == "updated" {
			eventType = "issue_updated"
		}
		metaJSON, err := json.Marshal(map[string]any{
			"issue_number": item.IssueNumber,
			"issue_url":    item.IssueURL,
			"action":       item.Action,
		})
		if err != nil {
			return fmt.Errorf("marshal lifecycle meta: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO lifecycle_events (finding_id, scan_id, event_type, message, metadata_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, findingID, scanID, eventType,
			fmt.Sprintf("forge issue #%d (%s)", item.IssueNumber, item.Action),
			string(metaJSON), formatTime(now)); err != nil {
			return fmt.Errorf("add lifecycle event: %w", err)
		}
	}

	return tx.Commit()
}
