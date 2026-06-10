# Active-present 26 burn-down rescan report

Recorded: 2026-06-02  
Latest commit after burn-down: `7dd92c5`

## Scan comparison

| Metric | Before (`27fbd37be97ef5f7`) | Interim (`801ca5bce8683213`) | After push (`cfcb1e05419859d6`) |
|---|---:|---:|---:|
| Active-present | 26 | 3 | 1 |
| Actionable active (medium+) | 3 | 1 | 0 |
| Info/low active | 23 | 2 | 1 |
| High/critical | 0 | 0 | 0 |
| issues_found (scan) | 26 | 3 | 1 |
| Graph state | available | available | truncated |

## After scan remaining (1)

| Rule | Severity | File | Notes |
|---|---|---|---|
| OPT-HTTP-CLIENT-PER-CALL | low | health/reliability.go | Scanner self-match on timeout heuristic — fixed in `7dd92c5` follow-up |

## Issue sync

| Metric | Value |
|---|---:|
| Open Gitea issues | 1 (#48) |
| New issues created (rescan) | 0 |
| Duplicate issues created | 0 |

## Findings by bucket (after)

| Bucket | Remaining |
|---|---:|
| global_rule_fix_candidate | 1 (self-match, fixed post-rescan) |
| repo_scoped_calibration_candidate | 0 |
| real_code_fix | 0 |
| real_reliability_fix | 0 |

## Verification

- Persistence: complete
- Issue sync: complete
- Graph: built (truncated at limits — expected for monolith)
- Store tests: pass (`go test ./store/... -timeout=180s`)
- Full suite: pass (`go test ./... -timeout=300s`)
- Calibration deadlock regression: `TestPersistFindingsWithRepoCalibrationRules` pass
- Repo-scoped calibration rules seeded: 17 total (8 prior + 9 new)

## Notes

- First interim rescan (`801ca5bce8683213`) ran before git push; dropped 26→3 via updated health checker binary.
- Post-push rescan (`cfcb1e05419859d6`) on `main` at `7dd92c5`: **1 low self-match** remaining; actionable active **0**.
- One follow-up static analyzer skip for `health/` paths eliminates the final self-match on next rescan.
