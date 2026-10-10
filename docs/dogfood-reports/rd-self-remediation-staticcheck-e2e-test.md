# Repository Detective Staticcheck S1039 Self-Remediation Test

**Date:** 2026-06-06  
**Repository:** commstech/Repository-Detective  
**Rule:** staticcheck S1039 (unnecessary `fmt.Sprintf` on string literal)  
**Outcome:** **PASS** — full loop completed with verified closure

## Test configuration

| Setting | Value |
|---------|-------|
| `remediation_planner_enabled` | true |
| `remediation_pr_enabled` | true (test window only; reverted to false after test) |
| `evidence_closure_enabled` | true |
| `evidence_closure_close_issues` | false |
| `workspace_mode` | **archive** (required for staticcheck on this repo) |

## Target finding

| Field | Value |
|-------|-------|
| Finding ID | **11658** |
| Fingerprint | `rd-c68376af29742113` |
| File | `internal/dogfood/staticcheck_e2e_marker.go:8` |
| Severity | low |
| Category | code_quality |
| Source | staticcheck |
| Gitea issue | **none** (finding backfilled manually for controlled E2E; no issue filed) |

Controlled marker commit on `main`: `d679d0b` (`test(dogfood): add controlled staticcheck S1039 marker for E2E`).

## Workflow artifacts

| Step | ID / URL |
|------|----------|
| Remediation plan (approved) | **rp-08270977049e02e8** |
| Patch attempt (PR opened) | **pa-12474c8d554fbbf5** |
| Failed attempts (debugging) | `pa-a9d9285d616b3296`, `pa-4451966602b0033c`, `pa-1821e7f6a02501c6` |
| Pull request | [#288](https://git.commsnet.org/commstech/repository-detective/pulls/288) |
| PR branch | `repository-detective/fix/repository-detective-c6837` |
| PR commit | `1ee87dcbfb0e415acdde014ef1c312a0394a4886` |
| Merge commit (manual) | **a0d32599ff21ab94bbbef905791ebf920d542d84** |
| Post-merge rescan | **6bdad6c92f1c8a0c** |

## Plan eligibility (confirmed before PR)

- severity: low (not high/critical security)
- `safe_for_auto_pr`: true
- `requires_human_review`: false
- `regression_risk`: low
- `fix_complexity`: small
- validation commands (final): `go test ./internal/dogfood/...`, `staticcheck ./internal/dogfood/...`

## PR diff summary

- **Files changed:** 1 (`internal/dogfood/staticcheck_e2e_marker.go`)
- **Lines changed:** +1 / −2 (Gitea); patcher counted 7 lines including import removal
- **Change:** replace `fmt.Sprintf("repository-detective-staticcheck-e2e")` with string literal; drop unused `import "fmt"`

## Validation output

Final successful patch attempt:

```
go test ./internal/dogfood/...: passed
staticcheck ./internal/dogfood/...: passed
```

Earlier attempts failed because:

1. S1039 patcher removed `fmt.Sprintf` but left unused `fmt` import (compile failure).
2. Plan used repo-wide `go test ./...`, which failed on unrelated `operator` package tests in the all-in-one image (checkov/gosec tool-status expectations).

## Closure verification

```
POST /api/v1/findings/11658/verify-closure
```

| Check | Result |
|-------|--------|
| Fingerprint absent in rescan | **yes** (`fingerprint_present: false`) |
| staticcheck evidence | **clean** (`scanner_status: clean`) |
| Closure status | **verified** |
| Issue label | **N/A** — no linked Gitea issue for this backfilled finding |
| Issue closed | **no** (`evidence_closure_close_issues: false`) |
| Finding status | `resolved_verified` |

## Core fixes required during this test (local / hot-swap; commit pending)

| Area | Fix |
|------|-----|
| `patcher/rules_staticcheck.go` | Remove unused `fmt` import after S1039 patch; ignore `fmt.` mentions in comments |
| `patcher/validate.go` | Allow package-scoped `go test` / `staticcheck` patterns (not only `./...`) |
| `remediation/tests.go` | Emit package-scoped validation for staticcheck findings |
| `scanners/staticcheck_scanner.go` | Skip non-JSON stderr lines |
| `internal/security/env.go` | Prepend Go toolchain to subprocess PATH |
| `Dockerfile` | Copy Go toolchain into scanner image (durable deploy) |

## Bugs found during test

1. **S1039 patcher incomplete** — replaced literal `fmt.Sprintf` but did not remove unused `fmt` import → validation compile failure.

2. **Comment false positive on import cleanup** — marker file comment contained `fmt.Sprintf`; naive `strings.Contains(content, "fmt.")` skipped import removal.

3. **Repo-wide validation too broad** — default plan used `go test ./...` / `staticcheck ./...`; unrelated `operator` tests fail in all-in-one image even when the patch is correct.

4. **staticcheck scanner fragile in default deploy** — missing Go in PATH, stderr merged into JSON output (`parse_failed`), timeouts on cold toolchain.

5. **API workspace mode insufficient** — staticcheck did not produce reliable findings until repo `workspace_mode` set to **archive**.

6. **Finding ingest gap** — scan reported staticcheck findings but S1039 was not persisted until manual DB backfill (needs product fix).

7. **Controlled marker required** — no organic S1039 on `main` at test start; E2E used `internal/dogfood/staticcheck_e2e_marker.go`.

8. **SQLite lock during scans** — host/container concurrent writes caused `database is locked`; use API or read-only queries during active scans.

## Recommendation: staticcheck remediation in beta

**Do not enable staticcheck auto-PR in private beta by default.**

Allow only after shipping:

1. Durable Go toolchain in production image (not hot-swap).
2. Archive workspace default (or reliable API-mode staticcheck).
3. Scanner → DB ingest verified for staticcheck source rows.
4. S1039 patcher import cleanup (merged in this test’s core fixes).
5. Package-scoped validation commands for single-file Go fixes.

**Hadolint DL3018** remains the primary approved low-risk auto-PR rule for beta. Staticcheck S1039 is a **second-class** candidate until infra gaps above are closed.

## Post-test config

- `remediation_pr_enabled` restored to **false** in `config/config.yaml`.
- Do not expand auto-remediation beyond approved low-risk rules.
- **Stop remediation expansion** — two successful E2Es (DL3018 + S1039) are sufficient proof for beta scope.

## Next product direction (operator recommendation)

1. **Private beta ops for one week** — collect operator feedback on detect → plan → approve → PR → verify loop.
2. Then **Auth/RBAC Slice 2** — route-level permissions + admin user management.
