package store

import (
	"encoding/json"
	"time"
)

const (
	RunnerJobTypeScanFullRepo      = "scan_full_repo"
	RunnerJobTypeContainerImageScan = "container_image_scan"

	RunnerJobStatusQueued     = "queued"
	RunnerJobStatusDispatched = "dispatched"
	RunnerJobStatusRunning    = "running"
	RunnerJobStatusCompleted  = "completed"
	RunnerJobStatusFailed     = "failed"
	RunnerJobStatusExpired    = "expired"
	RunnerJobStatusCancelled  = "cancelled"

	RunnerArtifactResultJSON   = "result_json"
	RunnerArtifactScannerLogs  = "scanner_logs"
	RunnerArtifactGraphJSON    = "graph_json"
)

// RunnerJob is a delegated scan job for an external runner.
type RunnerJob struct {
	ID                int64
	JobID             string
	RepositoryID      int64
	ScanID            string
	JobType           string
	Status            string
	RunnerMode        string
	Ref               string
	CommitSHA         string
	PRNumber          int
	PolicySnapshotJSON json.RawMessage
	JobSpecJSON       json.RawMessage
	ResultSummaryJSON json.RawMessage
	Error             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	ExpiresAt         *time.Time
}

// RunnerArtifact stores inline runner output when small enough.
type RunnerArtifact struct {
	ID           int64
	JobID        string
	ArtifactType string
	BodyJSON     json.RawMessage
	SizeBytes    int64
	SHA256       string
	CreatedAt    time.Time
}
