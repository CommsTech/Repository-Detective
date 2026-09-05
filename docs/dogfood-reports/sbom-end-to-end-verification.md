# SBOM end-to-end verification

**Date:** 2026-06-02

## Implementation verified

| Capability | Status |
|------------|--------|
| Store `sbom_artifacts` | yes |
| Runner syft generation (container) | yes (`alpine:3.20` demo) |
| UI `/ui/scans/:scan_id/sbom` | added RC sprint |
| UI `/ui/repos/:id/sbom` | added RC sprint |
| Download `/ui/scans/:scan_id/sbom/download` | added RC sprint |
| Clear failure when missing | yes (`SBOM not available` panel) |

## Formats

- CycloneDX when syft produces `.cdx.json`
- Internal normalized JSON in store metadata
- SPDX when tool output available

## Next actions

1. Run repo scan with SBOM job type and confirm UI panel populates
2. Re-run container scan with `container_scan_generate_sbom: true` and verify download
