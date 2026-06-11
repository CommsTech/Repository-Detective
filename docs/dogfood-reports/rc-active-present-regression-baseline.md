# RC active-present regression baseline

**Date:** 2026-06-11  
**Live revision:** `rc-e3e19ec`  
**Git commit:** `76bd87d`

## Before / after

| Metric | Scan `955e5f4816b9ff77` (pre-regression) | Scan `e42b3e175e313904` (regression) |
|--------|------------------------------------------|--------------------------------------|
| Scan time | 2026-06-10 12:02 UTC | 2026-06-10 23:45 UTC (post RC redeploy) |
| Files analyzed | 970 | 1005 |
| Health scanner | **clean** | **found** |
| Active-present open | **2** | **21** |
| Actionable (medium+) | **2** | **4** |
| Informational (low/info) | **0** | **17** |
| High/critical | **0** | **0** |

## Root cause

**Not an RC code regression in reconciliation logic.** A new manual product scan (`e42b3e175e313904`) ran after live redeploy with health checks reporting findings. The prior reconcilable scan (`955e5f4816b9ff77`) had health status **clean** (likely incomplete health pass or transient), leaving only 2 static `REL-INTERNAL-INFRA-REF` instances active-present.

The jump 2 → 21 is:

- **Same 2** medium static infra-ref findings (containers registry classifier self-match)
- **+19** health/maintainability findings (17 low/info + 2 medium in `analyzers/static.go`)

## Severity breakdown (21 active-present)

| Severity | Count | Actionable |
|----------|-------|------------|
| medium | 4 | yes |
| low | 7 | no |
| info | 10 | no |
| high | 0 | — |
| critical | 0 | — |

## Issue sync

| Item | Value |
|------|-------|
| Gitea open issues (mapped) | 0 |
| Issue sync status | complete |
| Issue filing enabled | true |
| Findings without linked issue | 2457 (historical open DB rows) |

## Planned remediation

1. Fix static FP for `containers/*` infra-ref classifier (scanner self-match)
2. Skip `analyzers/static.go` from health checks (rule engine self-match)
3. Product rescan with graph enabled
4. Accept remaining low/info health findings as informational dogfood backlog OR calibrate per-rule in follow-up
