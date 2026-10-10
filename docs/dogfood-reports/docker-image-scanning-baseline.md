# Docker image scanning baseline

Recorded: 2026-06-02  
Latest commit: `f06bfd5`

## Product dogfood

| Metric | Value |
|---|---:|
| Active-present | 0 |
| Actionable active (medium+) | 0 |
| High/critical | 0 |
| Latest scan | `95a5551881e866d4` |
| Open Gitea issues | 1 (#48 operator task) |

## Current scanner support (repo/workspace)

| Tool | Repo FS scan | OCI image scan | Live all-in-one |
|---|---|---|
| Trivy | `trivy fs` | **not implemented** | missing from PATH |
| Grype | `grype dir:` | **not implemented** | missing from PATH |
| Syft | SBOM job only (`syft dir`) | **not implemented** | missing from PATH |
| Hadolint | Dockerfile lint | n/a | missing |
| Docker CLI | n/a | n/a | not in core container |

Live `/health` tools_summary: **4/10** available (govulncheck, gosec, staticcheck, linters).

## Current runner support

| Capability | Status |
|---|---|
| Job types | `scan`, `sbom`, `graph`, `preinstall_audit`, `remediation_verify` |
| Native worker | implemented, HMAC auth |
| Runner delegation | **disabled** by default |
| Docker socket on core | **not mounted** |
| Docker socket on runner | optional (not required for repo scans) |
| Container image scan job | **not implemented** (this sprint adds `container_image_scan`) |

## Current image scanning support

- **None** for OCI/registry/local daemon images.
- Dockerfile/compose/K8s image references are not extracted as scan targets.
- Trivy/Grype findings today are filesystem/manifest oriented only.

## Security posture

| Control | Value |
|---|---|
| Core Docker socket | not mounted |
| Runner Docker socket | opt-in on runner host only |
| Pre-install audit | report-only |
| Remediation PR | disabled |
| Gitea Actions backend | disabled |
| All-repo scan | off |

## Marketing-readiness blockers

1. Gitea wiki HTTP 500 — not populated
2. First-run docs/screenshots need polish
3. Pre-install audit consistency on public repos unverified at scale
4. **Docker image scanning not implemented** (this sprint)
5. Scanner availability in all-in-one incomplete — must explain gaps in UI
6. Calibration/learning UI needs clearer operator explanations
7. Non-product report-only beta scans not yet run (2+ needed)
8. Product dogfood must stay at 0 active-present / 0 high-critical
