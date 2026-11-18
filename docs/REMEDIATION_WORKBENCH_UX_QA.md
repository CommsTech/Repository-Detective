# Remediation workbench: operator experience and acceptance

This document describes the **finding-detail remediation UI**, not an AI patch-execution guarantee.

## User goal

In a few seconds, an operator should identify whether Repository Detective has only detected an issue, generated a proposal, recorded a patch attempt, or gathered post-change verification evidence. Beginners should have a clear next action; advanced operators should retain access to raw scanner evidence, proposed validation commands, and underlying IDs.

## Implemented presentation

- A four-stage visual trail shows **Detected → Plan → Patch / PR → Verify**. It is an evidence-state summary, **not a progress percentage, security certification, or proof that all previous stages are complete**.
- A proposed plan remains an advisory plan. Plan approval is different from patch execution; recorded patch attempts do not imply a PR merged or passed tests.
- **Verified** appears in the verification stage only if the backend supplied closure evidence with status `verified`. The panel otherwise shows the existing closure status or `Not verified`.
- Approve/reject actions are grouped, retaining their existing POST routes, CSRF tokens, and server authorization. No new backend capabilities, auto-merge actions, or external calls were introduced.
- Secondary and destructive-adjacent controls have a quieter visual hierarchy than the principal review action, using CSS without a new JS/chart dependency.
- The trail collapses from four desktop columns to two tablet columns and a single mobile stack. Status text wraps rather than overflowing.

## Testing performed on the source snapshot

- Standard-library Go HTML-template rendering with synthetic **no plan**, **proposed**, **approved / attempt recorded**, and **verified** states: passed.
- Synthetic Chromium layout checks for those four states at **390 / 768 / 1280 px**, light and dark themes (**24 cases**): passed. Checks included document and stage overflow, four-stage presence, CSRF fields, and verified-state representation. These do **not** represent a live authenticated application or production data.
- Go source formatting, README relative links, and patch reproducibility checks are part of the handoff.

## Staging acceptance gates (not satisfied by synthetic tests)

1. Run `go test ./...` and `go vet ./...` using the version specified in `go.mod`, and execute `ui/finding_workbench_test.go`.
2. On a disposable Gitea repository, test finding with **no plan**, **proposed**, **approved**, **rejected**, **PR ineligible**, **PR attempted**, **scan blocked**, **finding persists**, and **verified**. Inspect actual persisted state and backend audit trail after every mutation.
3. Run `node scripts/ui-finding-workbench-check.js` with a dedicated staging account and finding ID. Verify keyboard navigation, screen reader labels, 200–400% zoom, themes, mobile widths, and long tool/error strings. Never run mutations on real repositories merely to validate UI.
4. Ensure unauthorized attempts to call approve/reject, PR, and closure routes are denied server-side. The UI does not replace authorization.
5. For any AI-enabled case, record token usage, model/provider errors, proposed commands, runner isolation evidence, test outcomes, and post-change scan coverage. A suggestion alone must not appear as a verified fix.
6. Validate that all routes load without 500s and benchmark operator flows and page latency on representative finding backlogs.

## Related documentation

[Finding workbench QA](FINDING_WORKBENCH_DESIGN_AND_UX_QA.md) · [Runner evidence](release/RUNNER_RESULT_EVIDENCE_INTEGRITY_2026-11-17.md) · [Runner operations](RUNNERS.md) · [Release readiness](release/CURRENT_PUBLIC_RELEASE_READINESS.md).
