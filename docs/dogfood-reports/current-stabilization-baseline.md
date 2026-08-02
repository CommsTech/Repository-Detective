# Stabilization baseline — 2026-06-06

## Git

| Field | Value |
|-------|-------|
| Branch | `main` |
| Baseline commit (pre-stabilization) | `9d875fd` |
| Stabilization commits | pending push |

## CI status (Gitea Actions)

| Run | SHA | Status | Notes |
|-----|-----|--------|-------|
| **#1842** | `9d875fd` | **in_progress (stuck)** | Lint/Test/Build job on `RemoteSupport`: all steps **success**, job never completed; Security Scan + Docker Build **waiting** on `needs:` |
| #1841 | `cdce47c` | failure | Code green; artifact upload failed (fixed in 9d875fd) |
| #1840 | — | failure | Runner infra flake (checkout) |
| #1839 | `962839e` | failure | gofmt on changed files |

### CI fix applied this sprint

- Consolidated `ci.yml` to a **single job** (lint, test, build, govulncheck, docker) with `timeout-minutes: 60` and `workflow_dispatch` — removes cross-job `needs:` deadlock when runner completion lags.
- Consolidated `release.yml` to a **single job** — drops `upload-artifact@v4` / `download-artifact@v4` (unsupported on Gitea runner).

## Application health (local)

- Container: healthy on `:8081`
- Database: enabled
- `remediation_pr_enabled`: false (intentional)
- Open Gitea issues: **241**

## Batch 2 gate

**Batch 2 (P1 reliability — ignored errors in handlers/ + store/) must NOT start** until the latest CI run after these fixes is fully green.

## Product observations (pre-fix)

- Dark-mode flash between pages during load
- Setup wizard always visible in nav after configuration
- Repository map/graph rendering inconsistent
- Executive reports not business-ready; no radar on repo report
- System Health shows generic “disabled” without setup guidance
- Graph orphan logic could reference `_test.go` in disconnected package findings
- Default compose used `network_mode: host`
- Query-string `?api_key=` accepted without hardened-mode option

## Verification plan post-deploy

1. Push stabilization commits → confirm new CI run green
2. Rebuild container → rescan `commstech/Repository-Detective@main`
3. Verify graph, executive report, print/PDF, health capabilities, dark mode, setup nav
