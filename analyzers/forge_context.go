package analyzers

import "context"

type forgeTypeKey struct{}

// WithForgeType stores the source forge (gitea, github) on the analysis context.
func WithForgeType(ctx context.Context, forgeType string) context.Context {
	if forgeType == "" {
		return ctx
	}
	return context.WithValue(ctx, forgeTypeKey{}, forgeType)
}

// ForgeTypeFrom returns the forge type from context, defaulting to gitea.
func ForgeTypeFrom(ctx context.Context) string {
	if v, ok := ctx.Value(forgeTypeKey{}).(string); ok && v != "" {
		return v
	}
	return "gitea"
}
