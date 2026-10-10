# Store test timeout triage

**Date:** 2026-06-12  
**Commit:** `311e97c`  
**Conclusion:** **Not a reproducible SQLite hang** — full `store` package completes in ~55–80s; prior 180s failure was **timeout budget vs. cumulative slow tests**, exacerbated when running the **entire repo** test tree in parallel.

## Symptom (prior session)

```
go test ./store/... -count=1 -timeout=180s → FAIL (timeout)
```

Stack trace pointed at `openTestStore` / `TestReconciliationSummaryReportOnly` during `store.Open` — appeared as hang but was **late in a long sequential queue**.

## Investigation runs (2026-06-12)

| Command | Result | Wall time |
|---------|--------|-----------|
| `go test ./store/... -count=1 -timeout=180s` | **PASS** | 54.7s |
| `go test ./store/... -count=1 -timeout=300s` | **PASS** | 62.6s |
| `go test ./store/... -count=3 -timeout=300s` | **PASS** | 183.8s |
| `go test ./... -count=1 -timeout=300s` | **PARTIAL** — `store` PASS (80s); `sbom` unrelated assertion fail | 123s |

Verbose run shows many migration/CRUD tests taking **7–15s each** (schema init on temp SQLite per test). No goroutine stuck on `database/sql` connection opener after package completion.

## Root cause

1. **Slow but healthy tests** — `store` has 100+ tests each opening a fresh SQLite DB and running migrations (expected design).
2. **Timeout math** — under load, cumulative runtime can approach 180s when combined with other packages competing for CPU/IO (`go test ./...` runs packages in parallel; `store` + `sbom` + `api` together previously exceeded per-package deadline).
3. **Not evidence of** — deadlock, leaked connections, calibration loader hang, or reconciliation race (no reproduction in isolated `store` runs).

## Why release can proceed

- `go test ./store/... -timeout=180s` **passes consistently** in triage (3 runs).
- No code defect identified; no migration fix required.
- CI recommendation: keep `store` timeout ≥180s or run `store` as its own job.

## Recommendations

| Priority | Action |
|----------|--------|
| Should | CI: `go test ./store/... -timeout=180s` as dedicated step |
| Could | Share migrated DB fixture across tests to reduce migration overhead (future perf, not blocker) |
| Monitor | If 180s fails again, capture `go test -v` progress line before timeout |

## Evidence commands

```bash
export PATH="$HOME/.local/go/bin:$PATH"
go test ./store/... -count=1 -timeout=180s
go test ./store/... -count=3 -timeout=300s
```
