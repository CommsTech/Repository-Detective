package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingOwnerQueueFiltersCountsAndPagination(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "owner-queue.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "app", FullName: "team/app"})
	if err != nil {
		t.Fatal(err)
	}
	makeUser := func(name string) store.User {
		t.Helper()
		u, err := s.CreateUser(ctx, store.User{Email: name + "@example.org", DisplayName: name, Role: store.RoleOwner, PasswordHash: "not-used", Enabled: true})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	alice := makeUser("alice")
	bob := makeUser("bob")
	for i, assignment := range []int64{alice.ID, bob.ID, 0, alice.ID} {
		f, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "owner-queue-" + string(rune('a'+i)), Title: "Example", Severity: "high", Category: "security", Source: "gosec", Status: "open", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()})
		if err != nil {
			t.Fatal(err)
		}
		if assignment > 0 {
			if err := s.SetFindingAssignee(ctx, f.ID, assignment, alice.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	cases := []struct {
		name   string
		filter store.FindingFilter
		count  int
	}{
		{"all", store.FindingFilter{Status: "open", IncludeOwner: true, Limit: 10}, 4},
		{"mine-alice", store.FindingFilter{OwnerScope: "mine", AssigneeUserID: alice.ID, Status: "open", IncludeOwner: true, Limit: 10}, 2},
		{"mine-bob", store.FindingFilter{OwnerScope: "mine", AssigneeUserID: bob.ID, Status: "open", IncludeOwner: true, Limit: 10}, 1},
		{"unassigned", store.FindingFilter{OwnerScope: "unassigned", Status: "open", IncludeOwner: true, Limit: 10}, 1},
		{"mine-without-identity", store.FindingFilter{OwnerScope: "mine", IncludeOwner: true, Limit: 10}, 0},
		{"unknown-scope", store.FindingFilter{OwnerScope: "arbitrary", IncludeOwner: true, Limit: 10}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			count, err := s.CountFindings(ctx, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.ListFindings(ctx, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			if count != tc.count || len(got) != tc.count {
				t.Fatalf("count %d / records %d, expected %d", count, len(got), tc.count)
			}
			for _, f := range got {
				if tc.filter.OwnerScope == "mine" && f.AssigneeUserID != tc.filter.AssigneeUserID {
					t.Fatalf("foreign assignment: %+v", f)
				}
				if tc.filter.OwnerScope == "unassigned" && f.AssigneeUserID != 0 {
					t.Fatalf("assigned finding in unassigned: %+v", f)
				}
			}
		})
	}
	page := store.FindingFilter{OwnerScope: "mine", AssigneeUserID: alice.ID, IncludeOwner: true, Limit: 1, Prioritize: true}
	first, err := s.ListFindings(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	page.Offset = 1
	second, err := s.ListFindings(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 || first[0].ID == second[0].ID {
		t.Fatalf("pagination: %+v %+v", first, second)
	}
}
