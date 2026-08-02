#!/usr/bin/env python3
"""Audit Repository Detective fleet scanning, scheduling, and issue filing."""

from __future__ import annotations

import argparse
import json
import os
import sqlite3
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DB = ROOT / "data/repository-detective.db"
DEFAULT_REPORT = ROOT / "docs/dogfood-reports/fleet-scanning-and-filing-audit.md"
STALE_HOURS = 24


def staggered_cron(repository_id: int) -> str:
    minute = 30 + (repository_id % 12) * 5
    hour = 3
    if minute >= 60:
        hour += 1
        minute -= 60
    return f"{minute} {hour} * * *"


def load_env_scheduler_hint() -> bool | None:
    env_path = ROOT / ".env"
    if not env_path.exists():
        return None
    return None


def query_fleet(conn: sqlite3.Connection) -> list[dict]:
    rows = conn.execute(
        """
        SELECT r.id, r.full_name, r.forge_type, r.connected_repo, r.default_branch,
            rs.enabled, rs.schedule_enabled, rs.schedule_cron, rs.issue_policy, rs.policy_level,
            ls.started_at, ls.trigger_type, COALESCE(ls.commit_sha, ''),
            CASE WHEN json_extract(ls.summary_json, '$.dry_run_report_only') = true THEN 1 ELSE 0 END,
            lw.started_at,
            COALESCE(fc.open_findings, 0),
            COALESCE(fc.no_mapped, 0),
            COALESCE(fc.mapped_issues, 0)
        FROM repositories r
        LEFT JOIN repo_settings rs ON rs.repository_id = r.id
        LEFT JOIN (
            SELECT s.repository_id, s.started_at, s.trigger_type, s.commit_sha, s.summary_json
            FROM scans s
            INNER JOIN (
                SELECT repository_id, MAX(started_at) AS max_started FROM scans GROUP BY repository_id
            ) m ON s.repository_id = m.repository_id AND s.started_at = m.max_started
        ) ls ON ls.repository_id = r.id
        LEFT JOIN (
            SELECT repository_id, MAX(started_at) AS started_at
            FROM scans WHERE trigger_type = 'push' GROUP BY repository_id
        ) lw ON lw.repository_id = r.id
        LEFT JOIN (
            SELECT f.repository_id,
                COUNT(1) AS open_findings,
                SUM(CASE WHEN NOT EXISTS (
                    SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open'
                ) THEN 1 ELSE 0 END) AS no_mapped,
                SUM(CASE WHEN EXISTS (
                    SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open'
                ) THEN 1 ELSE 0 END) AS mapped_issues
            FROM findings f WHERE f.status = 'open' GROUP BY f.repository_id
        ) fc ON fc.repository_id = r.id
        ORDER BY r.full_name
        """
    ).fetchall()
    out = []
    now = datetime.now(timezone.utc)
    for row in rows:
        (
            repo_id,
            full_name,
            forge_type,
            connected,
            default_branch,
            enabled,
            schedule_enabled,
            schedule_cron,
            issue_policy,
            policy_level,
            last_scan_at,
            last_trigger,
            last_commit,
            dry_run,
            last_webhook_at,
            open_findings,
            no_mapped,
            mapped_issues,
        ) = row
        scan_on = enabled is None or enabled == 1
        sched_on = schedule_enabled == 1
        filing_on = not (issue_policy == "off" or policy_level == "monitor_only")
        stale = True
        stale_hours = None
        if last_scan_at:
            try:
                dt = datetime.fromisoformat(last_scan_at.replace("Z", "+00:00"))
                if dt.tzinfo is None:
                    dt = dt.replace(tzinfo=timezone.utc)
                stale_hours = (now - dt).total_seconds() / 3600
                stale = stale_hours > STALE_HOURS
            except ValueError:
                pass
        schedule_eligible = bool(
            connected == 1 and scan_on and sched_on and schedule_cron and str(schedule_cron).strip()
        )
        skip_reason = ""
        if connected != 1:
            skip_reason = "not_connected"
        elif not scan_on:
            skip_reason = "scan_disabled"
        elif not sched_on:
            skip_reason = "schedule_disabled"
        elif not schedule_cron or not str(schedule_cron).strip():
            skip_reason = "missing_schedule_cron"
        out.append(
            {
                "repository_id": repo_id,
                "full_name": full_name,
                "forge_type": forge_type,
                "scan_enabled": scan_on,
                "schedule_enabled": sched_on,
                "schedule_cron": schedule_cron or "",
                "filing_enabled": filing_on,
                "report_only_enforced": bool(dry_run) or not filing_on,
                "last_scan_at": last_scan_at,
                "last_scan_trigger": last_trigger or "",
                "last_scan_commit": last_commit or "",
                "last_webhook_at": last_webhook_at,
                "stale": stale,
                "stale_hours": round(stale_hours, 1) if stale_hours is not None else None,
                "open_findings": open_findings,
                "no_mapped_findings": no_mapped,
                "mapped_forge_issues": mapped_issues,
                "schedule_eligible": schedule_eligible,
                "schedule_skip_reason": skip_reason,
                "default_branch": default_branch or "main",
            }
        )
    return out


