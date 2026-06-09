# Runner delegation — private beta guide

## Beta defaults

| Key | Default | Notes |
|-----|---------|-------|
| `runner_delegation_enabled` | `false` | Must opt in |
| `runner_mode` | `core` | Use `native` when enabling |
| `runner_require_hmac` | `true` | Do not disable in production |
| `gitea_actions_test_backend_enabled` | `false` | Separate from native delegation |

## Beta test checklist

1. **Disabled state** — System Health shows “Runner delegation disabled” with Configure link.
2. **Enable native runner** — Set secret + `runner_delegation_enabled: true` + `runner_mode: native`.
3. **Worker ping** — `POST /api/v1/runner/ping` with valid HMAC returns `ok`.
4. **Claim job** — Worker claims queued scan job; invalid HMAC rejected.
5. **Nonce replay** — Reused nonce within TTL rejected.
6. **Result ingest** — Completed job persists scan findings on core.
7. **Timeout** — Expired job marked failed/timed out.
8. **Gitea Actions** — Remains off until `gitea_actions_test_backend_enabled: true`.

## Token hygiene

- **Never** commit Gitea act_runner registration tokens or `runner_shared_secret`.
- **Rotate** any Gitea act_runner registration token that was pasted in chat or logs before registering new runners.
- Store tokens in runner host env or secret manager only.
- Native RD worker uses `REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET` — separate from Gitea act_runner registration tokens.

## Compute isolation

- Native workers should run in dedicated containers/VMs, not on the main RD server.
- Pre-install audits delegated to workers remain **report-only** (no issues/PRs).

## Failure states

| State | Operator action |
|-------|-----------------|
| `queued` (stuck) | Check worker heartbeat; verify HMAC secret matches |
| `failed` | Read job error in UI; check runner logs |
| `expired` | Increase timeout or reduce repo size |
| HMAC 401 | Sync `runner_shared_secret` between core and worker |

## Related docs

- [RUNNER_DELEGATION.md](../RUNNER_DELEGATION.md)
- [GITEA_ACTIONS_BACKEND.md](../GITEA_ACTIONS_BACKEND.md)
- [RUNNERS.md](../RUNNERS.md)
