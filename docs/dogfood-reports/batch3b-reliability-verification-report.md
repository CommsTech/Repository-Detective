# Batch 3b reliability verification report — 2026-06-06

## Issues fixed

| Issue | File | Fix |
|-------|------|-----|
| #263 | `api/suppressions_handler.go` | Bind JSON with 400 on error |
| #264 | `api/suppressions_handler.go` | Bind JSON with 400 on error |
| #265, #292 | `graph/orphans.go` | Removed dead `filepath.Base` call |
| #266 | `internal/auth/session.go` | Check HMAC write errors |
| #267, #268 | `internal/security/csrf.go` | Check HMAC write errors |
| #276 | `closure/engine.go` | Log close failure via lifecycle event |

## Commits

- `6d3e317` fix(reliability): resolve Batch 3b ignored-error findings
- `7e0bf50` chore(repo): prevent dogfood artifact bloat
- `908d552` docs(dogfood): all-repo scan readiness

## Tests

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| staticcheck (changed packages, local) | PASS |

## Scans

| Phase | Scan ID | Findings |
|-------|---------|----------|
| Before | `2b335070099f8936` | 1104 |
| After | **`a8bb4cddd72ab80c`** | 1101 |

## Backlog impact

| Metric | Before | After |
|--------|--------|-------|
| Gitea open | 283 | **288** |
| Real active (present in scan) | 48 | **41** |
| Resolved absent | 136 | **143** |
| Duplicates | 68 | 69 |

Open count rose slightly (+5) due to new mappings/backfill items; **real active code-fix backlog decreased by 7**.

## Fingerprints absent

Post-rescan API: no medium findings on `suppressions_handler`, `orphans`, `session`, `csrf`, `closure` paths targeted by Batch 3b.

Gitea issues remain **open by policy** — use verify-closure when operator enables close.

## Regression checks

| Check | Status |
|-------|--------|
| Batch 2 store fixes | Still on `main` (`f64789d`) |
| Batch 3a gosec #316/#323 | Still absent in prior scan |
| Duplicate burst | None observed |
| New duplicate issues | +1 labeled duplicate in classification |

## Next batch

**Batch 3c** — gosec medium / file inclusion (~24 issues) or remaining HEALTH-IGNORED-ERROR outside Batch 3b scope.
