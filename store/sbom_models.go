package store

import "time"

// SBOMArtifact records SBOM generation and vulnerability check for a scan.
type SBOMArtifact struct {
	ID            int64     `json:"id"`
	RepositoryID  int64     `json:"repository_id"`
	ScanID        string    `json:"scan_id"`
	Format        string    `json:"format"`
	PackageCount  int       `json:"package_count"`
	VulnCount     int       `json:"vuln_count"`
	Status        string    `json:"status"`
	Detail        string    `json:"detail"`
	ArtifactPath  string    `json:"artifact_path"`
	CreatedAt     time.Time `json:"created_at"`
}
