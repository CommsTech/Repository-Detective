package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/store"
)

func openTestStore(t *testing.T) store.QueryStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: path})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSQLiteInitCreatesSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.db")

	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file not created: %v", err)
	}

	sqlite, ok := s.(*store.SQLiteStore)
	if !ok {
		t.Fatalf("expected *SQLiteStore")
	}
	_ = sqlite
}

func TestMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrate.db")

	for i := 0; i < 3; i++ {
		s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: path})
		if err != nil {
			t.Fatalf("open attempt %d: %v", i+1, err)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("close attempt %d: %v", i+1, err)
		}
	}
}

func TestRepositoryUpsert(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner:         "acme",
		Name:          "demo",
		FullName:      "acme/demo",
		CloneURL:      "https://git.example/acme/demo.git",
		ConnectedRepo: true,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if repo.ID == 0 {
		t.Fatal("expected repository id")
	}

	repo2, err := s.UpsertRepository(ctx, store.Repository{
		Owner:    "acme",
		Name:     "demo",
		FullName: "acme/demo",
		CloneURL: "https://git.example/acme/demo.git",
	})
	if err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	if repo2.ID != repo.ID {
		t.Fatalf("expected same id, got %d vs %d", repo2.ID, repo.ID)
	}

	list, err := s.ListRepositories(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list repositories: len=%d err=%v", len(list), err)
	}
}

func TestRepoSettingsSaveGet(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	enabled := true
	policy := "gate_pr"
	depth := 2
	if err := s.SaveRepoSettings(ctx, store.RepoSettings{
		RepositoryID:  repo.ID,
		Enabled:       &enabled,
		PolicyLevel:   &policy,
		AnalysisDepth: &depth,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	got, err := s.GetRepoSettings(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if got.Enabled == nil || !*got.Enabled {
		t.Fatal("expected enabled=true")
	}
	if got.PolicyLevel == nil || *got.PolicyLevel != "gate_pr" {
		t.Fatalf("unexpected policy: %#v", got.PolicyLevel)
	}

	global := store.DefaultGlobalSettings()
	effective := store.ResolveRepoSettings(global, got)
	if !effective.Enabled || effective.PolicyLevel != "gate_pr" || effective.AnalysisDepth != 2 {
		t.Fatalf("unexpected effective settings: %+v", effective)
	}
}

func TestScanCreateFinish(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "scan1234567890ab"

	created, err := s.CreateScan(ctx, store.Scan{
		ID:           scanID,
		RepositoryID: repo.ID,
		TriggerType:  store.TriggerPush,
		Ref:          "main",
		Status:       store.ScanStatusStarted,
	})
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	if created.Status != store.ScanStatusStarted {
		t.Fatalf("unexpected status: %s", created.Status)
	}

	summary, _ := json.Marshal(map[string]any{"issues_found": 3})
	if err := s.FinishScan(ctx, scanID, store.ScanResult{
		Status:      store.ScanStatusCompleted,
		FinishedAt:  time.Now().UTC(),
		SummaryJSON: summary,
	}); err != nil {
		t.Fatalf("finish scan: %v", err)
	}

	got, err := s.GetScan(ctx, scanID)
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}
	if got.Status != store.ScanStatusCompleted || got.FinishedAt == nil {
		t.Fatalf("unexpected finished scan: %+v", got)
	}
}

func TestScannerResultPersistence(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "scan-scanner-test"

	_, err := s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual})
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}

	err = s.AddScannerResults(ctx, []store.ScannerResultRecord{
		{ScanID: scanID, ScannerName: "trivy", Status: "found", FindingsCount: 2, Detail: "ok"},
		{ScanID: scanID, ScannerName: "semgrep", Status: "clean", FindingsCount: 0},
	})
	if err != nil {
		t.Fatalf("add scanner results: %v", err)
	}
}

func TestFindingUpsertAndInstance(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	scanID := "scan-findings"

	_, err := s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerPR})
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}

	now := time.Now().UTC()
	f1, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID:    repo.ID,
		Fingerprint:     "rd-deadbeef",
		Category:        "security",
		Severity:        "high",
		Confidence:      0.99,
		Source:          "semgrep",
		Title:           "Test finding",
		FirstSeenScanID: scanID,
		LastSeenScanID:  scanID,
		FirstSeenAt:     now,
		LastSeenAt:      now,
	})
	if err != nil {
		t.Fatalf("upsert finding: %v", err)
	}

	f2, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID:   repo.ID,
		Fingerprint:    "rd-deadbeef",
		Category:       "security",
		Severity:       "high",
		Confidence:     0.99,
		Source:         "semgrep",
		Title:          "Test finding updated",
		LastSeenScanID: scanID,
		LastSeenAt:     now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("upsert finding again: %v", err)
	}
	if f2.ID != f1.ID {
		t.Fatalf("expected same finding id")
	}
	if !f2.FirstSeenAt.Equal(f1.FirstSeenAt) {
		t.Fatal("first_seen_at should be preserved")
	}

	if err := s.AddFindingInstance(ctx, store.FindingInstance{
		FindingID:        f2.ID,
		ScanID:           scanID,
		EvidenceRedacted: "evidence",
		LocationJSON:     json.RawMessage(`{"file":"main.go","line":10}`),
	}); err != nil {
		t.Fatalf("add instance: %v", err)
	}
}

