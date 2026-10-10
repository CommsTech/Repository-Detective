package analyzers

import "context"

type repositoryIDKey struct{}

// WithRepositoryID stores the local SQLite repository id on the analysis context.
func WithRepositoryID(ctx context.Context, repositoryID int64) context.Context {
	if repositoryID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, repositoryIDKey{}, repositoryID)
}

// RepositoryIDFrom returns the repository id from context, or 0 if unset.
func RepositoryIDFrom(ctx context.Context) int64 {
	if v, ok := ctx.Value(repositoryIDKey{}).(int64); ok {
		return v
	}
	return 0
}
