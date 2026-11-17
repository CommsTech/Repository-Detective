package store

import (
	"encoding/json"
	"time"
)

// ScanGraphRecord stores a generated repository map for a scan.
type ScanGraphRecord struct {
	ScanID       string
	RepositoryID int64
	GraphJSON    json.RawMessage
	NodeCount    int
	EdgeCount    int
	GeneratedAt  time.Time
}

// AuditGraphRecord stores a repository map for a pre-install audit.
type AuditGraphRecord struct {
	AuditID     string
	GraphJSON   json.RawMessage
	NodeCount   int
	EdgeCount   int
	GeneratedAt time.Time
}
