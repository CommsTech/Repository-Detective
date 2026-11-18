package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestFindingAssignmentAuthorizedAndAudited(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "assign.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	r, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "app", FullName: "team/app"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: r.ID, Fingerprint: "assignment-test", Title: "Review this", Status: "open", Severity: "high", Category: "security", Source: "gosec", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	create := func(email, role string, enabled bool) store.User {
		t.Helper()
		u, err := s.CreateUser(ctx, store.User{Email: email, DisplayName: email, PasswordHash: "not-used", Role: role, Enabled: enabled})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	owner := create("owner@example.org", store.RoleOwner, true)
	developer := create("developer@example.org", store.RoleDeveloper, true)
	viewer := create("viewer@example.org", store.RoleViewer, true)
	disabled := create("disabled@example.org", store.RoleDeveloper, false)
	original, err := s.GetFindingAssignment(ctx, f.ID)
	if err != nil || original.AssigneeUserID != 0 {
		t.Fatalf("initial assignment: %+v %v", original, err)
	}
	if err := s.SetFindingAssignee(ctx, f.ID, developer.ID, viewer.ID); err == nil {
		t.Fatal("viewer must not assign findings")
	}
	if err := s.SetFindingAssignee(ctx, f.ID, disabled.ID, owner.ID); err == nil {
		t.Fatal("disabled user must not be assignee")
	}
	if err := s.SetFindingAssignee(ctx, f.ID, developer.ID, owner.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFindingAssignee(ctx, f.ID, developer.ID, owner.ID); err != nil {
		t.Fatal(err)
	} // idempotent
	assigned, err := s.GetFindingAssignment(ctx, f.ID)
	if err != nil || assigned.AssigneeUserID != developer.ID || assigned.AssigneeName == "" {
		t.Fatalf("assigned: %+v %v", assigned, err)
	}
	events, err := s.ListFindingAssignmentEvents(ctx, f.ID, 20)
	if err != nil || len(events) != 1 {
		t.Fatalf("events after duplicate: %d %v", len(events), err)
	}
	if events[0].PreviousName != "Unassigned" || events[0].NewName != developer.DisplayName {
		t.Fatalf("event: %+v", events[0])
	}
	if err := s.SetFindingAssignee(ctx, f.ID, 0, owner.ID); err != nil {
		t.Fatal(err)
	}
	unassigned, err := s.GetFindingAssignment(ctx, f.ID)
	if err != nil || unassigned.AssigneeUserID != 0 {
		t.Fatalf("unassigned: %+v %v", unassigned, err)
	}
	events, err = s.ListFindingAssignmentEvents(ctx, f.ID, 20)
	if err != nil || len(events) != 2 || events[0].NewName != "Unassigned" {
		t.Fatalf("history: %+v %v", events, err)
	}
	if err := s.SetFindingAssignee(ctx, f.ID+999, developer.ID, owner.ID); err == nil {
		t.Fatal("unknown finding must fail")
	}
}
