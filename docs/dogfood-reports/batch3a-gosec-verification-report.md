# Batch 3a gosec verification report — 2026-06-06

## Issues fixed

| Issue | Fingerprint | Rule | File | Fix |
|-------|-------------|------|------|-----|
| #316 | rd-32ea466677b98678 | G115 | `scanners/archive_extract.go` | Safe size check via `uncompressedSizeWouldExceed` (bounds before int64 cast) |
| #323 | rd-a668d741a770ea04 | G101 | `ui/api_key_cookie.go` | Renamed cookie constant to `rd_ui_sess` (name only, not a credential) |

## Files changed

- `scanners/archive_extract.go`
- `scanners/archive_extract_overflow_test.go` (new)
- `ui/api_key_cookie.go`
- `ui/api_key_auth.go`
- `ui/api_key_auth_test.go`

## Tests run

| Check | Result |
|-------|--------|
| `go test ./scanners/... ./ui/...` | PASS |
| `gosec -include=G115,G101` (target files) | PASS (exit 0) |
| `go vet ./...` | PASS (via full test run) |

## Scan IDs

| Phase | Scan ID |
|-------|---------|
| Before (reference) | `852f2fb850b2b56d` |
| After | **`2b335070099f8936`** (completed, 1104 findings) |

## Gosec status

Local targeted gosec on changed files: **clean** for G115/G101.

## Fingerprints absent

Post-rescan API check: no HIGH findings on `archive_extract.go` or `api_key_cookie.go`.  
Gitea issues **#316** and **#323** remain **open** (expected — `evidence_closure_close_issues=false`; verify via verify-closure after deploy aligns).

## Active backlog (pre-Batch-3a)

| Category | Count |
|----------|-------|
| Active code-fix | 48 |
| Resolved absent | 129 |
| Duplicates | 68 |

Expected active after #316/#323 fix: **46** (if both verified absent).

## Next recommended batch

**Batch 3b** — HEALTH-IGNORED-ERROR reliability findings (#263–#268, #276, #292).

## Notes

- UI cookie rename (`rd_ui_api_key` → `rd_ui_sess`) invalidates existing browser cookies once; users re-enter API key via unlock flow.
- Batch 2 reliability fixes remain in `f64789d` / `38cc304`.
