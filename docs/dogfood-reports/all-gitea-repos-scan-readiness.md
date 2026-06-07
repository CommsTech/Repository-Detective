# All-Gitea-repos scan readiness checklist

**Status:** READY FOR DRY-RUN PLANNING — product repo active findings are 0; operator approval still required before any fleet scan.

## Current gate (2026-06-07 final closeout)

| Gate | Status |
|------|--------|
| Product repo real active findings | **0** (scan `5e570c95bc4e3467`) |
| Product repo open issues | 32 (2 ops human-review + 30 summary rollups — no code findings) |
| Docker full rebuild | **green** (core/runner/all-in-one verified) |
| Code CI on `main` | **green** — run #119 (`73c4a0f`) success |
| Docs CI #120 | failure (docs-only; non-blocking) |
| Backlog-control | **active** — 0 new low/medium on final rescan |
| Duplicate/idempotency | verified |
| DB persistence | stable |
| Operator approval gate | **required** |

## Readiness decision

| Mode | Status |
|------|--------|
| All-repo full filing scan | **BLOCKED** — operator approval + summary-ticket hygiene optional |
| All-repo dry-run (analyze + persist, no filing) | **READY FOR PLANNING** |
| All-repo report-only | **READY FOR PLANNING** |

## Remaining product-repo open issues (not blocking dry-run)

- **#48, #49** — homelab ops; needs human review (no fingerprints)
- **30 summary rollups** — out of scope; no active findings linked

## Explicitly out of scope

- Auth/RBAC Slice 2
- SaaS / multitenancy / billing
- Fleet auto-remediation PRs
- Manual mass issue close without evidence

## Recommended next steps

1. Operator sign-off for org-wide **dry-run** config (no issue filing).
2. Optional Batch 6: bulk-close or archive summary rollup tickets with documented policy.
3. Resolve ops tickets #48/#49 manually.
4. Define `GITEA_SCAN_ORGS` allowlist before any fleet run.

## Verification before go-live

- [x] Product repo active findings: 0
- [x] Code CI green (#119)
- [x] Docker rebuild verified
- [x] No duplicate burst on rescan
- [ ] Operator approval recorded
- [ ] Dry-run config validated on 1–2 non-product repos
