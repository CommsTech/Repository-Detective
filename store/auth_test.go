package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestAuthMigrationAndBootstrapUser(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auth.db")
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	count, err := s.CountUsers(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users, got %d", count)
	}

	hash, err := auth.HashPassword("abcdefghijkl1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := s.CreateUser(ctx, store.User{
		Email:        "admin@example.com",
		DisplayName:  "Admin",
		PasswordHash: hash,
		Role:         store.RoleOwner,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID <= 0 {
		t.Fatal("expected user id")
	}

	count, err = s.CountUsers(ctx)
	if err != nil || count != 1 {
		t.Fatalf("count after create: %d err=%v", count, err)
	}

	sessionID, err := auth.NewSessionID()
	if err != nil {
		t.Fatalf("session id: %v", err)
	}
	expires := time.Now().UTC().Add(12 * time.Hour)
	if err := s.CreateSession(ctx, store.Session{
		ID: sessionID, UserID: user.ID, ExpiresAt: expires,
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	sess, err := s.GetSession(ctx, sessionID)
	if err != nil || sess.UserID != user.ID {
		t.Fatalf("get session: %+v err=%v", sess, err)
	}
	if err := s.DeleteSession(ctx, sessionID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
}
