# Learning and efficiency audit — 2026-06-06

## Existing mechanisms (verified in codebase)

| Mechanism | Location | Purpose |
|-----------|----------|---------|
| Fingerprints | findings + issue bodies | Dedup and closure matching |
| Lifecycle events | `store` + `closure` | Audit trail for verify/close |
| Suppressions / false positives | `api/suppressions_handler`, calibration | Operator calibration |
| Calibration rule stats | `store/calibration_sqlite.go` | Noisy rule detection |
| Calibration recommendations | DB + UI | Proposed suppressions (not auto-applied by default) |
| Reconciliation engine | `reconcile/engine.go` | Scan/issue delta |
| Evidence closure | `closure/engine.go` | Verify absent fingerprints |
| Duplicate linking | lifecycle labels + reconcile | Canonical issue tracking |
| External issue mappings | `external_issues` table | Forge ↔ finding link |
| Issue filing guards | `issues/manager.go` + lifecycle commit | Prevent duplicate burst |
| Scanner results | `scanner_results` per scan | Variance tracking |

## Gaps addressed this sprint

1. **Classification export** — `scripts/export-classify-open-issues.py` now writes `real-active-backlog-report.md` and recognizes `resolved-verified` open-by-policy.
2. **Closure close failures** — recorded as `closure_issue_close_failed` lifecycle events (not silently ignored).
3. **Dogfood retention** — `docs/dogfood-reports/README.md` documents what to commit vs keep local.

## Recommended before all-repo scan

- Keep `calibration_auto_apply=false`
- Use per-run issue caps (existing config)
- Dry-run / report-only mode first (see all-gitea-repos-scan-readiness.md)
- Record scanner variance separately from code regressions (scanner_results table)

## Not added (deferred)

- ML-based ranking
- Global suppression rules across repos
- Heavyweight new subsystems
