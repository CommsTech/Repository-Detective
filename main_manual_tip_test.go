package main

import (
	"context"
	"errors"
	"testing"
)

type stubTipResolver struct {
	ref string
	sha string
	err error
}

func (s stubTipResolver) ResolveRef(context.Context, string, string, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.ref, nil
}

func (s stubTipResolver) ResolveTipCommit(context.Context, string, string, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.sha, nil
}

func TestResolvePinnedRefAndTip(t *testing.T) {
	ctx := context.Background()
	ref, sha := resolvePinnedRefAndTip(ctx, stubTipResolver{ref: "refs/heads/main", sha: "abc123abc123abc123abc123abc123abc123abcd"}, "o", "r", "")
	if ref != "refs/heads/main" || sha != "abc123abc123abc123abc123abc123abc123abcd" {
		t.Fatalf("ref=%q sha=%q", ref, sha)
	}

	ref, sha = resolvePinnedRefAndTip(ctx, nil, "o", "r", "develop")
	if ref != "develop" || sha != "" {
		t.Fatalf("nil client ref=%q sha=%q", ref, sha)
	}

	ref, sha = resolvePinnedRefAndTip(ctx, stubTipResolver{err: errors.New("offline")}, "o", "r", "main")
	if ref != "main" || sha != "" {
		t.Fatalf("resolve error should keep the requested ref, got ref=%q sha=%q", ref, sha)
	}
}
