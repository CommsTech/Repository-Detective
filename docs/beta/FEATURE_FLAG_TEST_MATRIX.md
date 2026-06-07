# Feature flag beta test matrix

Product baseline commit: sprint head (see `BETA_RELEASE_READINESS_REPORT.md`).

| Feature | Config keys | Default | Beta state | Test | Pass | Blocks beta |
|---------|-------------|---------|------------|------|------|-------------|
| Database | `database_enabled` | true | enabled_and_verified | `go test ./store/...` | pending | yes if fails |
| Scheduler | `scheduler_enabled` | true | enabled_and_verified | operator smoke / health | pending | medium |
| Runner delegation | `runner_delegation_enabled`, `runner_shared_secret` | false | disabled_by_default_but_tested | health capability row | pending | no |
| Notifications | `notifications_enabled`, channel URLs | false | disabled_missing_config_with_action | health shows missing channel | pending | no |
| Pre-install audit | `preinstall_audit_enabled` | false | disabled_by_default_but_tested | GET `/ui/preinstall` → 200 | pending | yes if 404 |
| Remediation planner | `remediation_planner_enabled` | true | enabled_and_verified | API + UI smoke | pending | medium |
| Remediation PR | `remediation_pr_enabled` | false | disabled_by_default_but_tested | capability status | pending | no |
| Evidence closure | `evidence_closure_enabled` | true | enabled_and_verified | closure unit tests | pending | medium |
| Operator UI | `operator_ui_enabled` | true | enabled_and_verified | `go test ./ui/...` | pending | yes |
| Scan profiles | `scan_profile` | standard_deterministic | enabled_and_verified | profile tests | pending | no |
| Report-only dry run | `report_only_dry_run` (API) | available | enabled_and_verified | dry-run docs | pass | no |
| Backlog-control | built-in | active | enabled_and_verified | product repo 0 active-present | pass | yes if off |
| Project grouping | DB + `/ui/projects` | new | enabled_and_verified | `TestProjectGroupCRUD` | pending | no |
| SBOM | scan pipeline + `sbom` pkg | new | enabled_and_verified | `go test ./sbom/...` | pending | medium |

## Disabled feature expected behavior

| Feature | When disabled | Expected UI/API |
|---------|---------------|-----------------|
| Pre-install audit | flag false | Page loads; banner explains enable steps |
| Runner delegation | flag false | Health: disabled + config keys |
| Notifications | flag false | Health: disabled + channel list |
| Remediation PR | flag false | Finding page hides PR attempt or shows disabled |

## Security notes

- Disabled ≠ 404 for operator-facing features.
- Beta-disabled auto-remediation outside controlled tests remains **not_beta_supported**.
