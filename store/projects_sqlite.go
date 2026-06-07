package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ListProjectGroups returns all project groups with member repository IDs.
func (s *SQLiteStore) ListProjectGroups(ctx context.Context) ([]ProjectGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, primary_repository_id, created_at, updated_at
		FROM project_groups ORDER BY name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("list project groups: %w", err)
	}
	defer rows.Close()
	var out []ProjectGroup
	var ids []int64
	for rows.Next() {
		var g ProjectGroup
		var created, updated string
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.PrimaryRepositoryID, &created, &updated); err != nil {
			return nil, err
		}
		g.CreatedAt = parseTime(created)
		g.UpdatedAt = parseTime(updated)
		out = append(out, g)
		ids = append(ids, g.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, groupID := range ids {
		repoIDs, err := s.listProjectGroupRepoIDs(ctx, groupID)
		if err != nil {
			return nil, err
		}
		out[i].RepositoryIDs = repoIDs
	}
	return out, nil
}

func (s *SQLiteStore) listProjectGroupRepoIDs(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repository_id FROM project_group_repositories WHERE project_group_id = ? ORDER BY repository_id
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CreateProjectGroup persists a new project group and members.
func (s *SQLiteStore) CreateProjectGroup(ctx context.Context, g ProjectGroup) (ProjectGroup, error) {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		return ProjectGroup{}, fmt.Errorf("project group name required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProjectGroup{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO project_groups (name, description, primary_repository_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, name, strings.TrimSpace(g.Description), nullInt64(g.PrimaryRepositoryID), now, now)
	if err != nil {
		return ProjectGroup{}, fmt.Errorf("create project group: %w", err)
	}
	id, _ := res.LastInsertId()
	for _, repoID := range g.RepositoryIDs {
		if repoID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO project_group_repositories (project_group_id, repository_id, created_at)
			VALUES (?, ?, ?)
		`, id, repoID, now); err != nil {
			return ProjectGroup{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ProjectGroup{}, err
	}
	g.ID = id
	g.Name = name
	g.CreatedAt = parseTime(now)
	g.UpdatedAt = g.CreatedAt
	return g, nil
}
