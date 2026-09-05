# Runner worker test baseline

Recorded: 2026-06-09

## Latest commit (baseline)

`8e8ae09` — docs(dogfood): verify runner and remediation live behavior

## Live revision

`df7e7f2` (pre-worker-packaging redeploy)

## Runner delegation state

| Setting | Value |
|---------|-------|
| `runner_delegation_enabled` | `false` (product default) |
| `runner_mode` | not active |
| Native worker API | ping, claim, result (HMAC) |
| Worker packaging | in progress this sprint |

## Remediation PR state

| Setting | Value |
|---------|-------|
| `remediation_pr_enabled` | `false` |
| Dry-run verification | planned (no PR creation) |

## Token handling

| Item | Status |
|------|--------|
| `.env` in git | not staged |
| `runner_shared_secret` in git | not staged |
| Gitea act_runner registration token | **rotate before any new act_runner registration** |
| Runner secret | generate via `openssl rand -hex 32` → `.env` only |

## Product repo (Gitea)

| Item | Status |
|------|--------|
| Open operator issue | #48 (expected) |
| Wiki push | HTTP 500 (server-side blocker) |

## Planned controlled test scope

1. Enable delegation in **test config/env only** (one worker, max 1 concurrent job).
2. Allowed job types initially: `graph`, `sbom`, `remediation_verify` (no preinstall_audit).
3. Enqueue **one** delegated graph job on product repo (`commstech/Repository-Detective`) via operator API.
4. Failure/rollback: stop worker, verify timeout messaging, disable delegation, confirm local path.
5. Remediation PR dry-run: verify gate only — **no PR created**.

## Hard stops

- No all-repo scanning
- No pre-install audit delegation until graph/SBOM proven
- No Remediation PR unless explicitly approved in Phase 6
