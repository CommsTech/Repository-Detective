# Nightly skill loop — operator runbook

## What it does

`scripts/nightly-rd-skill-loop.py` runs a **safe overnight calibration learner**:

- Validates tests and smoke checks (fixed harness).
- Ingests learning/calibration evidence from `data/repository-detective.db`.
- Proposes **repo-scoped** calibration candidates.
- Auto-applies **Tier 1** only when `--promote` is set and all gates pass.
- Never auto-applies Tier 3 (global suppressions, HIGH/CRITICAL downgrades, CVE scanners).
- Writes reports under `reports/nightly-rd-evolution/latest/`.

It does **not** modify `analyzers/static.go` or other protected scanner source.

It is **not** the nightly fleet scanner. For repo cron schedules and webhook vs scheduler behavior, see [FLEET_SCANNING.md](../FLEET_SCANNING.md).

## Run once (observe only)

```bash
cd /path/to/repository-detective
python3 scripts/nightly-rd-skill-loop.py --daily-mode --no-promote
```

Read:

- `reports/nightly-rd-evolution/latest/OPERATOR-DIGEST.md`
- `reports/nightly-rd-evolution/latest/promotion_decisions.json`
- `reports/nightly-rd-evolution/latest/full_loop_report.md`

## Run once (allow Tier 1 promotion)

```bash
python3 scripts/nightly-rd-skill-loop.py --daily-mode --promote --max-tier 1
```

`--max-tier` defaults to **1** when `--promote` is set. Tier 2 requires `--max-tier 2` and two consecutive clean runs. Tier 3 never auto-applies.

Requirements:

- Host `go` on PATH **or** Docker available for test execution (see [Go test runner](#go-test-runner))
- `data/repository-detective.db` present
- API key in `.env` (`REPOSITORY_DETECTIVE_API_KEY` or `REPOSITORY_DETECTIVE_API_KEY`) for optional recompute/scans
- Repository Detective healthy on `http://127.0.0.1:8081` for scan triggers

## Go test runner

The nightly loop runs `go test` gates before any promotion. Runner selection is deterministic:

1. **`$GO`** when set and executable
2. **`go` on PATH** (or common install paths)
3. **Docker fallback** — pinned image `golang:1.23-bookworm`

Docker fallback uses stable caches under `state/nightly-rd-evolution/cache/`:

- `go-build` — `GOCACHE`
- `go-mod` — `GOMODCACHE`

The repo is mounted read-only at `/src`; caches are writable. Containers run as the current UID/GID when possible so cache files are not root-owned.

Each run records `test_runner: host-go` or `test_runner: docker` in `full_loop_state.json` and `full_loop_report.md`.

**Smoke check** (no full loop):

```bash
python3 scripts/nightly-rd-skill-loop.py --daily-mode --no-promote --test-runner-smoke
```

**Force host Go** (faster when installed):

```bash
export GO=/path/to/go
python3 scripts/nightly-rd-skill-loop.py --daily-mode --no-promote --test-runner-smoke
```

Host Go is optional but faster. Docker fallback is supported and preferred for reproducibility when host Go is absent.

## Install cron (optional)

```bash
chmod +x scripts/rd-deterministic-daily.sh
crontab -e
```

Example (02:17 daily, Tier 1 only):

```cron
17 2 * * * cd /home/commstech/Repository-Detective && ./scripts/rd-deterministic-daily.sh >> reports/nightly-rd-evolution/cron.log 2>&1
```

**Cron-environment smoke test** (minimal env, same as cron):

```bash
env -i HOME="$HOME" PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
  bash -lc 'cd /home/commstech/Repository-Detective && ./scripts/rd-deterministic-daily.sh'
```

After the first scheduled run:

```bash
cat reports/nightly-rd-evolution/latest/full_loop_state.json
cat reports/nightly-rd-evolution/latest/OPERATOR-DIGEST.md
tail -100 reports/nightly-rd-evolution/cron.log
```

**Cron notes:**

- The cron user must be able to run Docker if the host has no `go` binary (Docker fallback).
- Ensure `state/nightly-rd-evolution/cache/` is writable by the cron user.
- To force host Go in cron, set `GO=/path/to/go` in the wrapper script or crontab environment.

## Disable promotion

Use either:

```bash
python3 scripts/nightly-rd-skill-loop.py --daily-mode --no-promote
```

Or edit `scripts/rd-deterministic-daily.sh` and remove `--promote`.

## Configure scan repos

Default: `commstech/Wifi_Collector` (report-only).

Override:

```bash
export NIGHTLY_RD_SCAN_REPOS="commstech/Wifi_Collector,commstech/PCAP_Analyser"
```

Skip scan triggers (DB ingest only):

```bash
python3 scripts/nightly-rd-skill-loop.py --dry-run-only --no-promote
```

## Read the digest

`OPERATOR-DIGEST.md` lists only:

- Tier 3 recommendations (manual)
- Failed gates
- Rollback events
- Items needing operator action

Routine Tier 1 promotions are recorded in `promotion_decisions.json`, not the digest.

## Rollback

If post-promotion validation fails, the loop:

1. Sets `active=0` on rules inserted this run.
2. Appends an entry to `state/nightly-rd-evolution/rollback_events.json`.

Audit history is append-only — nothing is deleted.

Manual rollback:

```bash
sqlite3 data/repository-detective.db "UPDATE repo_calibration_rules SET active=0 WHERE id IN (...);"
```

## Human approval required for

- Tier 3 candidates (global, HIGH/CRITICAL, secrets, CVE/trivy/grype)
- Analyzer source changes
- Cross-repo learning rules
- Accepting global `calibration_recommendations` via API

See `docs/beta/CALIBRATION_BETA_POLICY.md`.

## Safety invariants

Each run writes `full_loop_state.json` with:

```json
{
  "overall_pass": true,
  "false_auto_apply_count": 0,
  "tier3_auto_apply_count": 0,
  "protected_hashes_unchanged": true,
  "benchmark_tp_regressions": 0,
  "high_critical_auto_downgrades": 0,
  "repo_isolation_verified": true,
  "rollback_audit_append_only": true
}
```

## Troubleshooting

| Symptom | Action |
|---------|--------|
| Lock blocked | Check `state/nightly-rd-evolution/orchestration.lock`; remove only if stale |
| Test gate failed | Fix failing `go test` before enabling `--promote`; run `--test-runner-smoke` to verify runner |
| Docker permission denied | Add cron user to `docker` group or install host Go and set `GO=` |
| No candidates | Need more learning events or findings in DB |
| API recompute skipped | Start Repository Detective or run with `--dry-run-only` |

## Related docs

- [NIGHTLY_SKILL_LOOP_ARCHITECTURE.md](NIGHTLY_SKILL_LOOP_ARCHITECTURE.md)
- [../beta/CALIBRATION_BETA_POLICY.md](../beta/CALIBRATION_BETA_POLICY.md)
