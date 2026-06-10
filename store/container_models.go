package store

import (
	"encoding/json"
	"time"
)

// Container image scan status values.
const (
	ContainerScanStatusQueued    = "queued"
	ContainerScanStatusRunning   = "running"
	ContainerScanStatusCompleted = "completed"
	ContainerScanStatusFailed    = "failed"
)

// ContainerImageReference is a persisted discovered image reference.
type ContainerImageReference struct {
	ID              int64           `json:"id"`
	RepositoryID    int64           `json:"repository_id"`
	Image           string          `json:"image"`
	Tag             string          `json:"tag"`
	Digest          string          `json:"digest"`
	TargetType      string          `json:"target_type"`
	FilePath        string          `json:"file_path"`
	Line            int               `json:"line"`
	ServiceName     string          `json:"service_name"`
	MutableTag      bool              `json:"mutable_tag"`
	PrivateRegistry bool              `json:"private_registry"`
	LastScanID      string            `json:"last_scan_id"`
	LastDigest      string            `json:"last_digest"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	MetaJSON        json.RawMessage   `json:"meta_json,omitempty"`
}

// ContainerImageScan is one image scan execution record.
type ContainerImageScan struct {
	ID             int64           `json:"id"`
	RepositoryID   int64           `json:"repository_id"`
	ScanID         string          `json:"scan_id"`
	RunnerJobID    string          `json:"runner_job_id"`
	Image          string          `json:"image"`
	ImageDigest    string          `json:"image_digest"`
	Status         string          `json:"status"`
	VulnCount      int             `json:"vuln_count"`
	SBOMPath       string          `json:"sbom_path"`
	SBOMFormat     string          `json:"sbom_format"`
	CoverageJSON   json.RawMessage `json:"coverage_json"`
	WarningsJSON   json.RawMessage `json:"warnings_json"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
