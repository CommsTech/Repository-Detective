package store

import (
	"context"
	"fmt"
)

// CountConnectedRepositories returns how many repositories are marked connected.
func CountConnectedRepositories(ctx context.Context, s Store) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("store is nil")
	}
	if counter, ok := s.(interface {
		CountConnectedRepositories(ctx context.Context) (int, error)
	}); ok {
		return counter.CountConnectedRepositories(ctx)
	}
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range repos {
		if r.ConnectedRepo {
			n++
		}
	}
	return n, nil
}

// CountConnectedRepositories counts connected_repo=1 rows.
func (s *SQLiteStore) CountConnectedRepositories(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM repositories WHERE connected_repo = 1`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count connected repositories: %w", err)
	}
	return n, nil
}
