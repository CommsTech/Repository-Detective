# Runner and remediation live verification

Recorded: 2026-06-09

## Deploy

| Item | Value |
|------|-------|
| Git commit | `df7e7f2` |
| Image | `repository-detective:all-in-one` |
| Container | `repository-detective` (host network, port 8081) |
| `/api/v1/status` version | `df7e7f2` |

## Pre-install failed audit UX

| Field | Value |
|-------|-------|
| Audit ID | `0389d922-bf01-4bfa-944b-9a5fc38efc7e` |
| Trigger | Nonexistent public GitHub repo (clone failure) |
| Status | `failed` |
| Risk score display | **unavailable** (not 0/100) |
| Recommendation display | **audit failed** |
| `risk_unavailable` | `true` |
| Failure stage | `clone` |

## Pre-install successful audit (regression)

| Field | Value |
|-------|-------|
| Audit ID | `061578b7-2df7-4d13-b79f-69e0265928f7` |
| Status | `completed` |
| Risk score display | `10 / 100` |
| Recommendation | `safe` |
| Issues created | 0 |
| PRs created | 0 |

## Runner delegation

| Check | Result |
|-------|--------|
| `runner_delegation_enabled` | `false` |
| System Health message | Runner delegation disabled (clear) |
| Configure link | `#runner-delegation` present |
| Native runner ping | Routes not registered until delegation + secret enabled (expected) |
| Gitea Actions backend | `gitea_actions_test_backend_enabled: false` |

## Remediation PR

| Check | Result |
|-------|--------|
| `remediation_pr_enabled` | `false` |
| Configure section | Shows approval + test requirement keys |
| PR created during verification | **No** |

## Secrets

| Check | Result |
|-------|--------|
| Tokens in git commits | None |
| API responses | Failure text redacted patterns verified in unit tests |
| Gitea runner registration token | Not committed |

## Operator smoke test

`./scripts/operator-smoke-test.sh` — **passed** (healthy, version df7e7f2).

## Notes

- Private IP URL blocked at create (`400`) before audit queue — expected SSRF guard.
- Clone-failure audits now correctly show risk unavailable instead of misleading 0/100.
