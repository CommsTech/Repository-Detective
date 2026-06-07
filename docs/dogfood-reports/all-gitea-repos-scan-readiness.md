# All-Gitea-repos scan readiness checklist

**Status:** BLOCKED — product repo active findings cleared; await CI green + operator approval.

## Current gate (2026-06-07)

| Gate | Status |
|------|--------|
| Product repo real active findings | **0** (scan `68cab1ba3dc0591d`) |
| Product repo open issues | 43 (mostly summaries / ops / human-review) |
| Docker core rebuild | **green** (deterministic user setup) |
| Docker all-in-one verify | partial (homelab — run `scripts/docker-build-verify.sh`) |
| CI on `main` | pending run #119 |
| Backlog-control | **active** — 0 new low/medium on final rescan |
| Duplicate/idempotency | verified — 0 duplicate burst |
| DB persistence | stable — persistence complete on final scan |
| Scanner variance | documented (gosec/staticcheck/hadolint timeouts in archive mode) |
| Operator approval gate | **required** before any fleet scan |

## Readiness decision

- **All-repo scan:** BLOCKED until CI is green and operator explicitly approves dry-run.
- **Dry-run / report-only planning:** eligible once CI completes green (active findings already 0).
- **Full filing fleet scan:** not ready — open issue hygiene + human-review items remain.

## Explicitly out of scope

- Auth/RBAC Slice 2
- SaaS / multitenancy / billing
- Fleet auto-remediation PRs
- Manual mass issue close without evidence

## Recommended next steps

1. Confirm CI run #119 green on `73c4a0f`.
2. Run full `scripts/docker-build-verify.sh` on homelab runner.
3. Batch 5: classify remaining 43 open issues (summaries #48/#49 human-review).
4. Operator sign-off for org-wide **dry-run** (analyze + persist, no filing).

## Scope and discovery

- [ ] Define org/user list (`GITEA_SCAN_ORGS` or explicit repo allowlist)
- [ ] Exclude archived, fork, mirror, and test sandboxes by default
- [ ] Document repo discovery API and pagination limits
- [ ] Operator approval gate before enabling scheduled all-repo scans

## Rate limits and performance

- [ ] Per-repo analyze rate limit (avoid forge + runner overload)
- [ ] Max concurrent scans (scheduler queue depth)
- [ ] Database size monitoring (`data/bugbot.db` growth)
- [ ] Issue filing cap per scan/run (existing config — verify values)

## Issue policy

- [x] Duplicate prevention enabled (lifecycle guards on `main`)
- [x] Evidence closure enabled for dogfood (`close_issues=true` in homelab config)
- [x] Backlog-control pauses low/medium filing during burn-down
- [ ] Per-run issue cap without starving long-lived active findings
- [ ] Summary/code-review issues out of scope for auto batches

## Scanner baseline

- [x] Record scanner availability per scan (`scanner_results`)
- [x] Treat scanner variance separately from code regressions
- [x] AI disabled by default for fleet scans
- [ ] Token budget / no LLM auditor unless explicitly enabled

## Modes (recommended rollout)

1. **Dry-run** — analyze + persist findings, no issue filing
2. **Report-only** — dashboard/exports, no Gitea issues
3. **File issues** — capped per repo, operator-approved repos only

## Rollback

- [ ] Stop scheduler / disable auto-analyze
- [ ] Document label `duplicate` / canonical linking if issue storm occurs
- [ ] Restore DB from `deployment-backups/` if needed

## Verification before go-live

- [x] Product repo (Bugbot) batch 4b: 0 real active findings
- [ ] CI green on `main`
- [x] No duplicate burst on single-repo rescan
- [x] Classification export shows explainable open count
