# Remediation PR enablement

Remediation PR creates **gated pull requests** from approved, low-risk remediation plans. It is **disabled by default**.

## Safe flow

```text
finding → remediation plan → operator review → approve plan →
patch in scratch workspace → allowlisted validation in explicitly selected local mode (NOT sandboxed) →
tests pass → create PR with evidence
```

Pre-install audits **never** create remediation PRs.

## Configuration

```yaml
remediation_pr_enabled: false
remediation_pr_require_approval: true
remediation_pr_require_tests: true
remediation_pr_use_runner_verification: true
remediation_pr_block_high_critical_without_manual_override: true
remediation_pr_allowed_severities:
  - low
  - medium
remediation_pr_max_files_changed: 5
remediation_pr_max_diff_lines: 300
remediation_pr_branch_prefix: repository-detective/fix
remediation_pr_validation_timeout_seconds: 300
```

Requires `gitea_token` with repository write permission when enabled.

## Gates

| Gate | Behavior |
|------|----------|
| Global toggle | `remediation_pr_enabled: true` |
| Plan approval | `approved` when `remediation_pr_require_approval: true` |
| Severity | high/critical blocked unless override disabled |
| Patch size | Enforced by max files/lines |
| Tests | **All** validation commands must be allowlisted and pass; unsupported commands block eligibility |
| Runner verify | **Not wired to PR executor**. `true` blocks eligibility; `false` opts into control-plane validation, which is **not isolated** |
| Pre-install | Plans with `audit_id` blocked |

## UI

- **Configure** — enable switch, token present/missing, size limits, approval and test requirements
- **Finding page** — plan, risk, affected files, diff summary, test plan, Approve + Create PR (disabled until gates pass)

## Disable quickly

Set `remediation_pr_enabled: false` and restart — no new PRs will be created.

See [REMEDIATION_PRS.md](REMEDIATION_PRS.md) for detailed eligibility rules and [REMEDIATION.md](REMEDIATION.md) for planner context.

## Execution limitation (beta)

Runner-backed remediation verification is not yet integrated into the PR executor. The safer default `remediation_pr_use_runner_verification: true` therefore **blocks PR execution** instead of silently executing untrusted validation on the control plane. An operator may explicitly select `false` for a trusted, owned repository, but that runs `go test`, `go vet`, or other allowlisted tooling against repository code on the application host with no proven container sandbox. Do not use this mode for arbitrary third-party repositories.

A recorded existing PR prevents normal retries. This is **not** an atomic multi-instance job claim; a remote PR can exist even if writing the local attempt record failed. Inspect the forge before retrying an error. See [RD-008B](RD-008B_CLASS_B_EXECUTION.md).
