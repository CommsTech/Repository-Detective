# Active-present 26 burn-down baseline

Recorded: 2026-06-02  
Latest commit: `14ddb7c`  
Latest scan: `27fbd37be97ef5f7`

## Product repo (commstech/Repository-Detective)

| Metric | Value |
|---|---:|
| Open Gitea issues | 1 (#48) |
| Active-present | 26 |
| Actionable active (medium+) | 3 |
| High/critical | 0 |

### Severity

| Severity | Count |
|---|---:|
| info | 6 |
| low | 17 |
| medium | 3 |

### Source / rule breakdown

| Count | Source | Rule |
|---:|---|---|
| 8 | maintainability | HEALTH-MANY-PARAMS |
| 3 | maintainability | HEALTH-LARGE-FILE |
| 3 | tech_debt | HEALTH-DEPRECATED |
| 3 | tech_debt | HEALTH-TECH-PHRASE |
| 2 | maintainability | HEALTH-DEEP-NEST |
| 2 | reliability | HEALTH-IGNORED-ERROR |
| 2 | maintainability | HEALTH-LARGE-FUNC |
| 2 | performance | HEALTH-READ-ALL |
| 1 | reliability | HEALTH-HTTP-NO-TIMEOUT |

## DB deadlock fix summary

Prior full-suite failure: `ListRepoCalibrationRules` was invoked inside `BeginTx` during `PersistScanFindingsBatch`, causing SQLite lock contention and test hangs.

Fix (commit `14ddb7c` lineage): load repo calibration rules **before** `BeginTx` in `store/findings_persist_sqlite.go`. Rules are applied in-memory during the batch loop; no nested DB reads inside the transaction.

Regression guard: `TestPersistFindingsWithRepoCalibrationRules` in `store/persistence_test.go` (30s hang timeout).

## Runner / remediation state

| Setting | Value |
|---|---|
| Live container | healthy (`repository-detective`) |
| Runner delegation | disabled |
| Worker running | no |
| Remediation PR | disabled |
| Gitea Actions backend | disabled |
| All-repo scan | off |

## Remaining blockers

1. Gitea wiki HTTP 500 on `repository-detective.wiki.git` push (server-side; does not block burn-down)
2. Gitea issue #48 (operator task)
3. act_runner token rotation if earlier soak token was real

## Stop conditions

- Do not enable runner delegation by default
- Do not create PRs or enable remediation PR
- Do not globally suppress from product-only evidence
- High/critical protected from automatic downgrade
- Target: active-present near 0 after fix + repo-scoped calibration + rescan
