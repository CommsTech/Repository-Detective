package store_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestCreateFirstOwnerClosesBootstrap(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hash, err := auth.HashPassword("abcdefghijkl1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	first, err := s.CreateFirstOwner(ctx, store.User{
		Email: "owner@example.com", DisplayName: "Owner", PasswordHash: hash, Role: store.RoleOwner, Enabled: true,
	})
	if err != nil {
		t.Fatalf("first owner: %v", err)
	}
	if first.ID <= 0 {
		t.Fatal("expected owner id")
	}

	_, err = s.CreateFirstOwner(ctx, store.User{
		Email: "second@example.com", DisplayName: "Second", PasswordHash: hash, Role: store.RoleOwner, Enabled: true,
	})
	if !errors.Is(err, store.ErrBootstrapClosed) {
		t.Fatalf("expected ErrBootstrapClosed, got %v", err)
	}
	count, err := s.CountUsers(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected exactly 1 user, got %d err=%v", count, err)
	}
}

func TestCreateFirstOwnerConcurrentOnlyOneWins(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bootstrap-race.db")
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hash, err := auth.HashPassword("abcdefghijkl1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	results := make(chan error, workers)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := s.CreateFirstOwner(ctx, store.User{
				Email:        fmt.Sprintf("owner%d@example.com", i),
				DisplayName:  "Owner",
				PasswordHash: hash,
				Role:         store.RoleOwner,
				Enabled:      true,
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok := 0
	closed := 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, store.ErrBootstrapClosed):
			closed++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly one successful CreateFirstOwner, got %d", ok)
	}
	if closed != workers-1 {
		t.Fatalf("expected %d ErrBootstrapClosed, got %d", workers-1, closed)
	}
	count, err := s.CountUsers(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 user after race, got %d err=%v", count, err)
	}
}
