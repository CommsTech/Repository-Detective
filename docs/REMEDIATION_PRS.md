# Safe Remediation PRs (Repository Detective)

Repository Detective — **Inspect. Analyze. Improve.**

This phase creates **branches and pull requests only** for approved, low-risk remediation plans. It does **not** merge PRs, close issues, rotate secrets, or run broad auto-fixing.

**Disabled by default.** Enable explicitly after reviewing policy and validation allowlists.

## Enable

```yaml
remediation_planner_enabled: true   # required — plans must exist first
remediation_pr_enabled: true
remediation_pr_branch_prefix: repository-detective/fix
remediation_pr_require_approval: true
remediation_pr_max_files_changed: 3
remediation_pr_max_diff_lines: 100
remediation_pr_validation_timeout_seconds: 300
```

Environment variables (prefer `REPOSITORY_DETECTIVE_*`; legacy `REPOSITORY_DETECTIVE_*` via envcompat):

```text
REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED
REPOSITORY_DETECTIVE_REMEDIATION_PR_BRANCH_PREFIX
REPOSITORY_DETECTIVE_REMEDIATION_PR_REQUIRE_APPROVAL
REPOSITORY_DETECTIVE_REMEDIATION_PR_MAX_FILES_CHANGED
REPOSITORY_DETECTIVE_REMEDIATION_PR_MAX_DIFF_LINES
REPOSITORY_DETECTIVE_REMEDIATION_PR_VALIDATION_TIMEOUT_SECONDS
```

## Eligibility (hard rules)

All must pass before a PR is attempted:

| Rule | Requirement |
|------|-------------|
| Global toggle | `remediation_pr_enabled: true` |
| Plan status | `approved` (when `remediation_pr_require_approval: true`) |
| Safety flags | `safe_for_auto_pr: true`, `requires_human_review: false` |
| Risk | `regression_risk: low`, `fix_complexity: small` |
| Repository | Connected Gitea repo with clone URL |
| Validation | At least one allowlisted validation command on the plan |
| Patcher | Deterministic patcher exists for the rule |
| Forbidden | No secret, dependency major, gosec, graph, architecture, audit-only, or advisory plans |

Ineligible attempts return **HTTP 400** with `blocked_reasons` and a checklist.

## Allowed patch types (Phase 1)

| Source | Examples |
|--------|----------|
| staticcheck | Unnecessary `fmt.Sprintf("literal")` → `"literal"` (S1039) |
| hadolint | Add `--no-install-recommends` to obvious `apt-get install` lines (DL3015) |

Patches are bounded by `remediation_pr_max_files_changed` and `remediation_pr_max_diff_lines`.

If a plan is eligible but no patcher exists: blocked with **“no patcher available for this rule yet.”**

## Not allowed yet

- Secrets, credential rotation, history purge
- Dependency updates or lockfile edits
- Graph/orphan deletion, architecture rewiring
- gosec high-risk fixes, Checkov permission changes
- Test generation, broad refactors
- Commands requiring network installs (`npm install`, `pip install`, etc.)

## Validation command allowlist

Only fixed-argv commands without shell metacharacters:

| Allowed | Not allowed |
|---------|-------------|
| `go test ./...` | `bash -c`, `sh -c`, `make` |
| `go vet ./...` | `npm install`, `pip install` |
| `staticcheck ./...` | `curl`, `wget`, `docker build`, `kubectl` |
| `hadolint <relative Dockerfile path>` | `terraform apply` |

Commands run in the patched workspace with timeout, bounded output, and redacted logs. Failed validation **prevents PR creation**.

## Git workflow

1. Shallow clone connected repo (token-authenticated remote; token never logged)
2. Create branch `{remediation_pr_branch_prefix}/{short-fingerprint}`
3. Apply deterministic patch
4. Run allowlisted validation commands
5. Commit and push branch (**never** push to default/protected branch)
6. Open PR via Gitea API
7. Comment on linked issue with branch, PR URL, validation summary

Issues **remain open** until a future evidence-based closure phase (merge + rescan + fingerprint gone).

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/remediation/:plan_id/attempt-pr` | Run patch attempt (sync) |
| GET | `/api/v1/remediation/:plan_id/patch-attempts` | List attempts for plan |
| GET | `/api/v1/patch-attempts/:attempt_id` | Single attempt record |

## UI

Finding detail → remediation section shows eligibility checklist, blocked reasons, **Create remediation PR** (when eligible), latest patch attempts, validation results, and PR link.

Warning displayed: Repository Detective creates PRs only for approved low-risk plans. It never auto-merges.

## Beta rollout order

For owned repos, prioritize fixes in this order:

1. Critical/high true positives with approved plans
2. High-confidence medium findings with low regression risk
3. Safe staticcheck/hadolint low-risk fixes
4. Reliability/test-gap improvements after human approval

Do **not** enable broad auto-fixing across all repositories. Use the allowlist gates above and keep third-party audits report-only.

## Database

Migration **v12** adds `patch_attempts` (status, diff summary, validation JSON, PR URL, errors).

## Token safety

- Gitea token used only by core process, not runners
- Tokenized clone URLs are never logged
- Git error output is sanitized
- Validation output is redacted and truncated

## Rollback

1. Set `remediation_pr_enabled: false` and restart — no new PRs.
2. Open PRs/branches already created in Gitea are untouched; close manually if needed.
3. `patch_attempts` rows are historical audit data (migration v12 is additive).

## Next phase

**Evidence-based closure** — close or mark resolved only after PR merged, rescan completed, fingerprint disappeared, and gates passed.

See also: [REMEDIATION.md](REMEDIATION.md), [POLICY.md](POLICY.md), [SECURITY_HARDENING.md](SECURITY_HARDENING.md).
