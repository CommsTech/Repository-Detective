# Native runner worker quickstart

Run **one** Repository Detective native worker to offload supported scans, graph/SBOM, and remediation-verification work. A worker running in Docker is not a hostile-code security boundary.

## Prerequisites

1. Core service ready (`/health/ready` succeeds).
2. `runner_shared_secret` set in `.env` on core **and** worker (never commit).
3. `runner_delegation_enabled: true` and `runner_mode: native` (test window only).
4. Rotate any exposed Gitea `act_runner` registration token before registering act_runner.

## Build worker

```bash
go build -o bin/repository-detective-runner ./cmd/repository-detective-runner
```

Or use Docker:

```bash
docker compose -f docker-compose.yml -f docker-compose.runner.example.yml up -d
```

## Worker environment

| Variable | Example |
|----------|---------|
| `REPOSITORY_DETECTIVE_CORE_URL` | `http://127.0.0.1:8081` |
| `REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET` | from `.env` |
| `REPOSITORY_DETECTIVE_RUNNER_ID` | `rd-native-runner-1` |
| `REPOSITORY_DETECTIVE_RUNNER_ALLOWED_JOB_TYPES` | `scan,graph,sbom,remediation_verify` |

## Enqueue a controlled test job (operator API)

```bash
curl -X POST -H "X-Repository-Detective-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"repository_id":1,"job_type":"graph"}' \
  http://127.0.0.1:8081/api/v1/runner/jobs/enqueue-delegated
```

## Verify

- System Health → Runner job queue shows worker heartbeat and job status.
- `GET /api/v1/runner/workers` lists recent heartbeats.
- `GET /api/v1/runner/jobs` shows queued → running → completed.

## Rollback

1. Stop worker container/process.
2. Set `runner_delegation_enabled: false` (or remove env override).
3. Restart core — **only** scans explicitly configured for core execute locally. Recheck policies before sending untrusted code to core.

See [RUNNER_DELEGATION.md](RUNNER_DELEGATION.md).

See [sandbox threat model and trust evidence](SECURITY_SANDBOX_AND_TRUST.md). Live Gitea Actions dispatch is **not supported** as an RD backend.
