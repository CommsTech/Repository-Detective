# Controlled container image scan — `alpine:3.20`

Recorded: 2026-06-10

## Scan parameters

| Field | Value |
|---|---|
| `target_type` | `registry_image` |
| Image | `alpine:3.20` |
| Tools | syft, trivy, grype |
| `generate_sbom` | true |
| `issue_creation` | false |

## Job

| Field | Value |
|---|---|
| Job ID | `rj-fa8317b9a9c7b191` |
| Scan ID | `cis-5efa1708c078c012` |
| Runner ID | `rd-runner-container-scan` |
| Runner label | `container-scan` (documented; label matching not enforced in worker yet) |
| Core Docker socket | not mounted |

## Results

| Field | Value |
|---|---|
| Image digest captured | **no** (digest not yet populated by scanner pipeline) |
| SBOM generated | **yes** (syft `ok`, cyclonedx-json on runner workspace) |
| Syft status | `ok` |
| Trivy status | `ok` |
| Grype status | `ok` |
| Vulnerabilities (stored `vuln_count`) | 0 |
| Highest severity (stored) | n/a (coarse parser; Grype reports low-severity matches on host spot-check) |
| Issue creation count | 0 |
| PR creation count | 0 |
| Logs redaction | clean (no tokens in runner log) |
| UI route `/ui/repos/1/containers` | **200** (with API key) |

## Notes

- First attempt (`rj-f18c3f7592f5133c`) completed on runner but result POST failed when worker process exited early; job cancelled and re-queued.
- Fixed `EnqueueScan` validation: runner delegation must not pass `onCore=true` when enqueueing runner jobs.
- Added implicit Docker Hub allowlist matching for bare refs like `alpine:3.20` when `docker.io` is allowlisted.
- Server required `container_image_scan` in `runner_allowed_job_types` (`.env` override for test window).

## Rollback (Phase 4)

| Step | Status |
|---|---|
| Runner stopped | yes (`rd-runner-container-scan` terminated) |
| `runner_delegation_enabled` | false |
| `container_scanning_enabled` | false |
| `container_scan_create_issues` | false |
| Container recreated without test env overrides | yes |
| `/health` | healthy, version `9e10a40` |
| Worker running | no |
