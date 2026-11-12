package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SQLiteStore) SaveRemediationPlan(ctx context.Context, plan RemediationPlanRecord) (RemediationPlanRecord, error) {
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	if plan.Status == "" {
		plan.Status = RemediationStatusProposed
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO remediation_plans (
			plan_id, finding_id, repository_id, audit_id, fingerprint,
			category, severity, source, rule_id, title, summary, fix_strategy,
			affected_files_json, required_tests_json, validation_commands_json,
			regression_risk, fix_complexity, safe_for_auto_pr, requires_human_review,
			blocked_reasons_json, advisory, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		plan.PlanID,
		nullInt64Value(plan.FindingID),
		nullInt64Value(plan.RepositoryID),
		nullStringValue(plan.AuditID),
		plan.Fingerprint,
		plan.Category, plan.Severity, plan.Source, plan.RuleID,
		plan.Title, plan.Summary, plan.FixStrategy,
		stringOrEmptyJSON(plan.AffectedFilesJSON),
		stringOrEmptyJSON(plan.RequiredTestsJSON),
		stringOrEmptyJSON(plan.ValidationCommandsJSON),
		plan.RegressionRisk, plan.FixComplexity,
		boolToInt(plan.SafeForAutoPR), boolToInt(plan.RequiresHumanReview),
		stringOrEmptyJSON(plan.BlockedReasonsJSON),
		boolToInt(plan.Advisory), plan.Status,
		formatTime(plan.CreatedAt), formatTime(plan.UpdatedAt),
	)
	if err != nil {
		return RemediationPlanRecord{}, fmt.Errorf("insert remediation plan: %w", err)
	}
	id, _ := res.LastInsertId()
	plan.ID = id
	return plan, nil
}

func (s *SQLiteStore) GetRemediationPlanByPlanID(ctx context.Context, planID string) (RemediationPlanRecord, error) {
	row := s.db.QueryRowContext(ctx, remediationPlanSelect+` WHERE plan_id = ?`, planID)
	plan, err := scanRemediationPlan(row)
	if err != nil {
		return RemediationPlanRecord{}, fmt.Errorf("get remediation plan: %w", err)
	}
	return plan, nil
}

func (s *SQLiteStore) GetLatestRemediationPlanByFindingID(ctx context.Context, findingID int64) (RemediationPlanRecord, error) {
	row := s.db.QueryRowContext(ctx, remediationPlanSelect+`
		WHERE finding_id = ? AND status != ?
		ORDER BY created_at DESC LIMIT 1
	`, findingID, RemediationStatusSuperseded)
	plan, err := scanRemediationPlan(row)
	if err != nil {
		return RemediationPlanRecord{}, fmt.Errorf("get latest remediation plan: %w", err)
	}
	return plan, nil
}

func (s *SQLiteStore) UpdateRemediationPlanStatus(ctx context.Context, planID, status string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE remediation_plans SET status = ?, updated_at = ? WHERE plan_id = ?
	`, status, formatTime(now), planID)
	if err != nil {
		return fmt.Errorf("update remediation plan status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("remediation plan not found")
	}
	return nil
}

func (s *SQLiteStore) SupersedeRemediationPlansForFinding(ctx context.Context, findingID int64) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE remediation_plans SET status = ?, updated_at = ?
		WHERE finding_id = ? AND status = ?
	`, RemediationStatusSuperseded, formatTime(now), findingID, RemediationStatusProposed)
	if err != nil {
		return fmt.Errorf("supersede remediation plans: %w", err)
	}
	return nil
}

func (s *SQLiteStore) RemediationSummary(ctx context.Context) (RemediationSummary, error) {
	var summary RemediationSummary
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = ? AND safe_for_auto_pr = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? AND requires_human_review = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM remediation_plans
	`, RemediationStatusProposed, RemediationStatusProposed, RemediationStatusApproved)
	if err := row.Scan(&summary.Candidates, &summary.HumanReview, &summary.ApprovedWaiting); err != nil {
		if err == sql.ErrNoRows {
			return RemediationSummary{}, nil
		}
		return RemediationSummary{}, fmt.Errorf("remediation summary: %w", err)
	}
	return summary, nil
}

const remediationPlanSelect = `
	SELECT id, plan_id, finding_id, repository_id, audit_id, fingerprint,
		category, severity, source, rule_id, title, summary, fix_strategy,
		affected_files_json, required_tests_json, validation_commands_json,
		regression_risk, fix_complexity, safe_for_auto_pr, requires_human_review,
		blocked_reasons_json, advisory, status, created_at, updated_at
	FROM remediation_plans
`

func scanRemediationPlan(row *sql.Row) (RemediationPlanRecord, error) {
	var plan RemediationPlanRecord
	var findingID, repositoryID sql.NullInt64
	var auditID sql.NullString
	var affected, tests, commands, blocked string
	var safe, human, advisory int
	var created, updated string
	err := row.Scan(
		&plan.ID, &plan.PlanID, &findingID, &repositoryID, &auditID, &plan.Fingerprint,
		&plan.Category, &plan.Severity, &plan.Source, &plan.RuleID,
		&plan.Title, &plan.Summary, &plan.FixStrategy,
		&affected, &tests, &commands,
		&plan.RegressionRisk, &plan.FixComplexity, &safe, &human, &blocked, &advisory,
		&plan.Status, &created, &updated,
	)
	if err != nil {
		return RemediationPlanRecord{}, err
	}
	if findingID.Valid {
		v := findingID.Int64
		plan.FindingID = &v
	}
	if repositoryID.Valid {
		v := repositoryID.Int64
		plan.RepositoryID = &v
	}
	if auditID.Valid {
		v := auditID.String
		plan.AuditID = &v
	}
	plan.AffectedFilesJSON = json.RawMessage(affected)
	plan.RequiredTestsJSON = json.RawMessage(tests)
	plan.ValidationCommandsJSON = json.RawMessage(commands)
	plan.BlockedReasonsJSON = json.RawMessage(blocked)
	plan.SafeForAutoPR = safe != 0
	plan.RequiresHumanReview = human != 0
	plan.Advisory = advisory != 0
	plan.CreatedAt = parseTime(created)
	plan.UpdatedAt = parseTime(updated)
	return plan, nil
}

func nullInt64Value(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullStringValue(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func stringOrEmptyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "[]"
	}
	return string(raw)
}
