# RC active-present rescan report

**Date:** 2026-06-11  
**Fix commit:** `581d534` (dogfood scanner self-match fixes)  
**Deploy:** musl binary built in `golang:1.23-alpine`, copied via `/dev/shm` (full image rebuild blocked by disk)

## Scan

| Field | Value |
|-------|-------|
| Scan ID | `d3d6c4f279eeaf8c` (reconciliation latest: `926a5f56a26f03c9`) |
| Trigger | manual POST `/api/v1/analyze` |
| Profile | beta_standard |
| Graph | enabled |
| Duration | ~5 min |

## Before / after

| Metric | Before (e42b3e175e313904) | After rescan |
|--------|---------------------------|--------------|
| Active-present open | 21 | **0** |
| Actionable active | 4 | **0** |
| Informational active | 17 | **0** |
| High/critical active | 0 | **0** |
| Scan findings total | 21 | **0** |
| Issues created | — | **0** |
| Duplicate issues | — | **0** |

## Fix summary

1. `REL-INTERNAL-INFRA-REF` false-positive for `containers/*` registry classifiers
2. Health checks skip `analyzers/static.go` (rule engine self-match)

## Issue sync

| Item | Value |
|------|-------|
| Issue sync status | complete |
| Gitea open mapped issues | 0 |
| New issues created | 0 |

## Remaining DB state

`open_findings_total` 2457 — historical open rows without instances in latest scan (not active-present). Expected.

## Acceptance

**Product dogfood clean:** YES (0 active-present, 0 actionable)
