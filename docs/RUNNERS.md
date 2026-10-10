# Gitea Runner Delegation (Phase 12)

Repository Detective — **Inspect. Analyze. Improve.**

Phase 12 adds a **safe runner delegation foundation** so heavy deterministic scan work can run on Gitea Actions runners while the **core service remains authoritative** for policy, persistence, issue creation, and commit status gates.

## Core authority model

```text
Core service decides policy.
Runner executes heavy scan job.
Runner returns structured result.
Core validates result.
Core stores scan / finding / graph data.
Core creates issues and statuses.
```

Runners must **not** create Gitea issues, labels, PRs, commit statuses, or notifications directly. When runner jobs fail or expire, the **core service** emits notifications (if enabled) — see [NOTIFICATIONS.md](NOTIFICATIONS.md).

## Threat model and secrets boundary

### Runners receive

- Repository clone URL, ref, and metadata (no forge token)
- Scan ID and job ID
- Effective scan policy snapshot (scanners, health, graph toggles)
- Resource limits (timeout, max files, max result size)
- Callback base URL
- Runner HMAC shared secret (for signing requests only)

### Runners must NOT receive

- Gitea token
- AI / embedding keys
- Operator API key
- Database DSN
- Webhook secret
- Any other core server secrets

### Trust assumptions

- Result integrity depends on HMAC validation and runner host security
- Compromised runner can submit malicious findings — core still applies policy gates before filing issues
- Shared secret rotation requires updating core config and all runner workflows

## Configuration

```yaml
runner_delegation_enabled: false
runner_mode: core          # core | gitea_actions | auto
runner_shared_secret: ""
runner_job_timeout_seconds: 900
runner_max_concurrent_jobs: 2
runner_result_max_size_mb: 50
runner_artifact_retention_days: 14
runner_callback_base_url: ""   # optional; defaults to public_url or listen host
```

Env: `REPOSITORY_DETECTIVE_RUNNER_*` (legacy `REPOSITORY_DETECTIVE_RUNNER_*` supported).

| Mode | Behavior |
|------|----------|
| `core` | All scans run in-process (default) |
| `gitea_actions` | Delegate when repo `runner_policy` allows |
| `auto` | Delegate when possible; repo `auto` falls back to core on capacity/errors |

Per-repo `runner_policy`: `core`, `gitea_actions`, `auto` — **now enforced** for scheduled and manual full scans.

**Startup rule:** If `runner_delegation_enabled=true` and `runner_mode` is not `core`, `runner_shared_secret` must be set or startup fails.

## Job contract

### JobSpec (core → runner)

Signed JSON including:

- `job_id`, `job_type` (`scan_full_repo`), `scan_id`
- Repository metadata (`forge_type`, `owner`, `name`, `clone_url`, …)
- `ref`, `commit_sha`
- `effective_settings` (policy snapshot — no LLM tasks)
- `limits` (timeout, max repo size/files, max result size)
- `allowed_tasks`: `scanners`, `health`, `graph`
- `forbidden_tasks`: `issue_create`, `status_update`, `pull_request_create`, `secret_access`, `dependency_install`, `repo_script_execution`

### JobResult (runner → core)

Signed JSON including:

- `scanner_results`, `findings`, optional `graph`, `workspace_meta`
- `errors`, `warnings`
- Must not set `forbidden_action`

## API routes

### Operator (API key auth)

| Route | Description |
|-------|-------------|
| `GET /api/v1/runner/jobs` | List runner jobs |
| `GET /api/v1/runner/jobs/:job_id` | Job detail |
| `POST /api/v1/runner/jobs/:job_id/cancel` | Cancel queued/running job |

### Runner worker (HMAC auth)

Headers: `X-Runner-Timestamp`, `X-Runner-Nonce`, `X-Runner-Signature`

| Route | Description |
|-------|-------------|
| `POST /api/v1/runner/jobs/claim` | Claim oldest queued job + spec |
| `GET /api/v1/runner/jobs/:job_id/spec` | Fetch spec for known job |
| `POST /api/v1/runner/jobs/:job_id/result` | Submit signed result |

Runner endpoints do **not** use the operator API key.

## Gitea Actions setup

Repository Detective does **not** auto-create workflow files in user repos (Phase 12).

Example workflow: `.gitea/workflows/repository-detective.yml`

```yaml
name: Repository Detective Scan
on:
  workflow_dispatch:
  schedule:
    - cron: '0 3 * * *'
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Repository Detective runner
        env:
          REPOSITORY_DETECTIVE_CORE_URL: https://detective.example.com
          REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET: ${{ secrets.REPOSITORY_DETECTECTIVE_RUNNER_SECRET }}
          REPOSITORY_DETECTIVE_WORKSPACE: ${{ github.workspace }}
        run: |
          repository-detective-runner \
            --core-url "$REPOSITORY_DETECTIVE_CORE_URL" \
            --runner-secret "$REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET" \
            --workspace "$REPOSITORY_DETECTIVE_WORKSPACE"
```

