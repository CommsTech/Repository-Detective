# Release candidate acceptance test plan

Run after each RC sprint batch. Status values: `pass` | `fail` | `blocked` | `not_implemented` | `partial`.

Last updated: 2026-06-11 (blocker burn-down sprint)

| ID | Area | Status | Evidence |
|----|------|--------|----------|
| RC-01 | Repo scan | pass | Product rescan 0 findings |
| RC-02 | Pre-install audit | pass | Report-only enforced |
| RC-03 | Container image scan | pass | alpine:3.20 prior demo |
| RC-04 | Git-history secret scan | pass | Profile enabled |
| RC-05 | SBOM report | pass | Download proven CycloneDX |
| RC-06 | Repository map graph | pass | UI route 200 |
| RC-07 | Issue filing Gitea | partial | Dry-run + unit tests |
| RC-08 | Findings detail | pass | 37361 live verified |
| RC-09 | Learning/calibration | pass | UI route 200 |
| RC-10 | AI recommendations | pass | Disabled by default |
| RC-11 | Runner delegation | pass | Disabled |
| RC-12 | Remediation PR | pass | Disabled |
| RC-13 | Configure page | pass | Provider-neutral naming |
| RC-14 | Health page | pass | Live healthy |
| RC-15 | Docs/wiki | blocked | Wiki HTTP 500 |
| RC-16 | UI visual smoke | pass | 15/15 routes + screenshots |
| RC-17 | Container logs | pass | Clean |
| RC-18 | Beta package | pass | `make beta-release` |
| RC-19 | External clean install | partial | Beta package from clean tree |
| RC-20 | Provider matrix | pass | GitHub demoted honestly |
| RC-21 | Live RC deploy | pass | `rc-e3e19ec` |
| RC-22 | Product dogfood clean | pass | active-present 0 |

Full results: `docs/dogfood-reports/full-application-acceptance-report.md`
