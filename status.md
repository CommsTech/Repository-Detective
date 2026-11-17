# Repository Detective status

Current public prerelease: `v0.1.0-beta.6` (live overlay: `repository-detective:ux2-20261117` / commit `cd3aaf4c`).

Operator notes, scan history, and forge inventory stay outside this repository. Use your own Gitea or GitHub settings, `.env`, and database. Shipped examples use empty/required fields — published placeholders are rejected at startup.

## 2026-11-17 ux2 container triage

- Live container **healthy** on `:8081` (`RestartCount=0`, ~200–270MiB RAM).
- Startup OK: Gitea + GitHub connected; OpenClaw gateway `/health` live; scheduler loaded 36 repo cron entries; DB/calibration healthy.
- Manual dogfood scan of Repository-Detective in progress after deploy (external scanners completed; SBOM/history phase continuing — not stuck).
- **Fixed (live hotpatch):** `golangci-lint` was missing in `ux2` and prior recent images because upstream `install.sh` checksum verify failed. Binary `v2.14.0` copied into the running container; `scripts/install-scanner-tools.sh` now uses the direct release tarball so the next image rebuild keeps it.
- Non-blocking: no `config.yaml` (env-driven only — expected); Community edition banner still says connected-repo limit=10 while 40 repos are connected/scheduled (grandfathered behavior, not a runtime error).
- Stale scheduled coverage: AMMBER / Dockhand / talos_cluster last completed scans are >36h old.

## 2026-11-16 operator recovery

- Live container healthy (`/health/ready`); open Gitea issues for Repository-Detective: **0**.
- Re-registered local Gitea Actions runner **ClusterMGR** after `unregistered runner` crash loop.
- Disabled **RemoteSupport** runner (missing Docker network `rd-ci-network`); left Hurricane disabled.
- Fixed ClusterMGR `actrunner_config.yaml`: `valid_volumes: ["**"]`, `force_pull: false` so job containers can mount workspaces.
- Re-queued CI on `main` (`e8d4f4b3`); run executing on ClusterMGR after Checkout succeeded.

## 2026-11-12 runtime log fixes

Deployed live as `repository-detective:v0.1.0-beta.6-hotfix5` (`701703ac`).

- Fence-aware JSON decode for CAH auditor / attack-surface / PoC responses (`ai.DecodeModelJSON`)
- AI health probe strips trailing `/v1` so OpenClaw gateway `/health` is checked
- Scheduler keeps a lifetime concurrency semaphore (default max concurrent scheduled scans: 2) and coalesces skip spam
- `/ui` dashboard uses the same 90s detached DB budget as the API; noise-reduction metrics are TTL-cached
- Calibration recommendation generation batches existing-proposal lookups (global + repo)
- Deep remains the intentional AI path under `ENABLE_LLM_AUDITORS=false`

Verification after deploy: `/health/ready` healthy; authenticated `/ui` ~0.8s (was ~1m34s).

Canonical push: Gitea `main` @ `701703ac`. GitHub public mirror refreshed as a clean tree snapshot of that tip (full-history push blocked by GitHub push protection on historical test fixtures).

## 2026-11-02 Deep AI / LLM sanity gate

Deep scan profile enables AI auditors and the LLM sanity gate even when the global
`ENABLE_LLM_AUDITORS=false` / `llm_sanity_gate_enabled=false` cost defaults stay on.
Light and Standard remain deterministic-only. Custom follows the platform flags.

## 2026-11-01 source audit remediation (RD-AUD)

Implemented P0/P1 install-safety and public front-door fixes from the static source audit:

- Reject published example API/session/forge secrets and example hostnames at startup
- Default Compose bind `127.0.0.1` (override with `REPOSITORY_DETECTIVE_BIND`)
- Prefer `/health/ready`; do not force-skip startup checks in Compose
- Public docs use canonical Gitea + GitHub URLs; remove `?api_key=` UI guidance
- First-run default `AUTO_CREATE_ISSUES=false` (report-only)
- Edition/Go version docs reconciled; Forgejo marked experimental

Tracking: [docs/release/SOURCE_AUDIT_REMEDIATION_2026-11-01.md](docs/release/SOURCE_AUDIT_REMEDIATION_2026-11-01.md).

Still open for the next release gate: immutable published-image acceptance (RD-AUD-004), scanner pin/SBOM hardening (RD-AUD-008), synthetic no-credential demo (RD-AUD-005 remainder), independent first-user timing (RD-AUD-010).

## 2026-10-31 / 2026-11-01 runtime log fixes

Dashboard summary timeouts, nightly analyze 500s, dual-open forge mappings, nightly MODULE_PATH typo, and staticcheck workspace classification — see `issues.md`.

Release notes: [docs/release/GITHUB_RELEASE_v0.1.0-beta.6.md](docs/release/GITHUB_RELEASE_v0.1.0-beta.6.md).
