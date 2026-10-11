# Batch 3b — HEALTH-IGNORED-ERROR reliability queue

**Date:** 2026-06-06  
**Scan:** `2b335070099f8936`  
**Status:** IMPLEMENTED (pending rescan verification)

## Issues targeted

| Issue | File:line | Ignored call | Fix |
|-------|-----------|--------------|-----|
| #263 | `api/suppressions_handler.go:88` | `ShouldBindJSON` | Return 400 on bind error |
| #264 | `api/suppressions_handler.go:106` | `ShouldBindJSON` | Return 400 on bind error |
| #265 | `graph/orphans.go:156` | `filepath.Base` dead call | Removed useless statement |
| #292 | `graph/orphans.go:156` | same | same |
| #266 | `internal/auth/session.go:57` | `mac.Write` | Check write errors |
| #267 | `internal/security/csrf.go:39` | `mac.Write` | Check write errors |
| #268 | `internal/security/csrf.go:41` | `mac.Write` | Check write errors |
| #276 | `closure/engine.go:339` | `CloseIssue` | Record lifecycle event on failure |

## Verification plan

- `go test ./...`
- Rescan `commstech/Repository-Detective` on `main`
- Confirm fingerprints absent from scan `2b335070099f8936` baseline
- `POST /api/v1/findings/{id}/verify-closure` where applicable (policy: keep open)
