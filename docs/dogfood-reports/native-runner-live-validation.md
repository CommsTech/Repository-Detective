# Native runner live validation

Recorded: 2026-06-09

## Deploy revision

| Item | Value |
|------|-------|
| Git commit | `abb3eaa` (+ worker fixes uncommitted at validation) |
| Core image | `repository-detective:all-in-one` |
| Live version | `abb3eaa` during test window |

## Runner worker

| Field | Value |
|-------|-------|
| Runner ID | `rd-native-runner-1` |
| Mode | `worker` (daemon) |
| Binary | `bin/repository-detective-runner` |
| Heartbeat | verified via `GET /api/v1/runner/workers` |
| Capabilities | graph, sbom, remediation_verify |

## Delegated graph job (success)

| Field | Value |
|-------|-------|
| Job ID | `rj-e15f6cbd25c6c1e6` |
| Job type | `graph` |
| Repository | `commstech/Repository-Detective` (id=1) |
| Scan ID | `89644a43a8e001a9` |
| Started | ~2026-06-09T03:10:11Z |
| Finished | ~2026-06-09T03:10:13Z |
| Status | **completed** |
| Files analyzed (worker) | 931 |
| Result bytes | 474 (metrics-only transport) |
| Summary | `files_analyzed: 931`, `warnings: 1` (graph node/edge counts) |

## Delegated SBOM job (recovery)

| Field | Value |
|-------|-------|
| Job ID | `rj-76096a9d3d20f41c` |
| Job type | `sbom` |
| Status | **completed** after worker restart |
| Scanner results | sbom tool status recorded |

## Main-server load observation

- Core enqueued jobs and persisted results; clone + graph/SBOM compute ran on worker host (`/tmp/rd-runner-workspaces`).
- During job execution, worker log shows local file walk/scan; core container CPU not spiking on graph build for delegated job.

## Issues found and fixed during validation

1. **Large graph JSON (1.7MB) caused result submit HMAC 401** — graph delegation now returns metrics-only payload in v1 worker transport.
2. **Stuck `running` jobs** block capacity — cancel API used between tests.
3. **Worker must stay running** — documented `nohup` / compose run path.

## Logs redaction

- Worker uses `RedactLogLine` for errors; no secrets observed in `/tmp/rd-runner-worker.log`.

## Rollback

After validation, `REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED=false` restored in `.env` and core restarted.
