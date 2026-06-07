# Feature flag beta test matrix

Updated: Beta UX + Release Gate sprint

Configure page: `/ui/configure#<section>` — each capability **Configure** action deep-links here.

| Feature | Configure link | Beta state | Test | Pass |
|---------|----------------|------------|------|------|
| Database | `#database` | enabled_and_verified | `go test ./store/...` | pass |
| Scheduler | `#scheduler` | enabled_and_verified | health page | pass |
| Runner delegation | `#runner-delegation` | disabled_by_default_but_tested | configure section | pass |
| Notifications | `#notifications` | disabled_missing_config_with_action | configure secrets rows | pass |
| Pre-install audit | `#preinstall-audit` | disabled_by_default_but_tested | GET `/ui/preinstall` 200 | pass |
| Remediation planner | `#remediation-planner` | enabled_and_verified | unit tests | pass |
| Remediation PR | `#remediation-pr` | disabled_by_default_but_tested | health link + configure | pass |
| Evidence closure | `#evidence-closure` | enabled_and_verified | closure tests | pass |
| Operator UI | `#operator-ui` | enabled_and_verified | `go test ./ui/...` | pass |
| Scan profiles | `#scan-profile` | enabled_and_verified | profile tests | pass |
| Report-only dry run | `#report-only-dry-run` | enabled_and_verified | API docs | pass |
| Backlog-control | n/a (policy) | enabled_and_verified | 0 active-present | pass |
| SBOM | `#sbom` | enabled_and_verified | `go test ./sbom/...` | pass |
| Project grouping | `/ui/projects` | enabled_and_verified | CRUD test | pass |

## Disabled feature expected behavior

| Feature | When disabled | Expected |
|---------|---------------|----------|
| Pre-install audit | flag false | 200 page + configure instructions |
| Remediation PR | flag false | Configure shows keys + beta default disabled |
| Runner delegation | flag false | Configure shows missing secret guidance |

## Security

- Secrets: present/missing only on Configure page.
- Disabled features must not 404.
