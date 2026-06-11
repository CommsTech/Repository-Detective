# Non-product beta scan 2 — report-only

**Date:** 2026-06-10  
**Revision:** `rc-e3e19ec`

## Repo

| Field | Value |
|-------|-------|
| Repository | `commstech/ansible_playbooks` |
| Repo ID | 62 |
| Scan ID | `64684cbab7682847` |
| Mode | `report_only_dry_run: true` |
| Profile | `beta_standard` |

## Results

| Metric | Value |
|--------|-------|
| Status | completed |
| Files analyzed | 52 |
| Issues found | 78 |
| High/critical | 0 reported in summary |
| Issues created | **0** (dry-run) |
| Duration | fast completion |

## Noise estimate

78 findings on 52 files — **higher noise** for a medium ansible repo (IaC + graph heuristics likely). Expect calibration before external beta.

## Actionability

Report-only mode worked; no Gitea issues filed.

## Operator impression

Useful for internal beta; would need category filters and calibration before showing strangers.

## Recommended before marketing

- Suppress graph/IaC noise rules per repo
- Re-scan after calibration and compare issue count
