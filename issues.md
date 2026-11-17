# Repository Detective issue notes

Product issues belong on the forge you use to track this repository.

## Operator log triage (2026-11-17)

Live `repository-detective:ux2-20261117` (`cd3aaf4c`) review ~7–15 minutes after restart:

1. **App runtime** — healthy on `:8081`; Gitea + GitHub OK; OpenClaw gateway live; **0** `error`/`fatal` log lines since start (2 warnings only).
2. **`[SCANNER:golangci-lint] binary not found`** — real gap, also present on `hotfix5` / `cve-20261116` images. Root cause: upstream `install.sh` checksum verify fails (`hash_sha256_verify` mismatch), and the build treated that as optional skip. **Mitigated live** by installing `v2.14.0` into the running container; **durable fix** in `scripts/install-scanner-tools.sh` (direct GitHub release tarball + louder failure log). Needs image rebuild to persist across recreate.
3. **`No config file found`** — expected; `/app/config` only has `.example` / `.bak` files. Runtime is env-driven from Compose/`.env`.
4. **Active scan silence after scanners** — CAH continued into SBOM (`cyclonedx-gomod` / `grype sbom:` / `gitleaks detect` history). Status stayed `started` with empty summary while that phase ran; not a hang.
5. **Edition banner vs inventory** — logs say Community connected-repo limit=10 while API lists ~40 connected repos with schedules. No hard failure observed; treat as license/edition messaging drift.
6. **Stale repos** — AMMBER, Dockhand, talos_cluster last completed scans are weeks old (worth a manual/schedule review, not a container crash).

## Operator log triage (2026-11-16)

Live `repository-detective:v0.1.0-beta.6-hotfix5` review:

1. **App runtime** — healthy on `:8081`; Gitea + GitHub connections OK; no `error`/`fatal` in the last 7 days. Scheduler backlog under `max_concurrent=2` is expected load, not a failure.
2. **Open forge issues** — `commstech/Repository-Detective` open issues: **0**.
3. **Local `act_runner` (ClusterMGR) crash loop** — `unregistered runner` after Gitea dropped runner id 9. Re-registered as ClusterMGR (id 13); service online again.
4. **CI cancellations on RemoteSupport** — `network rd-ci-network not found`. RemoteSupport disabled in Gitea admin until its Docker network is repaired (host `192.168.255.20`; local `commstech` user lacks docker/sudo for the fix). Created `rd-ci-network` on ClusterMGR as a local safeguard.
5. **ClusterMGR job hang** — `container.valid_volumes: []` blocked workspace mounts; job network created with no containers. Set `valid_volumes: ["**"]` and `force_pull: false` in `/usr/local/bin/actrunner_config.yaml`, restarted runner. CI job containers start again.

## Operator log triage (2026-11-12)

Live `repository-detective:v0.1.0-beta.6-hotfix4` logs showed:

1. **Auditor JSON parse failures** — LLM returns markdown fences / prose; `ai.Client.RunAuditor` did raw `json.Unmarshal` (139 warnings). Advisory path already had `ExtractJSONObject` / `RepairRelaxedJSON`.
2. **AI health probe false negatives** — probe hit `{ai_base_url}/v1/health` instead of gateway origin `/health`; then auditors hit context deadlines.
3. **Scheduled scan skip spam** — `max concurrent scheduled scans reached` (~490 in window) because the concurrency semaphore was recreated each tick with default `MaxConcurrent=1`.
4. **`GET /ui` multi-minute stalls** — dashboard sequentially hit heavy SQLite rollups (incl. `finding_instances` COUNT) without the API's detached 90s budget / stale cache path.
5. **Calibration recompute ~1m12s** — full-table rule stats + per-repo recommendation generation on a ~2.3GB DB.

## Deep AI / sanity gate (2026-11-02)

Deep profile scans enable AI auditors and the LLM sanity gate even when global
`ENABLE_LLM_AUDITORS=false` / `llm_sanity_gate_enabled=false`. Light and Standard
stay deterministic. Live overlay: `v0.1.0-beta.6-hotfix4`.

## Source audit (2026-11-01)

External static audit RD-AUD-001…010. Remediation log:
[docs/release/SOURCE_AUDIT_REMEDIATION_2026-11-01.md](docs/release/SOURCE_AUDIT_REMEDIATION_2026-11-01.md).

**Fixed now:** unsafe example credentials, all-interface default publish, misleading
`/health`-only install checks, placeholder public URLs, `?api_key=` quickstart,
report-only first-run default, edition/limit doc contradictions.

**Still open:** published beta.6 digest acceptance receipt; reproducible scanner
pins (`@latest` / install-script master); no-credential synthetic demo path;
independent first-user timing study.

## Operator log triage (2026-10-31)

Running `repository-detective:v0.1.0-beta.6` showed:

1. **Dashboard summary 500s (~30s)** during the nightly smoke window — SQLite
   contention canceled client contexts mid-rollup. Fixed with indexes, scan-first
   rollup SQL, stale-while-revalidate cache, and a detached 90s API rebuild budget.
2. **Daily `POST /api/v1/analyze` 500s** from the nightly skill loop defaulting to
   `your-org/your-repo`, which hit the Community connected-repo cap. Fixed by
   skipping placeholders and returning HTTP 403 (`repo_limit_exceeded`) for the
   license gate instead of 500.
3. **Dual open forge mappings** for one fingerprint (batch persist path). Fixed so
   only the lowest issue number stays open locally.

## Operator follow-up (2026-11-01)

1. Nightly `MODULE_PATH` typo (`git.commstech.org`) emptied package enumeration.
2. Calibration UNIQUE inserts mismatched the active unique index (path_pattern).
3. Healed 57 legacy dual-open forge mappings.
4. Staticcheck incomplete-workspace diagnostics → `scanner_unavailable`.

This file does not list operator repositories, scan identifiers, or deployment inventory.
Those stay in your private database and environment.
