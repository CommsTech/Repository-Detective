package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingSearchMatchesCountAndListAndEscapesWildcards(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repoA, err := s.UpsertRepository(ctx, store.Repository{Owner: "ops", Name: "alpha", FullName: "ops/alpha"})
	if err != nil {
		t.Fatal(err)
	}
	repoB, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "beta", FullName: "team/beta"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	examples := []store.Finding{
		{RepositoryID: repoA.ID, Fingerprint: "fp-a", Title: "Outdated encryption library", RuleID: "CVE-2026-1000", PackageName: "openssl", FilePath: "modules/tls/main.go", Severity: "critical", Status: "open"},
		{RepositoryID: repoB.ID, Fingerprint: "fp-b", Title: "100% reproducible check", RuleID: "LINT-ONE", PackageName: "foo", FilePath: "app/manifest.txt", Severity: "medium", Status: "open"},
		{RepositoryID: repoB.ID, Fingerprint: "fp-c", Title: "literal_underscore", RuleID: "LINT-TWO", PackageName: "bar", FilePath: "other/file", Severity: "low", Status: "resolved"},
	}
	for i, f := range examples {
		f.FirstSeenAt, f.LastSeenAt = now.Add(time.Duration(i)*time.Second), now.Add(time.Duration(i)*time.Second)
		if _, err := s.UpsertFinding(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		search      string
		fingerprint string
	}{
		{"CVE-2026-1000", "fp-a"},
		{"openssl", "fp-a"},
		{"ops/alpha", "fp-a"},
		{"main.go", "fp-a"},
		{"FP-A", "fp-a"},
		{"%", "fp-b"},
		{"_", "fp-c"},
		{"' OR 1=1 --", ""},
		{"nonexistent", ""},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("search=%q", tc.search), func(t *testing.T) {
			filter := store.FindingFilter{Search: tc.search, IncludeSuppressed: true, Limit: 10}
			count, err := s.CountFindings(ctx, filter)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := s.ListFindings(ctx, filter)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(rows) {
				t.Fatalf("count %d != list %d", count, len(rows))
			}
			if tc.fingerprint == "" {
				if count != 0 {
					t.Fatalf("unexpected match for %q: %+v", tc.search, rows)
				}
			} else if count != 1 || rows[0].Fingerprint != tc.fingerprint {
				t.Fatalf("search %q returned %+v", tc.search, rows)
			}
		})
	}
	combined := store.FindingFilter{Search: "openssl", Status: "open", PriorityOnly: true, Prioritize: true, Limit: 1}
	n, err := s.CountFindings(ctx, combined)
	if err != nil || n != 1 {
		t.Fatalf("combined filter count %d, %v", n, err)
	}
	combined.Status = "resolved"
	n, err = s.CountFindings(ctx, combined)
	if err != nil || n != 0 {
		t.Fatalf("resolved with priority should have no results, got %d, %v", n, err)
	}
}
