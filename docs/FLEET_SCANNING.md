# Fleet scanning vs calibration learner

Repository Detective has **three separate scan paths**. They must not be confused.

## 1. Webhook scans (on every commit)

- **Trigger:** Gitea `POST /webhook` on push/PR
- **Requires:** Webhook registered on the repo + repo `enabled` (scan on)
- **Does not require:** `schedule_enabled`
- **Issue filing:** Depends on repo policy and whether the scan used `report_only_dry_run`

## 2. Nightly fleet scans (in-process scheduler)

- **Trigger:** `orch/scheduler` polls `repo_settings` where `schedule_enabled = 1` and `schedule_cron` is set
- **UI label:** `Nightly sched on/off` on `/ui/repos`
- **Default:** Schedule is **off** for new repos (`ScheduleEnabled: false` globally)
- **Not the same as:** `scripts/rd-deterministic-daily.sh` (calibration learner)

## 3. Nightly calibration learner (skill loop)

- **Script:** `scripts/rd-deterministic-daily.sh` (cron ~02:17)
- **Purpose:** Safe Tier 1 repo-scoped calibration from learning data
- **Does not:** Scan all 40 repos nightly
- **Default scan target:** `NIGHTLY_RD_SCAN_REPOS` (report-only when `--daily-mode`)

## Issue filing vs “unmapped findings”

Open findings without a mapped Gitea issue are normal when:

- Report-only / monitor-only policy
- Filing disabled (`issue_policy=off`)
- Latest scan used `dry_run_report_only`
- Severity/confidence gates or backlog control blocked filing

The fleet control center shows **unmapped** (not “no issue”) for open findings without an open `external_issues` row.

## Audit before enabling fleet schedules

```bash
python3 scripts/fleet-scheduler-audit.py --dry-run
cat docs/dogfood-reports/fleet-scanning-and-filing-audit.md
```

## Enable nightly fleet scans (operator command)

Staggered window **03:30–04:25 UTC** (after 02:00 backup and 02:17 calibration learner):

```bash
python3 scripts/fleet-scheduler-audit.py --enable-nightly --only-scan-enabled
```

Review the dry-run report first. Do not enable until stale repos and filing policy are understood.

## Reconcile issue filing

Per-repo: open repo detail → reconciliation panel, or:

```bash
curl -H "X-Repository-Detective-API-Key: $KEY" \
  http://127.0.0.1:8081/api/v1/repos/ID/reconcile-issues/preview
```

**Issue filing is separate from scan scheduling.** Enabling nightly schedules does not file thousands of backlog findings automatically.

### Unmapped reason audit (before any filing canary)

```bash
python3 scripts/issue-filing-reconcile-audit.py --summary
python3 scripts/issue-filing-reconcile-audit.py --unknown-details
```

`--unknown-details` writes `docs/dogfood-reports/unknown-unmapped-finding-audit.md` and must show `unknown_count: 0` (or an explained remainder) before any `--apply`.

### One-repo filing canary

```bash
python3 scripts/issue-filing-reconcile-audit.py --repo commstech/Infrastructure_as_Code --dry-run --limit 10
python3 scripts/issue-filing-reconcile-audit.py --repo commstech/Infrastructure_as_Code --apply --limit 10
```

Apply requires `--repo`, `--apply`, and `--limit` (default 10). It never bulk-files all repos.

## Related docs

- [NIGHTLY_SKILL_LOOP_RUNBOOK.md](learning/NIGHTLY_SKILL_LOOP_RUNBOOK.md) — calibration learner
- [SCHEDULER.md](SCHEDULER.md) — in-process scheduler
