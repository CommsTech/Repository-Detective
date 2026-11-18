# Findings ownership, export, and code-quality signals — release evidence

Date: 2026-11-18 UTC. Worktree: previous runner-ingestion-safety source (2026-11-18) plus new findings/README changes; original reference archive `repository-detective-main(20261118-124211).zip`. The delivered patch is **cumulative** against that original reference archive and includes the previous runner-ingestion-safety work, which may not yet have been merged upstream.

## Implemented and bounded

- Opt-in client-side download handling for CSV/JSON: visible preparation/completion/error messages, actual JSON Blob download, content type check (rejects login HTML), no-store and nosniff response headers, and non-JS direct links.
- SQLite migration 30: per-finding local assignee; append-only assignment events with actor/previous/new-name snapshots to preserve attribution when accounts change.
- Finding workbench owner card: enabled users in local-auth mode, owner/admin/security role for writes; viewers/developers cannot assign. No external issue assignee synchronization or notifications are implied.
- README external scanaislop code-quality badge with an explicit non-certification label; independent scores are not used as a trust gate.
- Technical evaluation of candidate deterministic code-quality scanners and writing/agent advice; no unreviewed detector is auto-installed.

## Verification performed in this environment

| Check | Result |
|---|---|
| Go AST parse | **PASS** — 708 source files |
| Go HTML template parse | **PASS** — 42 declarations |
| JavaScript syntax | **PASS** — updated app and export acceptance script |
| CSS parse | **PASS** — theme stylesheet |
| Browser: JSON export | **PASS** — mocked JSON download and filename in headless Chromium |
| Browser: export rejection | **PASS** — mocked HTTP 422 surfaced to operator without silent navigation |
| Browser: local owner component | **PASS** — synthetic rendering at 390, 768, and 1280 px, no page overflow |
| HTML escaping | **PASS** — synthetic untrusted display name rendered escaped |
| Migration 30 DDL | **PASS** — independent SQLite schema/foreign-key smoke |
| Complete Go 1.26.11 test suite | **NOT RUN** — only Go 1.23.2 present and module dependencies unavailable offline |
| Authenticated server-side assignment post | **NOT RUN** — Go integration regression tests written for CI |
| Live finding page, exported production JSON | **NOT RUN** — no authenticated staging server connected |
| External forge owner synchronization | **NOT IMPLEMENTED** — local ownership only |
| Actual isolated scanner integration and score calibration | **NOT IMPLEMENTED** in this pass |

## Required staging acceptance

1. Back up the existing SQLite database. Apply migration 30 on a staging copy; reopen and inspect assignment schema/foreign keys, migration checksum and user records.
2. Run `go test ./...`, `go vet ./...`, `go test -race ./store ./ui` with Go 1.26.11 and all module dependencies. Resolve any failures before merge.
3. Sign in as owner/admin/security and viewer/developer; verify allowed/denied assignment, CSRF checks, disabled accounts, migration upgrades, idempotency, and full assignment history. Confirm API-key-only mode denies local assignment.
4. On a disposable Gitea instance with representative findings, click Export JSON/CSV, inspect downloaded files and check browser console. Verify an expired session produces an error, not HTML masquerading as JSON. Trigger >20,000 matching findings in test data and confirm visible rejection.
5. Run `node scripts/ui-findings-export-acceptance.js` after installing Playwright in the staging CI image, using a scoped test account or key. Check light/dark and narrow layouts for overlaps.
6. Confirm external forge issues remain unchanged on local owner assignment, and a fresh source ZIP/patch is reproduced byte-for-byte. Do not enable unattended third-party issue disclosure based on experimental score or AI-only conclusions.

This patch is a **staging candidate**, not a claim of production security or complete end-to-end verification.
