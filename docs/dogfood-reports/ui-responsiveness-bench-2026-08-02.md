# UI responsiveness benchmark — 2026-08-02

Base: `http://127.0.0.1:8081` (operator DB ~685MB; warm process)

## Summary

| Route | Baseline p50 | After cold p50 | After warm p50 | Δ cold |
|-------|-------------:|---------------:|---------------:|-------:|
| `/ui` | 1245 ms | 517 ms | 247 ms | **−58%** |
| `/ui/repos` | 1084 ms | 293 ms | 288 ms | **−73%** |
| `/ui/health` | 1022 ms | 341 ms | 78 ms | **−67%** |
| `/ui/reports` | 1004 ms | 383 ms | 90 ms | **−62%** |
| `/api/v1/dashboard/summary` | 915 ms | 291 ms | 2 ms | **−68%** |
| `/ui/findings` | 180 ms | — | 185 ms | ~flat |
| `/ui/scans` | 89 ms | — | 84 ms | ~flat |

Verdict: **retain all changes** — every previously slow page improved; already-fast pages stayed flat.

## What changed

1. **Schema migration 24** — indexes on `finding_instances(created_at)`, `findings(status*)`, `scans(status/trigger, started_at)`, `scanner_results(status*)`, `external_issues(state, finding_id)`.
2. **Repo control queries** — scoped / scan-id–based unmapped + active-present counts (removed full-table correlated scan rollups).
3. **Dashboard charts** — one batched category query instead of N+1 per top repo; dashboard loads 20 repo summaries (not 200).
4. **Dashboard summary** — 2s in-process TTL cache shared by `/ui`, `/ui/health`, `/ui/reports`, and the API.
5. **Scanner platform rollups** — windowed to last 30 days (aligned with actionable health, cheaper aggregate).

## Baseline (before)

| Route | p50 | p90 | mean | min | max |
|-------|-----|-----|------|-----|-----|
| `/ui` | 1245 | 1267 | 1236 | 1190 | 1267 |
| `/ui/repos` | 1084 | 1103 | 1071 | 1024 | 1103 |
| `/ui/health` | 1022 | 1029 | 1011 | 984 | 1029 |
| `/ui/reports` | 1004 | 1053 | 1013 | 989 | 1053 |
| `/ui/findings` | 180 | 189 | 179 | 172 | 189 |
| `/ui/scans` | 89 | 94 | 87 | 79 | 94 |
| `/ui/learning` | 82 | 83 | 81 | 78 | 83 |
| `/ui/configure` | 84 | 104 | 88 | 79 | 104 |
| `/ui/projects` | 83 | 110 | 88 | 78 | 110 |
| `/ui/preinstall` | 56 | 131 | 74 | 41 | 131 |
| `/api/v1/dashboard/summary` | 915 | 989 | 923 | 884 | 989 |
| `/api/v1/repos` | 46 | 53 | 47 | 41 | 53 |

## After (cold-ish, 2.2s gap → cache expired)

| Route | p50 | mean | min | max |
|-------|-----|------|-----|-----|
| `/ui` | 517 | 1608* | 478 | 5988* |
| `/ui/repos` | 293 | 293 | 280 | 304 |
| `/ui/health` | 341 | 296 | 73 | 375 |
| `/ui/reports` | 383 | 382 | 363 | 405 |
| `/api/v1/dashboard/summary` | 291 | 289 | 266 | 310 |

\*One post-restart outlier on `/ui` (~6s); steady-state cold samples sit ~480–620 ms.

## After (warm burst)

| Route | p50 | mean | min | max |
|-------|-----|------|-----|-----|
| `/ui` | 247 | 251 | — | — |
| `/ui/repos` | 288 | 288 | — | — |
| `/ui/health` | 78 | 128 | — | — |
| `/ui/reports` | 90 | 143 | — | — |
| `/api/v1/dashboard/summary` | 2 | 55 | — | — |

Reproduce: `source .env && ./scripts/ui-responsiveness-bench.sh label`
