# Container image scan verification

Recorded: 2026-06-02  
Latest commit: (this sprint)

## Scope

Controlled verification only — no all-server image inventory.

## Tests performed

| Step | Result |
|---|---|
| Discover images from product repo (Dockerfile) | ✅ unit tests + API handler |
| `container_image_scan` job type registered | ✅ |
| Runner skips git clone for container jobs | ✅ |
| Core Docker socket not mounted | ✅ verified live |
| Disabled config rejects enqueue | ✅ `ErrScanningDisabled` |
| Runner-required policy on core | ✅ `ErrRunnerRequired` |
| Registry allowlist enforcement | ✅ unit test |
| Logs redact credential patterns | ✅ unit test |
| Credentials not in job payload | ✅ payload has image ref only |
| Issue creation default off | ✅ `container_scan_create_issues: false` |
| UI Container Images panel | ✅ `/ui/repos/1/containers` |

## Controlled image scan (live)

| Field | Value |
|---|---|
| Target (planned) | `alpine:3.20` |
| Executed live | **deferred** — requires native runner with Trivy/Grype/Syft and `container_scanning_enabled=true` |
| Vulnerabilities found | n/a (unit-test parsers only) |
| SBOM generated | n/a |
| Issues created | 0 (policy default) |

## Notes

- Live all-in-one has 4/10 scanners; Trivy/Grype/Syft missing from core container (by design).
- Image scans must run on runner host with optional Docker socket.
- Product active-present remains **0** after this sprint.

## Next operator steps

1. Label native runner: `container-scan`
2. Install Trivy/Grype/Syft on runner host
3. Set `container_scanning_enabled=true`, allow `docker.io` or `alpine`
4. Scan `alpine:3.20` via API or UI
5. Verify results in Container Images panel
