package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store is the persistence interface. SQLite is implemented today; PostgreSQL later.
type Store interface {
	Close() error

	UpsertRepository(ctx context.Context, repo Repository) (Repository, error)
	GetRepositoryByFullName(ctx context.Context, forgeType, fullName string) (Repository, error)
	ListRepositories(ctx context.Context) ([]Repository, error)

	GetRepoSettings(ctx context.Context, repositoryID int64) (RepoSettings, error)
	SaveRepoSettings(ctx context.Context, settings RepoSettings) error

	CreateScan(ctx context.Context, scan Scan) (Scan, error)
	FinishScan(ctx context.Context, scanID string, result ScanResult) error
	GetScan(ctx context.Context, scanID string) (Scan, error)

	AddScannerResults(ctx context.Context, results []ScannerResultRecord) error

	UpsertFinding(ctx context.Context, finding Finding) (Finding, error)
	AddFindingInstance(ctx context.Context, instance FindingInstance) error
	GetFindingByFingerprint(ctx context.Context, repositoryID int64, fingerprint string) (Finding, error)

	UpsertExternalIssue(ctx context.Context, issue ExternalIssue) (ExternalIssue, error)

	AddLifecycleEvent(ctx context.Context, event LifecycleEvent) error

	SaveScanGraph(ctx context.Context, record ScanGraphRecord) error
	GetScanGraph(ctx context.Context, scanID string) (ScanGraphRecord, error)
	GetLatestScanGraphForRepo(ctx context.Context, repositoryID int64) (ScanGraphRecord, error)
	SaveAuditGraph(ctx context.Context, record AuditGraphRecord) error
	GetAuditGraph(ctx context.Context, auditID string) (AuditGraphRecord, error)

	CreateRunnerJob(ctx context.Context, job RunnerJob) (RunnerJob, error)
	GetRunnerJob(ctx context.Context, jobID string) (RunnerJob, error)
	GetRunnerJobByScanID(ctx context.Context, scanID string) (RunnerJob, error)
	ClaimNextRunnerJob(ctx context.Context, now time.Time) (RunnerJob, error)
	UpdateRunnerJob(ctx context.Context, job RunnerJob) error
	CancelRunnerJob(ctx context.Context, jobID string) error
	ListRunnerJobs(ctx context.Context, opts ListOptions) ([]RunnerJob, error)
	ListRunnerJobsByRepository(ctx context.Context, repositoryID int64, opts ListOptions) ([]RunnerJob, error)
	CountRunningRunnerJobs(ctx context.Context) (int, error)
	ExpireStaleRunnerJobs(ctx context.Context, now time.Time) (int64, error)
	SaveRunnerArtifact(ctx context.Context, artifact RunnerArtifact) error
	TryRecordRunnerNonce(ctx context.Context, nonce string) (bool, error)
	CountRunnerJobsByStatus(ctx context.Context) (map[string]int, error)
}

// Config selects a database backend.
type Config struct {
	Enabled bool
	Driver  string
	Path    string
	DSN     string
}

// Open creates a QueryStore from configuration.
func Open(cfg Config) (QueryStore, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driver == "" {
		driver = "sqlite"
	}

	switch driver {
	case "sqlite":
		path := strings.TrimSpace(cfg.Path)
		if path == "" {
			path = "./data/repository-detective.db"
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		return OpenSQLite(path)
	default:
		return nil, fmt.Errorf("unsupported database driver %q (sqlite only in Phase 5)", driver)
	}
}
