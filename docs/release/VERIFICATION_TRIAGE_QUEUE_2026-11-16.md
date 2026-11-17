# Triage queue accuracy and UX — verification receipt

**Worktree:** based on user-provided `repository-detective-main(2).zip` (2026-11-16 Arizona / 2026-11-17 UTC). This is **not** an upstream release or deployment receipt.

## Changes

- Linked forge issues now select a single open/preferred ticket so queue totals, pagination and exports do not inflate or duplicate canonical findings.
- SQL filters `PriorityOnly` critical/high rows before limit and sorts severity before recency only when a caller requests prioritization. Default callers remain newest-first.
- Focus is constrained to **open** critical/high regardless of contradictory query-string status/suppression flags.
- Findings page includes server-backed total count, 50/100-row pagination, filter-preserving navigation, locked focus-mode options, readable operator instructions, UTC header, responsive horizontal table scrolling.
- CSV/JSON exports iterate the full filtered dataset in 200-row batches rather than silently truncating at the store's 200-row per-query cap. A 20,000-row resource limit rejects overly large exports explicitly. Unknown formats now return HTTP 400.
- CSV text is protected against spreadsheet formula interpretation. JSON preserves the original finding title/field contents.
- Dependency detail no longer equates a scanner-reported fixed version with low complexity or low regression risk; closure text requires actual scan evidence after deployment.

## Local validation run

| Check | Result |
|---|---|
| `gofmt` for modified Go source and tests | Formatted; source parse check required below |
| `git diff --check` | Passed |
| `node --check ui/static/app.js` | Passed |
| `go test ./ui ./store` | **BLOCKED**: Go 1.26.10 is required; environment has Go 1.23.2 and cannot resolve/download `proxy.golang.org` |
| Docker build and image smoke | **NOT RUN** |
| Disposable Gitea / live scanner E2E | **NOT RUN** |
| Real-browser/mobile/screen-reader regression | **NOT RUN** |

## Acceptance commands — run on an isolated Go 1.26.10 runner

```sh
go version # must be go1.26.10

go test ./store ./ui -run 'TestFindingPriorityQueueDoesNotHideOlderCritical|TestFindingsQueueOneRowPerFindingWithMultipleForgeIssues|TestFindingsQueuePaginationAndCompleteExport|TestFindingsExportRejectsUnknownFormat' -count=1

go test ./... -count=1
go vet ./...
node --check ui/static/app.js
```

Then build an isolated image (`docker compose build`) and test UI `/ui/findings`, focus, page 2+, CSV/JSON export, keyboard navigation and mobile horizontal scrolling with the target Gitea stack. Run a disposable dependency finding through detection → planner → human fix/PR → rescan → closure, including an incomplete-scan negative test. Do not turn on unattended dependency PRs on the strength of these unit tests.

## Known gaps after this pass

- No full end-to-end production certification; AI dependency-patching implementation and sandbox approval remain separate engineering milestones.
- Exports are not transactional snapshots across simultaneous scans; obtain formal reports during a stable scan window.
- Large inventories beyond 20,000 matches require filters; paginated bulk API and asynchronous export jobs may be appropriate later.
- Gitea live-system browser and accessibility checks still need independent verification.
- API-key URL handling and authentication paths need their own focused security validation; use local-session authentication for the UI and never share keyed URLs.
