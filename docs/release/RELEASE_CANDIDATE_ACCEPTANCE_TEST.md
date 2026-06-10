# Release candidate acceptance test plan

Run after each RC sprint batch. Status values: `pass` | `fail` | `blocked` | `not_implemented`.

| ID | Area | Status | Evidence |
|----|------|--------|----------|
| RC-01 | Repo scan | pass | Dogfood reconciliation clean |
| RC-02 | Pre-install audit | pass | Report-only enforced |
| RC-03 | Container image scan | pass | alpine:3.20 live job |
| RC-04 | Git-history secret scan | pass | gitleaks-history in profile |
| RC-05 | SBOM report | partial | UI routes + store; verify in dogfood report |
| RC-06 | Repository map graph | pass | UI route exists |
| RC-07 | Issue filing Gitea | partial | Needs repo-mapping regression |
| RC-08 | Findings detail | partial | RC sprint UX overhaul |
| RC-09 | Learning/calibration | pass | UI + suppress flows |
| RC-10 | AI recommendations | partial | Renamed; CAH gating; strict JSON pending provider |
| RC-11 | Runner delegation | pass | Disabled by default |
| RC-12 | Remediation PR | pass | Disabled by default |
| RC-13 | Configure page | pass | Provider-neutral AI section |
| RC-14 | Health page | pass | `/ui/health` |
| RC-15 | Docs/wiki | blocked | Wiki HTTP 500 |
| RC-16 | UI visual smoke | partial | `scripts/ui-route-smoke-test.sh` |
| RC-17 | Container logs | partial | `scripts/operator-log-health-check.sh` |
| RC-18 | Beta package | partial | `make beta-release` in Phase 10 |
| RC-19 | External clean install | not_implemented | Manual |
| RC-20 | Provider matrix | pass | `docs/ISSUE_PROVIDERS.md` |

Full results: `docs/dogfood-reports/full-application-acceptance-report.md`
