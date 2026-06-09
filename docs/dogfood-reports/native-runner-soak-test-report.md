# Native runner soak test report

Recorded: 2026-06-09  
Core revision: `8d5da54` (image `repository-detective:all-in-one`)  
Worker: `bin/repository-detective-runner --mode worker`

## Test window

| Setting | Value |
|---------|-------|
| Delegation enabled | yes (temporary `-e` override) |
| Workers | 1 (`rd-native-runner-soak-1`) |
| Allowed job types | graph, sbom, remediation_verify |
| HMAC + nonce | verified (jobs completed without 401) |

## Soak sequence

| # | Job type | Job ID | Status | Notes |
|---|----------|--------|--------|-------|
| 1 | graph | `rj-d806b14e92295838` | **completed** | metrics-only result persisted |
| 2 | sbom | `rj-c2f8bcd9ef7f9c16` | **completed** | scanner_count=1 |
| 3 | remediation_verify | `rj-9071994035cf0905` | **completed** | dry-run verify, 951 files |

## Heartbeat

- Worker visible in `GET /api/v1/runner/workers` with capabilities `[graph, sbom, remediation_verify]`.
- `last_seen_at` updated during test window.

## Failure recovery

| Step | Result |
|------|--------|
| Enqueue graph while worker running | job entered `running` |
| `pkill` worker mid-job | worker stopped |
| Stuck/running job | remained until worker restart or cancel |
| Worker restart | new worker re-registered |
| Subsequent graph job | worker claimed and completed after restart |

## Load observation

- Core container enqueued/ingested only; clone + graph/SBOM compute ran on worker host.
- Main server did not spike on graph build during delegated jobs (compute offloaded).

## Logs redaction

- No shared secret or token values observed in worker stdout during soak.

## Rollback

| Step | Result |
|------|--------|
| Stop worker | `pkill -f repository-detective-runner --mode worker` |
| Recreate core without delegation override | **done** |
| `/health` runner_delegation_enabled | **false** |

## Runner delegation left enabled

**No** — restored to disabled after test window.
