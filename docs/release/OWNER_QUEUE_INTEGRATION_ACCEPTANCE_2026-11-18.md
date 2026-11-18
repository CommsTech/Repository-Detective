# Finding ownership queues — integration acceptance

**Status:** Source implemented. Static checks and synthetic layout validation completed. Go 1.26.11 tests, authenticated staging browser QA, and upgrade-migration acceptance remain pending.

## Audience and workflow

A signed-in local user opens **Findings → Assigned to me** to find open issues assigned to their account. An operator opens **Unassigned** to identify unclaimed work, assigns the finding from the finding-detail page, and verifies it leaves the unassigned queue. Search, severity, repository, and status filters remain usable in either view. CSV/JSON exports preserve the selected owner filter.

Assignment is only **ownership of investigation**: it does not close the finding, signal trust, send notifications, or update external forge issue assignees. Forge account mapping requires explicit opt-in, identity reconciliation, and permission checks.

## Implemented design

- Query filtering uses the logged-in user ID supplied by the server, never a user ID sent by the browser.
- API-key mode rejects ownership filters; malformed scopes and an unavailable current-user ID fail closed.
- SQL ownership predicates execute before severity ordering and pagination; `CountFindings` uses the same predicates.
- Schema migration 31 creates `idx_finding_assignments_assignee` for efficient lookup. Existing assignment records and history remain unchanged.
- The local-user queue loads assignee labels in a single bounded query; API-key mode does not request owner metadata.
- The header and table expose assignment state without claiming that investigation or remediation has been verified.

## Acceptance matrix

| Check | Expected behavior | Local result |
|---|---|---|
| New database and migration 31 | Schema creates and list/count match | Go test written, **not run** |
| Upgrade existing schema 30 | Assignment/history survives migration | **Not run** |
| Owner A and B | Both users see only their assigned findings | Go test written, **not run** |
| Unassigned queue | Only findings with no active assignment appear | Go test written, **not run** |
| Pagination and filtered exports | Stable count, matching records, preserved scope | Go test written, **not run** |
| API-key and malformed scope | Never fall back to a global queue | Code safeguards implemented; staging **not run** |
| Read-only user assignment attempt | Write rejected; reading permitted | Existing authorization test, full suite **not run** |
| Light/dark, mobile/desktop, long names | No full-page overflow; accessible tab state | Synthetic HTML layout checked; live sign-in **not run** |
| CI `go test ./...`, `go vet ./...` | Pass on Go 1.26.11 | **Not run** |

## Next release checks

1. Run `go test ./...` and `go vet ./...` in the pinned Go 1.26.11 Gitea runner.
2. Upgrade a copy of the current production SQLite database, validate migration 31, and measure findings-list latency under concurrent scan ingestion.
3. Sign in as two different local users, a read-only user, and an administrator. Confirm all queue and export boundaries with at least 201 distinct findings.
4. Check representative phone/tablet/desktop layouts in both themes and keyboard-only navigation; test a 100-character repository name and international display names.
5. Confirm no external forge issues or assignees change when local ownership changes.
6. Publish only after the staging smoke checks pass. The source patch is **not automatically deployed**.
