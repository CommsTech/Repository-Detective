# Scan policy

Repository Detective filing behavior for connected repositories.

## Modes

| Mode | When | Issue filing |
|------|------|----------------|
| `production_self_hosted` | `auto_create_issues: true` and repo policy allows | Files/updates per policy |
| `private_beta_safe` | `auto_create_issues: false` or `issue_policy: off` | Report-only enforced |
| `report_only_dry_run` | Operator checks dry run or API `report_only_dry_run: true` | 0 issues |
| `preinstall_audit` | Pre-install audit flow | Always 0 issues, 0 PRs |

## Normal connected repo scan

Default (no dry run):

- Issue filing enabled when `ShouldCreateForgeIssues(effective)` is true
- Existing issues updated by fingerprint
- New issues for eligible findings
- Duplicates prevented
- Backlog-control respected (`dogfood_backlog_control_*`)
- `reporting.max_issues_per_scan` cap applied

## Manual scan

- Dry run checkbox **unchecked** when issue filing is enabled globally/per-repo
- Dry run checkbox **checked and locked** when filing disabled by policy
- Preflight summary states whether issues/PRs will be created

## Pre-install audit

Always report-only. Generates promotional-quality report and optional disclosure draft. Never files upstream issues or creates PRs.

## Configuration keys

- `auto_create_issues` → global `issue_policy` (`all` / `off`)
- Per-repo `issue_policy`, `policy_level`
- `report_only_dry_run` — per-scan API/UI override
- `dogfood_backlog_control_enabled` and related keys
- `reporting.max_issues_per_scan`

## Resolver

`store.ResolveScanFilingPolicy()` is the single effective policy resolver for UI preflight and manual scan handlers.

Legacy `REPOSITORY_DETECTIVE_*` env vars remain supported; preferred prefix is `REPOSITORY_DETECTIVE_*`.
