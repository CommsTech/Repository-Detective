# Non-product dry-run repo selection

Generated: 2026-06-07  
Selection: exactly **2** non-Repository-Detective repos (not fleet-wide)

## Selected repos

### 1. Small — `commstech/nextcloud_scripts`

| Attribute | Value |
|-----------|-------|
| Size | ~24 KB |
| Language | Shell |
| Open issues (pre-scan) | 0 |
| File count (scan) | 1 |
| Risk level | **Low** — homelab shell utility, no production secrets expected |
| Reason selected | Minimal surface for fast scanner sanity check |
| Expected scanners | trivy, gitleaks, semgrep, shellcheck (if present), graph, health, static |

### 2. Medium — `commstech/netmapper`

| Attribute | Value |
|-----------|-------|
| Size | ~677 KB |
| Language | Python |
| Open issues (pre-scan) | 1 (pre-existing, untouched) |
| File count (scan) | 65 |
| Risk level | **Low–medium** — homelab network tool, not mission-critical |
| Reason selected | Representative Python repo with graph + static findings |
| Expected scanners | static, health, graph, test_gap, trivy, gitleaks, semgrep, ruff (if present) |

## Excluded repos

| Repo | Reason |
|------|--------|
| `commstech/Repository-Detective` | Product repo — already closed out |
| `commstech/paperless_thunder` | 17 open issues — risk of noise/confusion |
| `Infrastructure_as_Code`, `netmon`, `nettech` | Fleet-scale / high blast radius |

## Issue filing confirmation

- Per-scan `report_only_dry_run: true` on both scans
- Effective settings: `issue_policy=off`, `policy_level=monitor_only`
- Global `auto_create_issues` left unchanged; dry-run override is per-request only
- **No Gitea issues created or modified in non-product repos**

## Rollback / stop conditions

Stop immediately if:

- Open issue count increases on target repo
- Any PR is opened
- Duplicate issue burst detected
- DB growth > 500 MB in session

**Observed:** none triggered.

## Scan profile

`standard_deterministic` on `main` branch via `/api/v1/analyze`.
