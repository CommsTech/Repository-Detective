# SBOM live verification

**Date:** 2026-06-10  
**Revision:** `rc-e3e19ec`

## Routes tested

| Route | HTTP | Result |
|-------|------|--------|
| `/ui/repos/1/sbom` | 200 | SBOM summary panel; honest detail when tools missing |
| `/ui/scans/1a4fc7a409f6d376/sbom` | 200 | SBOM summary for beta scan |
| `/ui/scans/1a4fc7a409f6d376/sbom/download` | 404 | No artifact on disk for this scan (expected) |

## Repo SBOM (commstech/Repository-Detective)

- Page renders **SBOM summary** with format/status/package counts when artifact exists.
- Detail text: `cyclonedx-gomod and syft unavailable for Go SBOM` — **does not claim success** when generation failed/unavailable.

## Container SBOM (prior demo)

- Container scan `alpine:3.20` proven in earlier sprint (`rj-fa8317b9a9c7b191`); not re-run this pass.
- UI panel exists on container result pages; live re-check deferred to next container opt-in window.

## Acceptance

| Item | Status |
|------|--------|
| Repo SBOM page live | **PASS** |
| Scan SBOM page live | **PASS** |
| Download when artifact exists | route exists; 404 when missing (honest) |
| Failure not masked as success | **PASS** |

## Remaining

- Prove download with a scan that persisted `artifact_path` on disk.
- Re-verify container image SBOM panel after next scoped container scan.
