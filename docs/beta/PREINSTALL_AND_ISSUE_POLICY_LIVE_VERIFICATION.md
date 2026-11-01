# Pre-install and issue policy live verification

Date: 2026-06-08

## Deploy

| Item | Value |
|------|-------|
| Prior revision | `ddb79d6` |
| Sprint commits | pending push |
| Config change | `preinstall_audit_enabled: true` in `config/config.yaml` |

## Pre-install

| Check | Expected after redeploy |
|-------|-------------------------|
| `/ui/preinstall` | Enabled workflow (not disabled banner) |
| System Health | Pre-install audit enabled |
| Private IP | Blocked by default |
| Issue/PR creation | 0 |

## Issue policy

| Check | Result |
|-------|--------|
| Product Gitea open issues | 1 (#48 ops) after closeout |
| Dry-run product scan | `47993b1eecb63e47` started |
| Benchmark fixture issues | #347–351 closed with evidence |
| Branding | Repository Detective in UI title |

## Follow-up after redeploy

1. Confirm `/health` reports `preinstall_audit_enabled: true`
2. Run product repo rescan to drop active-present fixture findings
3. Sync stale `external_issues` rows (132 → match Gitea)
