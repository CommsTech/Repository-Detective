# Issue filing policy end-to-end report

Date: 2026-06-08

## Normal connected repo (filing allowed)

| Check | Result |
|-------|--------|
| Policy resolver | `ResolveScanFilingPolicy` — production mode when `auto_create_issues: true` |
| Duplicate prevention | Fingerprint-based idempotency in issue manager |
| Backlog-control | Active — blocks new low/medium when configured |

## Manual dry-run (product repo)

| Check | Result |
|-------|--------|
| Scan ID | `47993b1eecb63e47` |
| `report_only_dry_run` | true |
| Expected issues created | 0 |

## Pre-install audit

| Check | Result |
|-------|--------|
| Issue creation | 0 (separate audit store; no forge calls) |
| PR creation | 0 |
| Private IP default | blocked (`preinstall_allow_private_networks: false`) |
| Disclosure approval | operator must mark reviewed before external submission |

## Controlled filing test

Not run on non-product repos per safety rules. Product repo fixture issues closed with evidence instead of new filing.

## Reconciliation visibility

`GET /api/v1/repos/1/reconciliation` exposes:

- `findings_with_open_issue` vs `findings_without_open_issue`
- `forge_open_issues` vs `mapped_open_issues` / `unmapped_open_issues`
- `issue_filing_enabled` and `dry_run_report_only` per scan

Example: fingerprint `rd-a7fb8b9ed08e7f8f` maps to Gitea #205 (historical); open findings without issues remain visible in queue.
