# Post-runner product rescan report

Recorded: 2026-06-09  
Rescan commit deployed: `8d5da54`

## Scan

| Field | Value |
|-------|-------|
| Scan ID | `4f8617f80f1ef1e8` |
| Trigger | manual |
| Analysis depth | 2 |
| Graph enabled | yes |
| Files analyzed | 929 |
| Findings in scan | 79 |
| Graph state | **available** (3830 nodes) |
| Persistence | complete |
| Issue sync | complete |
| Reconciliation | complete |

## Active-present delta

| Metric | Before (`f6102e4fed8e2b37`) | After (`4f8617f80f1ef1e8`) |
|--------|----------------------------:|---------------------------:|
| Active-present | 89 | **79** |
| High/critical | 0 | **0** |

Reduction: **10** findings (health/static calibration batch).

## Top remaining rules (after)

| Rule | Count |
|------|------:|
| HEALTH-IGNORED-ERROR | 33 |
| HEALTH-MANY-PARAMS | 13 |
| HEALTH-READ-ALL | 8 |
| HEALTH-DEPRECATED | 3 |
| HEALTH-TECH-PHRASE | 3 |
| Other maintainability | 19 |

## Gitea issues

| Check | Result |
|-------|--------|
| Open issues before | 1 (#48) |
| Open issues after | 1 (#48) |
| Duplicate issues created | **0** |
| external_issues sync | complete |

## Notes

- Prior scan `2463e276e8a2b979` analyzed 0 files (invalid); superseded by this rescan.
- Remaining HEALTH-IGNORED-ERROR items are low/info production paths (store, runner, main) — next batch can address or allow-list with evidence.
- REL-INTERNAL-INFRA-REF: 1 medium in `preinstall/audit_failure.go` (blocked-host catalog — candidate for false-positive rule).
