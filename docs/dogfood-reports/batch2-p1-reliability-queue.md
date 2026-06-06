# Batch 2 — P1 reliability queue (PREPARED, NOT STARTED)

**Gate status:** CI not green as of 2026-06-06 — **do not implement until run #1845+ passes.**

## Scope

- Category: P1 reliability — ignored error returns
- Packages: `handlers/`, `store/` (max ~20 issues)
- No broad refactors; no fleet repos

## Selection criteria

From `product-repo-issue-burndown-plan.md`:

> Batch 2: P1 reliability ignored-error returns (packages: handlers, store, scanners) — 20 issues max

This queue focuses **handlers/** and **store/** first (scanners deferred to avoid scope creep).

## Queue (to be populated from rescan + Gitea export)

After CI green, export open issues matching:

- Labels/categories: reliability, ignored error, errcheck
- Paths: `handlers/*.go`, `store/*.go`
- Rule IDs: staticcheck SA9003, errcheck, custom REL-* rules

| # | Issue | Fingerprint | File | Planned fix |
|---|-------|-------------|------|-------------|
| — | *Pending CI gate + issue export* | | | return error / log with context |

## Planned fix types

1. **Return error** — DB/write paths, never swallow
2. **Log with context** — non-fatal background tasks
3. **HTTP error response** — handler paths
4. **Documented intentional ignore** — with comment when truly safe
5. **Test update** — where behavior changes

## Verification plan (when batch starts)

```bash
go test ./handlers/... ./store/...
go vet ./...
staticcheck ./handlers/... ./store/...
./scripts/operator-smoke-test.sh
```

Rescan → verify absent fingerprints → `resolved-verified` label only; keep issues open if `evidence_closure_close_issues=false`.

## Status

**Batch 2 started:** NO  
**Batch 2 completed:** NO
