# Native runner enablement plan (one worker)

Recorded: 2026-06-09

## Goal

Prove one Repository Detective native worker can safely claim and complete a delegated job without broad feature changes.

## Config keys (test window only)

| Key | Test value | Product default |
|-----|------------|-----------------|
| `runner_delegation_enabled` | `true` | `false` |
| `runner_mode` | `native` | `core` |
| `runner_shared_secret` | `.env` only | empty |
| `runner_max_concurrent_jobs` | `1` | `2` |
| `runner_require_hmac` | `true` | `true` |
| `runner_allowed_job_types` | graph, sbom, remediation_verify | full list |

Environment overrides (preferred):

```bash
REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED=true
REPOSITORY_DETECTIVE_RUNNER_MODE=native
REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET=<openssl rand -hex 32>
REPOSITORY_DETECTIVE_RUNNER_MAX_CONCURRENT_JOBS=1
REPOSITORY_DETECTIVE_RUNNER_ALLOWED_JOB_TYPES=graph,sbom,remediation_verify
```

## Runner secret handling

1. Generate: `openssl rand -hex 32`
2. Set **same value** on core (`.env`) and worker host.
3. Never commit to git, docs, or screenshots.
4. Rotate Gitea act_runner registration token separately if previously exposed.

## Allowed job types (initial)

- `graph` — first live validation target
- `sbom` — second target after graph success
- `remediation_verify` — dry-run verification only
- **Excluded until proven:** `preinstall_audit`, full `scan`

## Stop conditions

- Worker cannot authenticate (HMAC failures)
- Job stuck > `runner_job_timeout_seconds`
- Unexpected issue/PR creation from delegated path
- Main server still runs heavy work locally for delegated jobs

## Rollback steps

1. `docker stop rd-native-runner-1` or kill worker process
2. Remove `REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED` override (or set `false`)
3. Restart `repository-detective` container
4. Confirm `/health` shows `runner_delegation_enabled: false`
5. Confirm manual scan on core still completes locally

## Expected load reduction

- Delegated graph job: core enqueues only; worker clones + builds graph; core ingests result.
- CPU/memory spike moves to worker host/workspace tmpfs, not main container during job execution.

## Enqueue method

Operator API (single repo, no fleet scan):

`POST /api/v1/runner/jobs/enqueue-delegated` with `{"repository_id":1,"job_type":"graph"}`
