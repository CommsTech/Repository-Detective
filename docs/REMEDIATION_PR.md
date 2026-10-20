# Remediation PR enablement

Remediation PR creates **gated pull requests** from approved, low-risk remediation plans. It is **disabled by default**.

## Safe flow

```text
finding → remediation plan → operator review → approve plan →
patch in scratch workspace → tests (sandbox / native runner / Gitea Actions) →
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
| Tests | Validation commands must pass |
| Runner verify | Optional native runner or Gitea Actions backend |
| Pre-install | Plans with `audit_id` blocked |

## UI

- **Configure** — enable switch, token present/missing, size limits, approval and test requirements
- **Finding page** — plan, risk, affected files, diff summary, test plan, Approve + Create PR (disabled until gates pass)

## Disable quickly

Set `remediation_pr_enabled: false` and restart — no new PRs will be created.

See [REMEDIATION_PRS.md](REMEDIATION_PRS.md) for detailed eligibility rules and [REMEDIATION.md](REMEDIATION.md) for planner context.
