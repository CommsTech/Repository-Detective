# All-Gitea-repos scan readiness checklist

**Status:** PREPARED — do **not** run all-repo scan until operator approval.

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

- [ ] Duplicate prevention enabled (lifecycle guards on `main`)
- [ ] Evidence closure with `close_issues=false` until operator opts in
- [ ] Per-run issue cap without starving long-lived active findings
- [ ] Summary/code-review issues out of scope for auto batches

## Scanner baseline

- [ ] Record scanner availability per scan (`scanner_results`)
- [ ] Treat scanner variance separately from code regressions
- [ ] AI disabled by default for fleet scans
- [ ] Token budget / no LLM auditor unless explicitly enabled

## Modes (recommended rollout)

1. **Dry-run** — analyze + persist findings, no issue filing
2. **Report-only** — dashboard/exports, no Gitea issues
3. **File issues** — capped per repo, operator-approved repos only

## Learning and calibration

- [ ] Calibration recommendations reviewed before apply
- [ ] Per-repo noisy-rule stats (`calibration_rule_stats`)
- [ ] No global unsafe suppression across repos
- [ ] Reconciliation after each scan

## Rollback

- [ ] Stop scheduler / disable auto-analyze
- [ ] Document label `duplicate` / canonical linking if issue storm occurs
- [ ] Restore DB from `deployment-backups/` if needed

## Verification before go-live

- [ ] Product repo (Bugbot) stable: Batch 2/3 verified
- [ ] CI green on `main`
- [ ] No duplicate burst on single-repo rescan
- [ ] Classification export shows explainable open count

## Explicitly out of scope for first fleet scan

- Auth/RBAC Slice 2
- SaaS / multitenancy / billing
- Fleet auto-remediation PRs
- Manual mass issue close
