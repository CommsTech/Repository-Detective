# Post-overnight fleet checkpoint

Generated: 2026-07-12

## Verdict: **PASS / monitor** (with one learner gate fix)

### Fleet scanning

| Metric | Before schedules | After overnight |
|--------|------------------|-----------------|
| Nightly schedule on | 0/40 | **40/40** |
| Stale scans (>24h) | 33 | **0** |
| Mapped forge issues | 22 | **238** |
| Scheduler globally enabled | yes | yes |

### Issue filing

| Bucket | Count | Action |
|--------|-------|--------|
| `unknown` | **0** | Closed |
| True canary-eligible | **0** | Do not `--apply` |
| Fixture / docs FP (former unknown) | 19 | Operator triage / allowlist later |
| `report_only` | 33 | Expected residual |

### Nightly calibration learner (02:17)

Last cron run `20260712T081702` reported **Pass: False** because `TestReposControlPage` expected the old reconciliation hint string, and fleet-health SQL failed on NULL `trigger_type` for never-scanned edge cases during tests.

**Fixes in this checkpoint:**

1. Restore reconciliation hint text on `/ui/repos` (keep clearer webhook/sched copy).
2. `COALESCE(ls.trigger_type, '')` in fleet audit SQL.

Re-run observe-only after the gate fix restored **Pass: True** (`20260712T151124`, consecutive successful runs: 1).

**Ops note (2026-07-12):** Docker/systemd became unresponsive (cgroup unit timeouts), which stopped the `repository-detective` container. Service was restored by running the **statically linked** `bin/repository-detective` on the host against the same `data/` and `.env`. When Docker recovers, recreate the container from `rc-a7f3fc7`/`all-in-one` (or a newer image) and stop the host process.

### Still blocked

- Broad issue filing
- Tier 2 promotion (`--max-tier 2`)
- Filing canary (no safe repo)

### Paths remain separated

| Path | State |
|------|-------|
| Tier 1 calibration learner | Cron 02:17 — gate fixed; observe-only green again |
| Fleet repo scans | 03:30–04:25 UTC — healthy (40/40, 0 stale) |
| Webhooks | Separate |
| Issue filing | Guarded / canary-only |
