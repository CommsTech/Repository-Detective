# Current readiness reconciliation

**Date:** 2026-06-11 (private beta expansion packet)  
**Commit:** `6d011cf` (pre-expansion docs); expansion docs commit pending  
**Method:** Read existing docs + live checks (no rescans, no redeploy)

## Git

| Field | Value |
|-------|-------|
| Latest commit | `1c862aa` — docs(release): update RC marketing readiness |
| Branch | `main` |
| Origin sync | **synced** (`main...origin/main`) |
| Working tree | clean |

## Live system

| Field | Value |
|-------|-------|
| Container | `repository-detective:rc-e3e19ec` |
| Status | healthy (23h+ uptime at reconciliation) |
| `/health` | `status=healthy`, `database_healthy=true` |
| Version reported | `dev` (image tag `rc-e3e19ec`) |
| Redeploy since burn-down | **no** (per instruction) |

## Product dogfood (live SQLite, 2026-06-11)

| Field | Value |
|-------|-------|
| Latest reconcilable scan | `926a5f56a26f03c9` |
| Scan status | completed |
| Active-present | **0** |
| Actionable active | **0** |
| High/critical | **0** |
| Informational active | **0** |
| Forge open issues | **0** |

Prior regression baseline (stale): scan `e42b3e175e313904`, active-present **21**, actionable **4**, informational **17**.

## Evidence sources (commits `581d534`–`1c862aa`)

| Topic | Document | Result |
|-------|----------|--------|
| Regression baseline | `rc-active-present-regression-baseline.md` | Root cause documented |
| 21-finding classification | `rc-active-present-21-classification.md` | All 21 classified |
| Code fixes | `581d534` (analyzers/static.go, health/skip.go) | Scanner self-match |
| Rescan | `rc-active-present-rescan-report.md` | 0/0/0 |
| Gitea issue target | `gitea-issue-target-regression-report.md` | Partial |
| GitHub provider | `github-issue-provider-rc-status.md` | Not release-proven |
| SBOM download | `sbom-artifact-download-verification.md` | PASS |
| Screenshots | `rc-screenshot-visual-qa-report.md` | 12 pages |
| External install | `external-clean-install-report.md` | Partial |
| Wiki | `gitea-wiki-http-500-triage.md` | Blocked |
| Marketing gate | `MARKETING_READINESS_GATE.md` | NOT READY |
| Acceptance | `RELEASE_CANDIDATE_ACCEPTANCE_TEST.md` | Updated |

## Live spot checks (2026-06-11)

| Check | Result |
|-------|--------|
| SBOM download `/ui/scans/926a5f56a26f03c9/sbom/download` | HTTP **200** |
| SBOM artifact | CycloneDX, 895 components |
| Screenshot count | **12** PNGs in `docs/assets/screenshots/` |
| Beta package | `dist/repository-detective-beta/` exists |
| Wiki clone | HTTP **500** (unchanged) |

## Tests (prior session, not re-run)

| Suite | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go test ./store/...` | PASS (deadlock regression) |
| `go vet ./...` | PASS |
| staticcheck v0.5.1 | PASS |
| Docker build `rc-dogfood` + `all-in-one` | PASS |
| Operator smoke / UI route / log health | PASS |
| `make beta-release` | PASS |

## Reconciliation conclusion

Blocker burn-down sprint is **complete**. Private beta expansion packet added. Product dogfood is **clean**. Marketing remains **blocked** on wiki, live Gitea filing proof, and full external VM install.

**Recommended next move:** onboard first invited tester (report-only) + operator/server validation — **not** another product fix sprint.

## Expansion artifacts (this packet)

| Document | Purpose |
|----------|---------|
| `docs/beta/PRIVATE_BETA_RC_RELEASE_NOTES.md` | Tester-facing RC notes |
| `docs/beta/PRIVATE_BETA_TEST_SCOPE.md` | Allowed / not allowed |
| `docs/beta/PRIVATE_BETA_*_TEMPLATE.md` | Feedback intake |
| `docs/beta/PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md` | Operator procedures |
| `docs/dogfood-reports/gitea-wiki-server-repair-plan.md` | Wiki blocker |
| `docs/dogfood-reports/gitea-filing-controlled-test-plan.md` | Filing proof (not_run) |
| `docs/dogfood-reports/external-clean-install-test-plan.md` | VM proof (not_run) |
