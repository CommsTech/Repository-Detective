# Product clean rescan and issue sync report

Recorded: 2026-06-09

## Scans triggered

| Scan ID | Trigger | Profile / depth | Status |
|---------|---------|-----------------|--------|
| `7e1a7c522a36fb89` | manual curl | analysis_depth=2, enable_code_graph=true | completed |
| `f6102e4fed8e2b37` | product-repo-resync.py --execute | maintainer_deep | completed |

Latest reconcilable scan: **`f6102e4fed8e2b37`**

## Graph (latest scan)

| Field | Value |
|-------|-------|
| state | available |
| nodes | 3723 |
| edges | 6140 |
| truncated | no |

## Reconciliation

| Metric | Before | After |
|--------|--------|-------|
| Active-present | 89 | 89 |
| Gitea open issues | 1 | 1 |
| DB external_issues stale rows repaired | — | 1 |
| New issues created | — | 0 |
| Duplicate issues created | — | 0 |

## Issue sync

- `external_issues` reconciled against live Gitea open set (1 open: #48).
- One stale DB row marked closed.
- Issue filing remains report-only / monitor policy — no new Gitea issues from rescan.

## Notes

Active-present unchanged at 89 — all low/medium/info health/static findings (see triage report). No high/critical active-present findings after rescan.
