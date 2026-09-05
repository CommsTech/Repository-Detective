package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestNotificationSettingsPersist(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "notif.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}

	enabled := true
	sev := "critical"
	events := "scan_failed,critical_finding"
	cooldown := 120
	settings := store.RepoSettings{
		RepositoryID:                repo.ID,
		NotificationsEnabled:        &enabled,
		NotificationMinSeverity:     &sev,
		NotificationEvents:          &events,
		NotificationCooldownSeconds: &cooldown,
	}
	if err := store.ValidateRepoSettings(settings); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRepoSettings(ctx, settings); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NotificationsEnabled == nil || !*loaded.NotificationsEnabled {
		t.Fatal("notifications_enabled not persisted")
	}
	if loaded.NotificationMinSeverity == nil || *loaded.NotificationMinSeverity != "critical" {
		t.Fatalf("severity: %v", loaded.NotificationMinSeverity)
	}
}

func TestInvalidNotificationEventsRejected(t *testing.T) {
	if err := store.ValidateNotificationEventsCSV("not_a_real_event"); err == nil {
		t.Fatal("expected invalid event error")
	}
}
