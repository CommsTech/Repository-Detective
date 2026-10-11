# Batch 2 — P1 reliability queue (READY — NOT STARTED)

**Gate status:** CI job #1855 all steps **PASS** on `9a1f629`; post-stabilization rescan **complete** (`f85f8e66e3c9fc9a`). **Implementation not started.**

## Scope

- Category: P1 reliability — ignored error returns
- Packages: `handlers/`, `store/` (max ~20 issues)
- No broad refactors; no fleet repos; no manual issue closes

## Selection notes

- Open Gitea issues titled `[MEDIUM] Potential reliability issue: ignored error return` mostly map to **graph/**, **api/**, etc. — not `handlers/` / `store/`.
- Production code scan (2026-06-06) shows **ignored errors concentrated in `store/`**; `handlers/` production code has none (only test helpers).
- Queue below is from **source inspection** of non-test `store/*.go` — align to matching Gitea fingerprints during implementation via rescan.

## Queue (max 20)

| # | Issue | File:line | Planned fix |
|---|-------|-----------|-------------|
| 1 | TBD (rescan match) | `store/sqlite_queries.go:626-629` | Return/aggregate dashboard summary errors |
| 2 | TBD | `store/sqlite_queries.go:847` | Handle `RowsAffected` error |
| 3 | TBD | `store/recorder.go:141` | Handle `json.Marshal` error on scan summary |
| 4 | TBD | `store/recorder.go:226-246` | Handle marshal/unmarshal errors on finding meta |
| 5 | TBD | `store/recorder.go:323` | Handle marshal error |
| 6 | TBD | `store/migrations.go:546,555` | Document defer rollback or log on migration paths |
| 7 | TBD | `store/calibration_sqlite.go:24,151` | defer rollback — verify tx commit semantics |
| 8 | TBD | `store/calibration_sqlite.go:233,310-312` | Handle `QueryRowContext` / Scan errors |
| 9 | TBD | `store/calibration_sqlite.go:318-324` | Propagate list/summary errors to caller |
| 10 | — | `handlers/` | No production ignored-error sites found — skip unless rescan adds |

*Issues #11–20 reserved for additional `store/` sites surfaced by Batch 2 rescan diff.*

## Planned fix types

1. **Return error** — DB/write paths, never swallow
2. **Log with context** — non-fatal background tasks only
3. **HTTP error response** — if surfaced through handlers
4. **Documented intentional ignore** — with comment when truly safe (e.g. rollback after commit failure)

## Verification plan (when batch starts)

```bash
go test ./handlers/... ./store/...
go vet ./...
staticcheck ./handlers/... ./store/...
./scripts/operator-smoke-test.sh
```

Rescan → verify absent fingerprints → `resolved-verified` label only; keep issues open if `evidence_closure_close_issues=false`.

## Status

**Batch 2 started:** YES (2026-06-06)  
**Batch 2 completed:** YES — see `batch2-p1-reliability-report.md`  
**Batch 2 allowed:** YES  
**Post-rescan:** `84a96d5ad8458965` — store `HEALTH-IGNORED-ERROR` **0**