def summarize(rows: list[dict]) -> dict:
    scan_on = [r for r in rows if r["scan_enabled"]]
    sched_on = [r for r in rows if r["schedule_enabled"]]
    sched_off_scan_on = [r for r in rows if r["scan_enabled"] and not r["schedule_enabled"]]
    stale = [r for r in rows if r["stale"] and r["scan_enabled"]]
    eligible = [r for r in rows if r["schedule_eligible"]]
    return {
        "total_tracked": len(rows),
        "scan_enabled": len(scan_on),
        "schedule_enabled": len(sched_on),
        "schedule_disabled_scan_enabled": len(sched_off_scan_on),
        "schedule_eligible": len(eligible),
        "stale_scan_enabled": len(stale),
        "no_mapped_findings_total": sum(r["no_mapped_findings"] for r in rows),
        "mapped_forge_issues_total": sum(r["mapped_forge_issues"] for r in rows),
    }


def write_report(path: Path, rows: list[dict], summary: dict, mode: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    lines = [
        "# Fleet scanning and filing audit",
        "",
        f"Generated: {datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')}",
        f"Mode: **{mode}**",
        "",
        "## Fleet summary",
        "",
        f"- Tracked repositories: **{summary['total_tracked']}**",
        f"- Scan enabled (webhook/manual): **{summary['scan_enabled']}**",
        f"- Nightly schedule on: **{summary['schedule_enabled']}**",
        f"- Nightly schedule off (scan enabled): **{summary['schedule_disabled_scan_enabled']}**",
        f"- Scheduler-eligible: **{summary['schedule_eligible']}**",
        f"- Stale scans (&gt;24h, scan enabled): **{summary['stale_scan_enabled']}**",
        f"- Open findings without mapped forge issue: **{summary['no_mapped_findings_total']}**",
        f"- Mapped open forge issues: **{summary['mapped_forge_issues_total']}**",
        "",
        "> The nightly calibration learner (`rd-deterministic-daily.sh` at 02:17) is **not** a full-fleet scanner.",
        "",
        "## Per-repository",
        "",
        "| Repo | Scan | Sched | Filing | Last scan | Stale | Unmapped findings | Mapped issues | Skip reason |",
        "|------|------|-------|--------|-----------|-------|-------------------|---------------|-------------|",
    ]
    for r in rows:
        lines.append(
            f"| {r['full_name']} | {'on' if r['scan_enabled'] else 'off'} | "
            f"{'on' if r['schedule_enabled'] else 'off'} | "
            f"{'on' if r['filing_enabled'] else 'off'} | "
            f"{r['last_scan_at'] or 'never'} | "
            f"{'yes' if r['stale'] else 'no'} | "
            f"{r['no_mapped_findings']} | {r['mapped_forge_issues']} | "
            f"{r['schedule_skip_reason'] or '—'} |"
        )
    lines.append("")
    path.write_text("\n".join(lines) + "\n")


def apply_enable_nightly(conn: sqlite3.Connection, rows: list[dict], only_scan_enabled: bool) -> list[dict]:
    changed = []
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    for r in rows:
        if only_scan_enabled and not r["scan_enabled"]:
            continue
        if r["schedule_enabled"] and r["schedule_cron"]:
            continue
        cron = staggered_cron(r["repository_id"])
        repo_id = r["repository_id"]
        existing = conn.execute(
            "SELECT 1 FROM repo_settings WHERE repository_id = ?", (repo_id,)
        ).fetchone()
        if existing:
            conn.execute(
                "UPDATE repo_settings SET schedule_enabled = 1, schedule_cron = ?, updated_at = ? WHERE repository_id = ?",
                (cron, now, repo_id),
            )
        else:
            conn.execute(
                """
                INSERT INTO repo_settings (repository_id, enabled, schedule_enabled, schedule_cron, updated_at)
                VALUES (?, 1, 1, ?, ?)
                """,
                (repo_id, cron, now),
            )
        changed.append({**r, "proposed_cron": cron})
    conn.commit()
    return changed


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Fleet scheduler and filing audit")
    p.add_argument("--db-path", type=Path, default=DEFAULT_DB)
    p.add_argument("--report", type=Path, default=DEFAULT_REPORT)
    p.add_argument("--dry-run", action="store_true", help="Report only; default when no apply flag")
    p.add_argument("--enable-nightly", action="store_true", help="Enable staggered nightly schedules")
    p.add_argument("--only-scan-enabled", action="store_true", help="With --enable-nightly, skip scan-disabled repos")
    p.add_argument("--json", action="store_true", help="Print JSON summary to stdout")
    args = p.parse_args(argv)

    if not args.db_path.exists():
        print(f"database not found: {args.db_path}", file=sys.stderr)
        return 2

    conn = sqlite3.connect(args.db_path)
    try:
        rows = query_fleet(conn)
        summary = summarize(rows)
        mode = "dry-run"
        if args.enable_nightly:
            if args.dry_run:
                mode = "dry-run (would enable nightly)"
                candidates = [
                    r
                    for r in rows
                    if (not args.only_scan_enabled or r["scan_enabled"])
                    and not (r["schedule_enabled"] and r["schedule_cron"])
                ]
                for r in candidates:
                    r["proposed_cron"] = staggered_cron(r["repository_id"])
                print(json.dumps({"would_enable": len(candidates), "repos": candidates[:5]}, indent=2))
            else:
                changed = apply_enable_nightly(conn, rows, args.only_scan_enabled)
                mode = f"applied ({len(changed)} repos)"
                rows = query_fleet(conn)
                summary = summarize(rows)
        else:
            args.dry_run = True

        write_report(args.report, rows, summary, mode)
        if args.json:
            print(json.dumps({"summary": summary, "rows": rows}, indent=2))
        else:
            print(json.dumps(summary, indent=2))
            print(f"Report: {args.report}")
        return 0
    finally:
        conn.close()


if __name__ == "__main__":
    raise SystemExit(main())