### Runner image (recommended)

Build the **`repository-detective:runner`** target (see [DOCKER.md](DOCKER.md)):

```bash
docker build --target runner -t repository-detective:runner \
  --build-arg INSTALL_EXTERNAL_TOOLS=true .
```

The image includes `repository-detective-runner` plus pinned scanner binaries (trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov).

Example Gitea Actions step:

```yaml
      - name: Run Repository Detective runner
        image: registry.example.com/repository-detective:runner
        env:
          REPOSITORY_DETECTIVE_CORE_URL: https://detective.example.com
          REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET: ${{ secrets.REPOSITORY_DETECTIVE_RUNNER_SECRET }}
          REPOSITORY_DETECTIVE_WORKSPACE: ${{ github.workspace }}
        run: |
          repository-detective-runner \
            --core-url "$REPOSITORY_DETECTIVE_CORE_URL" \
            --runner-secret "$REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET" \
            --workspace "$REPOSITORY_DETECTIVE_WORKSPACE"
```

Legacy `REPOSITORY_DETECTIVE_*` env names still work via [envcompat](../internal/config/envcompat).

### Runner binary (source build)

```bash
go build -o repository-detective-runner ./cmd/repository-detective-runner
```

The runner:

1. Claims a job (or loads spec by `--job-id`)
2. Runs deterministic scanners, health checks, and graph on the checkout
3. Submits signed `JobResult`

**No LLM** on runners in Phase 12.

### Scanner tools on runner host

If not using the Docker runner image, install the same external scanner binaries as [DOCKER.md](DOCKER.md) (Trivy, Grype, Gitleaks, Semgrep, Go scanners, hadolint, checkov). Missing binaries produce scanner status `binary_missing` in results — core persists them as usual.

Runner jobs merge `effective_settings` from the job spec into scanner config (including Go and IaC scanner toggles, timeouts, and max findings). Profile metadata (`scan_profile`, `profile_source`) is included in the policy snapshot. See [SCAN_PROFILES.md](SCAN_PROFILES.md) — `maintainer_deep` is recommended for scheduled runner delegation.

## Scan integration (Phase 12)

| Trigger | Delegation |
|---------|------------|
| Scheduled full scan | Yes, when policy + global mode allow |
| Manual full repo scan (`POST /api/v1/analyze`) | Yes, when policy allows |
| Webhook push / PR | **Core only** in Phase 12 |
| Pre-install audit | Core only (global config) |

When delegated:

1. Core creates scan row (`started`) and `runner_jobs` row (`queued`)
2. Runner claims job, executes, submits result
3. Core validates HMAC, size, scan ID, nonce
4. Core persists scanner results, findings, graph
5. Core runs issue creation and status logic

## Database (migration v6)

Tables: `runner_jobs`, `runner_artifacts`, `runner_nonces`.

## Known limitations

- Gitea Actions only — no GitHub/GitLab runners
- Scheduled/manual full scans first; webhooks stay on core
- No automatic workflow provisioning in repos
- No remediation / auto-fix jobs
- No LLM on runners
- Scanner binaries must exist on runner host
- Result trust depends on shared secret + runner host security

## Rollback

1. Set `runner_delegation_enabled: false` or `runner_mode: core`
2. Set repo `runner_policy: core`
3. Restart core — in-flight runner jobs may expire; scans stay `started` until manually cleared or jobs complete
4. Migration v6 tables are additive; no downgrade required for emergency rollback

## Smoke test procedure

Use this checklist after enabling runner delegation in a test environment.

### 1. Build the runner binary

```bash
go build -o repository-detective-runner ./cmd/repository-detective-runner
```

### 2. Configure core

```yaml
runner_delegation_enabled: true
runner_mode: gitea_actions   # or auto
runner_shared_secret: "<generate-a-long-random-secret>"
runner_job_timeout_seconds: 900
runner_max_concurrent_jobs: 2
```

Env equivalents: `REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED`, `REPOSITORY_DETECTIVE_RUNNER_MODE`, `REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET`, etc.

Restart core after changing secrets or delegation flags.

### 3. Configure one test repository

In repo settings (UI or API):

```yaml
runner_policy: gitea_actions   # or auto for fallback-to-core behavior
enabled: true
```

Ensure scheduled or manual full scans are allowed for that repo.

### 4. Trigger a delegated scan

Either:

- `POST /api/v1/analyze` with `type: repository` for the test repo, or
- Wait for / force a scheduled full scan when `schedule_enabled: true`

Confirm a `runner_jobs` row and scan row are created (`status: started` on scan, `queued` on job).

### 5. Run the worker

On the runner host (or Gitea Actions workflow):

```bash
repository-detective-runner \
  --core-url "https://detective.example.com" \
  --runner-secret "$REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET" \
  --workspace "/path/to/checkout"
```

### 6. Verify job lifecycle

| Stage | Where to check |
|-------|----------------|
| `queued` | `GET /api/v1/runner/jobs` (operator API key) |
| `claimed` / `running` | Job detail after worker claims |
| `completed` or `failed` | Job detail + scan detail UI |

Dashboard shows runner job counts by status.

### 7. Verify core ingestion

After `completed`:

- **Scanner results** — scan detail shows scanner rows from runner payload
- **Findings** — persisted in DB / visible on scan and repo pages
- **Graph** — present when policy enabled and runner returned graph JSON
- **Issues / statuses** — core creates Gitea issues and commit statuses (runner does not)

Failed runner jobs mark the scan failed via core ingestion; check scan `error` and job `error`.

### 8. Verify secrets boundary

Inspect `JobSpec` (`GET /api/v1/runner/jobs/:job_id` or worker logs):

- Present: clone URL, ref, policy snapshot, limits, callback URL
- **Absent:** Gitea token, operator API key, AI keys, DB DSN, webhook secret

Runner HTTP routes authenticate with HMAC only — not the operator API key.

## Operational safety

### Runner never returns

- Each job has `expires_at = created_at + runner_job_timeout_seconds` (default 900s).
- `ExpireStaleRunnerJobs` runs on claim and periodically marks stale `queued` / `dispatched` / `running` jobs as **`expired`**.
- Expired jobs reject further result submission (`410`-class errors).
- The associated scan may remain **`started`** until the operator clears it or a result/failure is ingested — monitor stuck scans via UI/API.

**Mitigation:** Cancel via `POST /api/v1/runner/jobs/:job_id/cancel`, disable delegation, or re-run scan on core.

### Job expiration

| Setting | Default | Effect |
|---------|---------|--------|
| `runner_job_timeout_seconds` | 900 | Job `expires_at`; stale jobs → `expired` |

### Failed jobs in UI/API

| Job status | Meaning |
|------------|---------|
| `failed` | Runner returned errors or core rejected processing |
| `expired` | No result before timeout |
| `cancelled` | Operator cancelled queued/running job |

List: `GET /api/v1/runner/jobs`. Scan detail links to the runner job when present.

### `runner_mode=auto` fallback

When repo `runner_policy` is **`auto`** and delegation fails (capacity, DB error, missing secret at dispatch time):

- Core logs: `Runner delegation unavailable, falling back to core scan`
- Scan runs in-process on core

When repo policy is **`gitea_actions`**, delegation failure **does not** fall back — the scan fails with the dispatch error.

### `runner_mode=gitea_actions`

- Delegates when repo policy allows (`gitea_actions` or `auto`).
- Does **not** silently fall back for `gitea_actions` repo policy.
- Startup **fails closed** if `runner_delegation_enabled=true`, mode is not `core`, and `runner_shared_secret` is empty.

### Rotating `runner_shared_secret`

1. Generate a new secret.
2. Update core config and restart core.
3. Update all runner workflows / hosts with the new secret **before** or **during** a brief overlap window.
4. In-flight jobs signed with the old secret will fail HMAC validation after core restart.

There is no dual-secret grace period in Phase 12 — plan a maintenance window.

### Disabling delegation

| Scope | Action |
|-------|--------|
| Global | `runner_delegation_enabled: false` or `runner_mode: core` + restart |
| Per repo | `runner_policy: core` in repo settings |

Queued jobs may remain until expired or cancelled; new scans use core immediately after config reload.

## Result validation (core)

Core rejects runner results when:

| Check | Error |
|-------|-------|
| Invalid HMAC signature | **or** missing/invalid timestamp | `401` runner auth failed |
| Nonce replay | `401` runner nonce rejected |
| Unknown job ID | `401` / unknown job |
| Expired job | `410` job expired |
| Cancelled / already completed job | `400` job already finalized |
| `scan_id` mismatch | `400` scan_id mismatch |
| `forbidden_action` set in payload | `400` forbidden action |
| Result JSON over `runner_result_max_size_mb` | `413` size limit |

Runners cannot call issue or status APIs — those tasks are listed in `forbidden_tasks` on every `JobSpec`.

Tests: `runner/runner_test.go`, `runner/receiver_test.go`, `store/runner_test.go`.
