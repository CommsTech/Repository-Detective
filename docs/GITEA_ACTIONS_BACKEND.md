# Gitea Actions verification backend

Optional integration for **repo-native test execution** after remediation planning. This is **not** the primary scan path.

## Separation from native runners

| Use case | Backend |
|----------|---------|
| Full-repo scan, SBOM, graph | Native RD runner |
| Pre-install audit | Native RD runner (report-only on core) |
| Remediation patch verification | Native runner **or** Gitea Actions workflow |
| PR creation | Core only, after tests pass + operator approval |

## Configuration

```yaml
gitea_actions_test_backend_enabled: false
gitea_actions_workflow_name: repository-detective-verify.yml
gitea_actions_trigger_mode: workflow_dispatch
gitea_actions_timeout_seconds: 1800
gitea_actions_require_operator_approval: true
```

## Workflow template

Repository Detective ships a reusable workflow at:

`.gitea/workflows/repository-detective-verify.yml`

Typical steps:

- Checkout
- Language-specific tests (when detected)
- Optional staticcheck / ruff / shellcheck
- Optional SBOM check

## act_runner registration

Gitea act_runner uses a **registration token** obtained from your Gitea instance (instance/org/repo scope). Store this token only in runner configuration or secrets — **never** in the Repository Detective git repo.

**If a registration token was pasted in chat or screenshots, rotate/revoke it in Gitea before registering new act_runner instances.**

Repository Detective native workers use `REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET` (HMAC) — this is a different credential from Gitea act_runner registration tokens.

## When Repository Detective triggers a workflow

- Remediation plan approved by operator
- `remediation_pr_use_runner_verification: true`
- `gitea_actions_test_backend_enabled: true`
- Operator approval gate satisfied (`gitea_actions_require_operator_approval`)

Repository Detective records:

- Workflow run ID
- Status
- Logs URL
- Pass/fail result

## Not used for

- Pre-install audits on untrusted repos
- Automatic PR creation without explicit remediation PR enablement
- Running arbitrary repo scripts without approval gates

## Failure handling

- Failed workflow → remediation PR blocked, failure logs shown
- Timeout → job marked failed, operator notified
- Backend disabled → verification falls back to native runner or local sandbox

See [RUNNER_DELEGATION.md](RUNNER_DELEGATION.md) and [REMEDIATION_PR.md](REMEDIATION_PR.md).
