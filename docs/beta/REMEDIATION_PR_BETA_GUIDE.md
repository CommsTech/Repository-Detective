# Remediation PR — private beta guide

## Beta defaults (keep these until verified)

```yaml
remediation_pr_enabled: false
remediation_pr_require_approval: true
remediation_pr_require_tests: true
remediation_pr_use_runner_verification: true
```

## Enablement steps

1. Review remediation planner output on owned repos for 1–2 weeks.
2. Confirm `gitea_token` is configured (Configure page shows “present”).
3. Enable runner verification (native or Gitea Actions) and confirm tests pass on a scratch plan.
4. Set `remediation_pr_enabled: true` in config; restart.
5. Approve a **low/medium** plan manually; click **Create PR** only after checklist passes.

## Beta test checklist

| Test | Expected |
|------|----------|
| Default disabled | Configure shows “disabled”; Create PR unavailable |
| Missing token | PR blocked with clear error |
| Unapproved plan | Blocked |
| High/critical finding | Blocked by default |
| Oversized patch | Blocked by file/line limits |
| Failing tests | No PR; logs visible |
| Passing tests | PR created with evidence |
| Pre-install plan | Never eligible |

## Never during beta

- Auto-enable remediation PR on fresh installs
- Create PRs from pre-install audit findings
- Skip test gate for “speed”

## Related

- [REMEDIATION_PR.md](../REMEDIATION_PR.md)
- [RUNNER_DELEGATION.md](../RUNNER_DELEGATION.md)
- [GITEA_ACTIONS_BACKEND.md](../GITEA_ACTIONS_BACKEND.md)
