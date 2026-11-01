# Scheduled full scans (Phase 7)

Repository Detective can periodically scan connected repositories even when no webhook fires. Scheduling uses per-repo settings from the Phase 5 database (Phase 6 UI/API).

> **Naming:** See [NAMING.md](NAMING.md). Internal config still uses `REPOSITORY_DETECTIVE_*` env vars.

## Enable globally

```yaml
database_enabled: true
scheduler_enabled: true
scheduler_poll_interval_seconds: 60
scheduler_max_concurrent_scans: 1
```

Environment variables:

```text
REPOSITORY_DETECTIVE_SCHEDULER_ENABLED=true
REPOSITORY_DETECTIVE_SCHEDULER_POLL_INTERVAL_SECONDS=60
REPOSITORY_DETECTIVE_SCHEDULER_MAX_CONCURRENT_SCANS=1
```

Defaults:

| Setting | Default | Notes |
|---------|---------|-------|
| `scheduler_enabled` | `true` | Forced off when `database_enabled=false` |
| `scheduler_poll_interval_seconds` | `60` | Poll interval for due schedules |
| `scheduler_max_concurrent_scans` | `1` | Max scheduled scans at once |

## Enable per repository

In the operator UI (**Repositories → Settings**) or via `PUT /api/v1/repos/{id}/settings`:

```json
{
  "enabled": true,
  "schedule_enabled": true,
  "schedule_cron": "0 2 * * *"
}
```

Requirements for a repo to be scheduled:

- `connected_repo=true` (onboarded / seen via webhook)
- `repo_settings.enabled=true` (or unset — inherits enabled)
- `repo_settings.schedule_enabled=true`
- Valid 5-field cron in `schedule_cron`

## Cron format

Standard 5-field cron (minute hour day month weekday):

```text
0 2 * * *        # daily at 02:00 UTC
0 */6 * * *      # every 6 hours
30 3 * * 1       # Mondays at 03:30 UTC
```

Invalid cron expressions are rejected on settings save and skipped by the scheduler (logged, never crash).

## Scan behavior

- **Trigger type:** `scheduled`
- **Scope:** full-repository scan via `AnalyzeRepository` (not changed-files)
- **Ref:** repository `default_branch`, or `main` if unset
- **Workspace:** global `workspace_mode` (Phase 8 will apply per-repo workspace policy)
- **Persistence:** scan rows, scanner results, findings, lifecycle events via existing recorder path
- **Issues:** created/updated through existing fingerprint dedup

## Safety / anti-overlap

- No scan if the repo already has a `started` scan row
- No scan if the global analysis limiter is full (skipped, logged)
- No pile-up of missed runs after downtime — baseline is last finished scheduled scan, or now on first enable
- No startup stampede — only repos whose cron is due run, subject to concurrency limits
- Skipped attempts are logged with `trigger_type=scheduled` and reason

## Phase 7 vs Phase 8

Scheduled scans use the repo's full effective settings (Phase 8): scanners, workspace, AI policy, issue policy, and gates — not just schedule fields.

## Observability

Logs include:

- scheduler start/stop
- repo schedule loaded (with next run)
- scheduled scan started / finished / skipped
- invalid cron per repo

Fields: `scan_id`, `repo`, `schedule_cron`, `trigger_type=scheduled`, `reason` (when skipped).

## Runner delegation (Phase 12)

When global runner delegation is enabled and a repo's `runner_policy` allows it, scheduled scans may create a `runner_jobs` row instead of running the in-process analyzer immediately. The scan stays `started` until the runner submits a signed result. See [RUNNERS.md](RUNNERS.md).

## Rollback

1. Set `scheduler_enabled: false` or `REPOSITORY_DETECTIVE_SCHEDULER_ENABLED=false` and reload/restart.
2. Disable per-repo schedules via UI/API (`schedule_enabled: false`).
3. No schema migration required for Phase 7 — existing tables unchanged.