func TestNoDuplicateFindingsForSameFingerprint(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		_, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID,
			Fingerprint:  "rd-same",
			Title:        "same",
			LastSeenAt:   now,
			FirstSeenAt:  now,
		})
		if err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}

	got, err := s.GetFindingByFingerprint(ctx, repo.ID, "rd-same")
	if err != nil {
		t.Fatalf("get finding: %v", err)
	}
	if got.ID == 0 {
		t.Fatal("expected finding")
	}
}

func TestExternalIssueMapping(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID,
		Fingerprint:  "rd-issue",
		Title:        "issue",
		FirstSeenAt:  time.Now().UTC(),
		LastSeenAt:   time.Now().UTC(),
	})

	issue, err := s.UpsertExternalIssue(ctx, store.ExternalIssue{
		FindingID:   finding.ID,
		IssueNumber: 42,
		IssueURL:    "https://git.example/o/r/issues/42",
		State:       "open",
	})
	if err != nil {
		t.Fatalf("upsert external issue: %v", err)
	}
	if issue.ID == 0 {
		t.Fatal("expected external issue id")
	}
}

func TestLifecycleEventInsert(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID,
		Fingerprint:  "rd-life",
		Title:        "life",
		FirstSeenAt:  time.Now().UTC(),
		LastSeenAt:   time.Now().UTC(),
	})
	findingID := finding.ID

	if err := s.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: &findingID,
		ScanID:    "scan1",
		EventType: "issue_created",
		Message:   "created issue",
	}); err != nil {
		t.Fatalf("add lifecycle event: %v", err)
	}
}

func TestDatabaseDisabledReturnsNilStore(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: false})
	if err != nil {
		t.Fatalf("open disabled: %v", err)
	}
	if s != nil {
		t.Fatal("expected nil store when disabled")
	}
}

func TestResolveRepoSettingsInheritsGlobal(t *testing.T) {
	global := store.DefaultGlobalSettings()
	repoSettings := store.RepoSettings{}
	effective := store.ResolveRepoSettings(global, repoSettings)
	if effective.WorkspaceMode != global.WorkspaceMode {
		t.Fatalf("expected inherit workspace mode, got %s", effective.WorkspaceMode)
	}

	override := "archive"
	repoSettings.WorkspaceMode = &override
	effective = store.ResolveRepoSettings(global, repoSettings)
	if effective.WorkspaceMode != "archive" {
		t.Fatalf("expected override, got %s", effective.WorkspaceMode)
	}
}

func TestRecorderNoOpWhenDisabled(t *testing.T) {
	rec := store.NewRecorder(nil, nil)
	if rec.Enabled() {
		t.Fatal("recorder should be disabled")
	}
	ctx := context.Background()
	if _, err := rec.BeginScan(ctx, store.ScanContext{Owner: "o", Repo: "r", ScanID: "x"}); err != nil {
		t.Fatalf("begin scan: %v", err)
	}
	if err := rec.FinishScan(ctx, "x", nil, nil); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
}

func TestRecorderPersistsIssues(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	rec := store.NewRecorder(s, nil)

	repo, err := rec.BeginScan(ctx, store.ScanContext{
		Owner:       "acme",
		Repo:        "demo",
		ScanID:      "scanrec001",
		TriggerType: store.TriggerManual,
	})
	if err != nil {
		t.Fatalf("begin scan: %v", err)
	}

	issuesList := []ai.CodeIssue{{
		Fingerprint: "rd-rec001",
		Category:    "security",
		Severity:    "high",
		Title:       "Recorder test",
		File:        "main.go",
		LineNumber:  1,
	}}
	processed := []issues.ProcessedIssueRecord{{
		Fingerprint: "rd-rec001",
		IssueNumber: 7,
		IssueURL:    "https://git.example/acme/demo/issues/7",
		Action:      "created",
	}}

	if err := rec.RecordIssues(ctx, repo.ID, "scanrec001", "gitea", issuesList, processed); err != nil {
		t.Fatalf("record issues: %v", err)
	}

	got, err := s.GetFindingByFingerprint(ctx, repo.ID, "rd-rec001")
	if err != nil {
		t.Fatalf("get finding: %v", err)
	}
	if got.Title != "Recorder test" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}

func TestGetFindingByFingerprintNotFound(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})

	_, err := s.GetFindingByFingerprint(ctx, repo.ID, "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}
