# Active-present 79 burn-down rescan report

Recorded: 2026-06-09

## Scans

| Field | Before | After |
|-------|--------|-------|
| Scan ID | `4f8617f80f1ef1e8` | `27fbd37be97ef5f7` |
| Files analyzed | 929 | 934 |
| Graph state | available | **available** |

## Active-present delta

| Metric | Before | After | Change |
|--------|-------:|------:|-------:|
| Active-present | 79 | **26** | **−53** |
| Actionable active (medium+) | 12 | **3** | −9 |
| Info/low active | 67 | **23** | −44 |
| High/critical | 0 | **0** | 0 |

## Remaining findings (after)

| Rule | Count |
|------|------:|
| HEALTH-MANY-PARAMS | 8 |
| HEALTH-TECH-PHRASE | 3 |
| HEALTH-LARGE-FILE | 3 |
| HEALTH-DEPRECATED | 3 |
| HEALTH-READ-ALL | 2 |
| HEALTH-LARGE-FUNC | 2 |
| HEALTH-IGNORED-ERROR | 2 |
| HEALTH-DEEP-NEST | 2 |
| HEALTH-HTTP-NO-TIMEOUT | 1 |

## Gitea issues

| Check | Result |
|-------|--------|
| Open issues before | 1 (#48) |
| Open issues after | 1 (#48) |
| New issues created | **0** |
| Duplicate issues created | **0** |
| Issue sync | complete |
| external_issues | synced |

## Interim rescan note

Scan `67d38abbbc558723` (pre-push) remained at 79 active-present but shifted severity mix (actionable 3, informational 76) after repo calibration wiring. Scan `27fbd37be97ef5f7` with expanded health skips produced the −53 reduction.

## Bucket summary (remaining 26)

| Bucket | Approx. count |
|--------|--------------:|
| repo_scoped_calibration_candidate | 14 |
| docs_only_low_priority | 6 |
| intentional_ignore_with_context | 4 |
| needs_human_review | 2 |

## Next steps

- Push burn-down commits to Gitea `main` so remote clone matches local health rules
- Optional second pass on remaining 26 (mostly maintainability + tech-debt phrases in production paths)
- Gitea wiki HTTP 500 remains server-side blocker
