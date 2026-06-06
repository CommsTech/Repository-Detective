# Batch 3a gosec verification report — 2026-06-06

## Issues fixed

| Issue | Fingerprint | Rule | File | Fix |
|-------|-------------|------|------|-----|
| #316 | bugbot-32ea466677b98678 | G115 | `scanners/archive_extract.go` | Safe size check via `uncompressedSizeWouldExceed` (bounds before int64 cast) |
| #323 | bugbot-a668d741a770ea04 | G101 | `ui/api_key_cookie.go` | Renamed cookie constant to `rd_ui_sess` (name only, not a credential) |

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
| After | *Pending post-push rescan* |

## Gosec status

Local targeted gosec on changed files: **clean** for G115/G101.

## Fingerprints absent

*To be confirmed after rescan on `main` post-push.*

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
