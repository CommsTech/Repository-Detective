# Batch 4 — active findings queue

Scan baseline: `cd4cb8d70d357f26`  
Open Gitea issues after resolved-absent closeout: **56**

## Priority order

1. HIGH security (SEC-SQL-CONCAT false positives on `db.Query`)
2. Reliability (HEALTH-IGNORED-ERROR)
3. Gosec medium (permissions / path validation)
4. Out of scope for Batch 4a: REL-INTERNAL-INFRA-REF, DL3018, CKV template noise, HEALTH-COMMENT-BLOCK

## Batch 4a targets (12)

| # | Fingerprint | Source | Rule | Sev | File | Planned fix | Test |
|--:|-------------|--------|------|-----|------|-------------|------|
| 327 | rd-22ea3ab4d75a571d | static | SEC-SQL-CONCAT | high | issuelink/backfill.go:101 | Skip `db.Query == nil` in static FP filter | analyzers/static_test.go |
| 328 | rd-56df68c874191f41 | static | SEC-SQL-CONCAT | high | issuelink/backfill.go:30 | Same | analyzers/static_test.go |
| 196 | rd-d0a05cf6594a828f | health | HEALTH-IGNORED-ERROR | medium | notify/http.go:36 | Check `io.Copy` error | notify tests |
| 198 | rd-17ce603a440b2a8a | health | HEALTH-IGNORED-ERROR | medium | notify/webhook.go:73 | Check HMAC write error | notify tests |
| 199 | rd-d391ab02bcc6f715 | health | HEALTH-IGNORED-ERROR | medium | notify/webhook.go:89 | Document test helper write | notify tests |
| 212 | rd-01184640a65833ca | health | HEALTH-IGNORED-ERROR | medium | patcher/executor.go:198 | Surface issue comment failure | patcher tests |
| 213 | rd-52ca71ddebab9092 | health | HEALTH-IGNORED-ERROR | medium | preinstall/checks.go:189 | Propagate WalkDir errors | preinstall tests |
| 214 | rd-89bda9c93b10f19b | health | HEALTH-IGNORED-ERROR | medium | preinstall/checks.go:38 | Handle workspace walk errors | preinstall tests |
| 215 | rd-8a66eeb95c8d98f0 | health | HEALTH-IGNORED-ERROR | medium | preinstall/runner.go:104 | Check UpdateAuditRequest | manual |
| 322 | rd-ed1c73d74547022b | gosec | G301 | medium | store/store.go:83 | MkdirAll `0o750` | store tests |
| 330 | rd-837f90a24801401d | gosec | G301 | medium | scanners/archive_extract.go:32 | MkdirAll `0o750` | scanners tests |
| 331 | rd-094959790721052d | gosec | G301 | medium | scanners/archive_extract.go:78 | MkdirAll `0o750` | scanners tests |

## Deferred to Batch 4b

| # | Rule | Reason |
|--:|------|--------|
| 312 | G304 preinstall | Fixed in 26eae14; verify on next rescan |
| 321 | G201 store | Parameterized IN query; store-layer FP |
| 324 | G203 ui | Template auto-escape review |
| 332 | G304 archive | Zip-slip validated path; verify post-fix |
| 206, 228 | CKV_SECRET_6 | Template placeholders; not runtime secrets |
| 53–296 | REL-INTERNAL-INFRA-REF | Homelab catalog refs; out of scope |
| 232–262 | DL3018 | Hadolint pin noise |
| 326 | HEALTH-COMMENT-BLOCK | Low priority tech debt |
