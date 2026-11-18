package store

import (
	"context"
	"fmt"
	"strings"
)

// annotateFindingOwners loads ownership for an entire result page in one query.
// This avoids N+1 database traffic when the local team opens the findings queue.
// The caller explicitly opts in to team metadata through IncludeOwner.
func (s *SQLiteStore) annotateFindingOwners(ctx context.Context, items []FindingListItem) ([]FindingListItem, error) {
	if len(items) == 0 {
		return items, nil
	}
	args := make([]any, 0, len(items))
	positions := make(map[int64]int, len(items))
	for i, f := range items {
		args = append(args, f.ID)
		positions[f.ID] = i
	}
	query := `SELECT fa.finding_id, COALESCE(fa.assignee_user_id, 0), COALESCE(u.display_name, '')
		FROM finding_assignments fa
		LEFT JOIN users u ON u.id = fa.assignee_user_id
		WHERE fa.finding_id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(args)), ",") + `)`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load finding owners: %w", err)
	}
	defer closeRows(rows)
	for rows.Next() {
		var id, ownerID int64
		var name string
		if err := rows.Scan(&id, &ownerID, &name); err != nil {
			return nil, fmt.Errorf("scan finding owner: %w", err)
		}
		if pos, exists := positions[id]; exists {
			items[pos].AssigneeUserID = ownerID
			items[pos].AssigneeName = name
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finding owners: %w", err)
	}
	return items, nil
}
