# Non-product dry-run baseline

Generated: 2026-06-07  
Mission: controlled report-only dry run (1 small + 1 medium non-Repository-Detective repo)

## Product repo state (Phase 0)

| Metric | Value |
|--------|-------|
| Latest commit | `87800af` (after dry-run policy fixes; baseline closeout was `586d4b4`) |
| Latest product scan ID | `5e570c95bc4e3467` |
| Gitea open issues | **1** (#48 — homelab Qdrant/AI operator task) |
| Active-present findings | **0** |
| New issues on latest scan | **0** |
| `.env` staged | **no** |
| `repository-detective-build` ELF staged | **no** |

## Backlog-control state

| Setting | Value |
|---------|-------|
| `dogfood_backlog_control_enabled` | `true` |
| `dogfood_backlog_update_existing_only` | `true` |
| `dogfood_backlog_max_open_issues` | 100 |
| Allow new issue severity | high, critical only |

## Issue filing policy (global vs dry-run)

| Layer | Policy |
|-------|--------|
| Global config | `auto_create_issues: true` (unchanged) |
| Dry-run per-scan | `report_only_dry_run: true` via `/api/v1/analyze` |
| Effective at scan time | `issue_policy=off`, `policy_level=monitor_only` |
| `max_issue_creation` | **0** (forge filing skipped) |
| AI / remediation | disabled / off |

## Dry-run mode

- **Mode:** analyze + persist findings + generate scan summary — **report-only**
- **Issue filing:** disabled per request flag
- **PR / auto-remediation:** disabled
- **Fleet-wide settings:** not modified
- **All-repo scan:** not started

## Code changes for report-only guard

| Commit | Description |
|--------|-------------|
| `7f59470` | `ApplyReportOnlyDryRunSettings`, API flag, helper script |
| `87800af` | Fix `runAnalysis` context propagation (report-only flag was dropped) |

## Acceptance (Phase 0)

- [x] Product repo clean (0 active-present, 1 intentional open issue)
- [x] Backlog-control active
- [x] Report-only guard deployed to `repository-detective` container
- [x] No secrets or ELF in git staging
