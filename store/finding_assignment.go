package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// FindingAssignment is local work ownership, not forge issue assignment.
// AssigneeUserID=0 means unassigned.
type FindingAssignment struct {
	FindingID      int64
	AssigneeUserID int64
	AssigneeName   string
	UpdatedAt      time.Time
}

// FindingAssignmentEvent records who changed local ownership. It is append-only.
type FindingAssignmentEvent struct {
	ID           int64
	FindingID    int64
	ActorName    string
	PreviousName string
	NewName      string
	CreatedAt    time.Time
}

func (s *SQLiteStore) GetFindingAssignment(ctx context.Context, findingID int64) (FindingAssignment, error) {
	out := FindingAssignment{FindingID: findingID}
	var when string
	err := s.db.QueryRowContext(ctx, `
        SELECT COALESCE(a.assignee_user_id, 0), COALESCE(u.display_name, ''), a.updated_at
        FROM finding_assignments a
        LEFT JOIN users u ON a.assignee_user_id = u.id
        WHERE a.finding_id = ?`, findingID).Scan(&out.AssigneeUserID, &out.AssigneeName, &when)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("get finding assignment: %w", err)
	}
	out.UpdatedAt = parseTime(when)
	return out, nil
}

func (s *SQLiteStore) ListFindingAssignmentEvents(ctx context.Context, findingID int64, limit int) ([]FindingAssignmentEvent, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, finding_id, actor_name, previous_name, new_name, created_at
        FROM finding_assignment_events
        WHERE finding_id = ? ORDER BY id DESC LIMIT ?`, findingID, limit)
	if err != nil {
		return nil, fmt.Errorf("list finding assignment events: %w", err)
	}
	defer closeRows(rows)
	var out []FindingAssignmentEvent
	for rows.Next() {
		var e FindingAssignmentEvent
		var ts string
		if err := rows.Scan(&e.ID, &e.FindingID, &e.ActorName, &e.PreviousName, &e.NewName, &ts); err != nil {
			return nil, fmt.Errorf("scan assignment event: %w", err)
		}
		e.CreatedAt = parseTime(ts)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignment events: %w", err)
	}
	return out, nil
}

// SetFindingAssignee performs an atomic local assignment and event write.
// It never modifies an external forge issue or sends notifications.
// actorUserID must be an authenticated enabled user with a write-capable role.
func (s *SQLiteStore) SetFindingAssignee(ctx context.Context, findingID, assigneeUserID, actorUserID int64) error {
	if findingID <= 0 || actorUserID <= 0 || assigneeUserID < 0 {
		return fmt.Errorf("invalid assignment identifiers")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin assignment: %w", err)
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings WHERE id = ?`, findingID).Scan(&exists); err != nil {
		return err
	}
	if exists != 1 {
		return fmt.Errorf("finding not found")
	}
	var role string
	var actorName string
	var enabled int
	if err := tx.QueryRowContext(ctx, `SELECT role, enabled, display_name FROM users WHERE id = ?`, actorUserID).Scan(&role, &enabled, &actorName); err != nil {
		return fmt.Errorf("assignment actor not found: %w", err)
	}
	if enabled != 1 || !RoleCanWriteSecurity(role) {
		return fmt.Errorf("assignment requires active owner, admin, or security role")
	}
	newName := "Unassigned"
	if assigneeUserID > 0 {
		if err := tx.QueryRowContext(ctx, `SELECT enabled, display_name FROM users WHERE id = ?`, assigneeUserID).Scan(&enabled, &newName); err != nil {
			return fmt.Errorf("assignee does not exist: %w", err)
		}
		if enabled != 1 {
			return fmt.Errorf("assignee is disabled")
		}
	}
	var prev sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT assignee_user_id FROM finding_assignments WHERE finding_id = ?`, findingID).Scan(&prev)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read prior assignment: %w", err)
	}
	oldID := int64(0)
	if prev.Valid {
		oldID = prev.Int64
	}
	if oldID == assigneeUserID {
		return tx.Commit()
	}
	previousName := "Unassigned"
	if oldID > 0 {
		if err := tx.QueryRowContext(ctx, `SELECT display_name FROM users WHERE id = ?`, oldID).Scan(&previousName); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read previous owner: %w", err)
		}
	}
	now := formatTime(time.Now().UTC())
	_, err = tx.ExecContext(ctx, `INSERT INTO finding_assignments(finding_id, assignee_user_id, assigned_by_user_id, updated_at)
         VALUES(?, NULLIF(?, 0), ?, ?)
         ON CONFLICT(finding_id) DO UPDATE SET assignee_user_id=excluded.assignee_user_id,
         assigned_by_user_id=excluded.assigned_by_user_id, updated_at=excluded.updated_at`,
		findingID, assigneeUserID, actorUserID, now)
	if err != nil {
		return fmt.Errorf("persist assignment: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO finding_assignment_events
         (finding_id, actor_user_id, actor_name, previous_user_id, previous_name, new_user_id, new_name, created_at)
         VALUES(?, ?, ?, NULLIF(?, 0), ?, NULLIF(?, 0), ?, ?)`,
		findingID, actorUserID, actorName, oldID, previousName, assigneeUserID, newName, now)
	if err != nil {
		return fmt.Errorf("record assignment event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit assignment: %w", err)
	}
	return nil
}
