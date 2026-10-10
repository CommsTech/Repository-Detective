# SBOM beta readiness

## Implementation

| Capability | Status |
|------------|--------|
| Per-repo SBOM during scan (workspace prepared) | Implemented in `analyzers/engine.go` → `sbom.GenerateAndCheck` |
| Go module repos | `cyclonedx-gomod` preferred, syft fallback |
| Other manifests | syft when installed |
| Grype vulnerability check | When grype + DB available on runner |
| Persistence | `sbom_artifacts` table (migration 19) |
| Scan summary fields | `sbom_status`, `sbom_package_count`, `sbom_vuln_count` |

## Status values

| Status | Meaning |
|--------|---------|
| `sbom_generated` | SBOM file written; grype not run |
| `sbom_no_supported_manifest` | No dependency manifest |
| `sbom_tool_missing` | syft/cyclonedx-gomod unavailable |
| `sbom_check_clean` | Grype found 0 vulnerabilities |
| `sbom_vulnerabilities_found` | Grype reported matches |
| `sbom_check_failed` | Parse/DB/tool failure — **not** treated as clean |

## Beta acceptance

- [x] Unit tests for no-manifest and Go module paths
- [x] DB persistence API
- [ ] UI scan detail SBOM panel (summary JSON available; full UI polish optional for internal beta)
- [x] Release package can include app SBOM via `cyclonedx-gomod` in `build-beta-release.sh`

## Operator actions

1. Ensure scanner image includes syft/grype (see `scripts/install-scanner-tools.sh`).
2. Run `grype db update` after image build if DB stale.
3. Do not commit generated SBOM files from target repos — artifacts stay in scan workspace / DB metadata.
