# Repository Detective status

Current public prerelease: `v0.1.0-beta.6` (live overlay: `v0.1.0-beta.6-hotfix5` — Deep AI + runtime log fixes).

Operator notes, scan history, and forge inventory stay outside this repository. Use your own Gitea or GitHub settings, `.env`, and database. Shipped examples use empty/required fields — published placeholders are rejected at startup.

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
