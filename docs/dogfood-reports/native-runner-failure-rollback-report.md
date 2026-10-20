# Native runner failure and rollback report

Recorded: 2026-06-09

## Test 1 — Worker offline, job queued

| Step | Result |
|------|--------|
| Stop worker (`pkill repository-detective-runner`) | OK |
| Enqueue SBOM job `rj-76096a9d3d20f41c` | `status: queued` |
| Wait 20s without worker | Remained **queued** (not stuck forever) |

## Test 2 — Worker recovery

| Step | Result |
|------|--------|
| Restart worker | Claimed queued job |
| Job completion | `status: completed` with sbom scanner result |

## Test 3 — Stuck running job (prior graph attempt)

| Step | Result |
|------|--------|
| Failed HMAC submit left job `running` | Observed on `rj-e6d36538bd329104` |
| Operator cancel | `POST /api/v1/runner/jobs/:id/cancel` → cancelled |
| Capacity gate | New jobs blocked until cancel (`runner job capacity reached`) |

## Test 4 — Disable delegation rollback

| Step | Result |
|------|--------|
| Set `REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED=false` in `.env` | OK |
| Restart `repository-detective` container | OK |
| `/health` | `runner_delegation_enabled: false` |
| Worker stopped | No background worker process |

## Test 5 — Local execution path

With delegation disabled, core resumes in-process scan path (default). No all-repo scan triggered during rollback test.

## Actionable UI/errors

| State | Operator sees |
|-------|----------------|
| Delegation disabled | System Health + Configure show disabled state |
| Capacity reached | API `runner job capacity reached` |
| Worker offline | Jobs remain queued until worker returns or cancel |
| Failed auth | Worker log `runner authentication failed` (fixed via compact result payload) |

## Timeout

Full `runner_job_timeout_seconds` (900s) not exercised in this window; cancel used for stuck jobs. Expiry path exists via `ExpireStaleRunnerJobs` on claim.
