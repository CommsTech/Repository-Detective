# Native runner soak test report

Recorded: 2026-06-09 (finalized 2026-06-09)  
Core revision: `8d5da54` → guardrail follow-up on `main`  
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
- In-memory registry drops workers after heartbeat max-age (offline when not seen).

## Failure recovery

| Step | Result |
|------|--------|
| Enqueue graph while worker running | job entered `running` |
| `pkill` worker mid-job | worker stopped |
| Mid-test graph submit | `context canceled` (expected) |
| Recovery job `rj-285aeeaccd178f88` | stayed `running` while worker dead |
| Stuck job cleanup | marked `expired` during stabilization pass |
| Worker restart | new worker re-registered; subsequent jobs completed |

## Background task outcomes (final)

| Task | Outcome |
|------|---------|
| `INSTALL_EXTERNAL_TOOLS=true` Docker build | stopped with **exit 143** after ~22 minutes |
| Faster rebuild without external tools | **succeeded** |
| `./scripts/docker-build-verify.sh` | **passed** afterward |
| graph / SBOM / remediation_verify before failure test | **completed** |
| Worker killed during failure-recovery test | intentional |
| `context canceled` on mid-test graph | **expected** failure path |
| `rj-285aeeaccd178f88` while worker dead | stayed `running` until cleanup |
| Stuck job cleanup | `expired` with audit error message (not deleted) |
| Live app after rollback | **healthy** |
| Runner delegation | **disabled** |
| Worker process | **none running** |

## Docker build notes

- Do **not** use `INSTALL_EXTERNAL_TOOLS=true` on the critical release path unless scanner image time budget (~20+ min) is acceptable.
- Faster all-in-one build (no external tools) is sufficient for runner delegation validation.

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

## Operational guardrails

| Capability | Status |
|------------|--------|
| Job `expires_at` lease | yes (default 900s) |
| `ExpireStaleRunnerJobs` on claim | yes |
| `ExpireStaleRunnerJobs` on core startup | **added** (stabilization pass) |
| Worker heartbeat registry | yes (in-memory, max-age filter) |
| Cancel API | `POST /api/v1/runner/jobs/:id/cancel` |
| Silent job deletion | **no** — expired/failed jobs retained |

## Conclusion

**Native runner delegation is beta-viable for controlled test windows.**

- Keep **disabled by default**.
- Enable only for short soak/validation windows; disable and stop worker after test.
- Do not use `INSTALL_EXTERNAL_TOOLS=true` builds in the critical release path unless scanner image time budget is acceptable.
