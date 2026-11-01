# Issue / finding reconciliation UX

## Problem

Scan finding counts, open findings in the DB, and open Gitea issues often differ — especially in report-only beta. The UI must explain why without looking like a bug.

## Concepts

| Term | Meaning |
|------|---------|
| Scan findings | Instances detected in a specific scan |
| Active-present | Open findings present in the latest/referenced scan |
| Open findings (DB) | Deduplicated findings with status `open` |
| Report-only / no issue | Open findings without a linked open forge issue |
| Forge open issues | Tracked `external_issues` with state `open` |
| Mapped open | Same as forge open (fingerprint-linked) |
| Unmapped open issues | Mapped forge issues whose finding is not in this scan |
| Resolved-verified open | Finding fixed in scan but forge issue still open |
| Duplicates | Findings linked via `canonical_finding_id` |

## UI panel

Shown on:

- Repository detail (`/ui/repos/:id`)
- Scan detail (`/ui/scans/:id`)

Includes:

- Side-by-side counts (clickable where filters exist)
- Report-only / filing explanation paragraph
- Sync status (`issue_sync_status`, persistence)
- Mismatch warning only when unexpected (not for report-only)

## API

`GET /api/v1/repos/:id/reconciliation?scan_id=optional`

Returns `store.ReconciliationSummary` JSON.

## AMMBER validation pattern

Expected when report-only or filing off:

- Many findings, few or one mapped issue (e.g. `#205` for hardcoded secret)
- `counts_differ_expected: true`
- No false mismatch warning when `dry_run_report_only` or issue filing disabled

## Actions

- **Run reconciliation preview** — existing `/ui/repos/:id/reconcile` (does not enable filing)
- **Browse findings** — filter open findings for triage
