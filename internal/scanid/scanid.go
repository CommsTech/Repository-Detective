package scanid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey struct{}

// New generates a short correlation ID for a scan run.
func New() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// With attaches a scan ID to a context.
func With(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, id)
}

// From returns the scan ID stored in ctx, or empty if none.
func From(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}
