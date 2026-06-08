# Product rescan and issue sync report

**Date:** 2026-06-08

## Scan

| Field | Value |
|-------|-------|
| Scan ID | `c98c6c090fa1fbf4` |
| Profile | `maintainer_deep` |
| Dry-run | **no** |
| Issue sync | **complete** |
| Duration | ~5 min |
| Issues found (pipeline) | 1192 |

## Gitea open issues

| | Count |
|---|------|
| Before | 1 (#48 operator task) |
| After | 1 |

No duplicate issues filed during rescan (severity gate `high` blocks graph/low noise).

## DB `external_issues`

| | Count |
|---|------|
| Open before (stale) | 132 |
| Open after repair | **0** |
| Stale rows repaired | **281** |

## Active-present

| | Count |
|---|------|
| Before | 1201 |
| After | **1192** |

Dominant sources in scan `c98c6c090fa1fbf4`: `graph/low` (1044), health/reliability/tech_debt (79). Graph calibration fix committed to downgrade orphan spam in large repos — **re-scan after deploy** expected to drop active-present sharply.

## New / duplicate issues

| | Count |
|---|------|
| New issues created | 0 |
| Duplicate issues | 0 |

## Follow-up

1. Deploy commit `f64b922`+ (graph calibration + git history secrets).
2. Re-run `scripts/product-repo-resync.py --execute` after deploy.
3. Verify active-present near 0 excluding intentional operator task #48.
