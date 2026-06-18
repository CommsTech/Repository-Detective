#!/usr/bin/env python3
"""Safe issue-filing reconciliation audit and one-repo canary."""

from __future__ import annotations

import argparse
import json
import os
import sqlite3
import sys
import time
import urllib.error
import urllib.request
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DB = ROOT / "data/bugbot.db"
REPORT_DIR = ROOT / "docs/dogfood-reports"
DEFAULT_LIMIT = 10

SEVERITY_RANK = {"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}
SEVERITY_GATES = {"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}


def load_env() -> None:
    env_path = ROOT / ".env"
    if not env_path.exists():
        return
    for line in env_path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        k, v = k.strip(), v.strip().strip('"').strip("'")
        if k and k not in os.environ:
            os.environ[k] = v


def api_config() -> tuple[str, str]:
    key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("BUGBOT_API_KEY", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_PUBLIC_URL") or os.environ.get("BUGBOT_PUBLIC_URL", "http://127.0.0.1:8081")
    return key.rstrip("/"), base.rstrip("/")


def api_request(method: str, path: str, body: dict | None = None) -> dict:
    key, base = api_config()
    if not key:
        raise RuntimeError("API key missing")
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        f"{base}{path}",
        data=data,
        headers={
            "X-Repository-Detective-API-Key": key,
            "Content-Type": "application/json",
        },
        method=method,
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode())


def severity_passes(severity: str, gate: str) -> bool:
    s = SEVERITY_RANK.get((severity or "low").lower(), 1)
    g = SEVERITY_GATES.get((gate or "high").lower(), 3)
    return s >= g


def classify_unmapped(
    finding_status: str,
    has_mapped: bool,
    scan_dry_run: bool,
    issue_sync: str,
    filing_enabled: bool,
    policy_level: str | None,
    issue_policy: str | None,
    severity: str,
    severity_gate: str,
    confidence: float,
    confidence_gate: float,
) -> str:
    if has_mapped:
        return "already_mapped"
    if finding_status in ("suppressed", "false_positive"):
        return "suppressed"
    if scan_dry_run or issue_sync == "skipped":
        return "report_only"
    if issue_policy == "off" or policy_level == "monitor_only":
        return "no_issue_policy"
    if not filing_enabled:
        return "filing_disabled"
    if issue_sync == "failed":
        return "forge_error"
    if not severity_passes(severity, severity_gate):
        return "below_threshold"
    if confidence < confidence_gate:
        return "below_threshold"
    return "unknown"


def repo_settings(conn: sqlite3.Connection, repo_id: int) -> dict:
    row = conn.execute(
        """
        SELECT enabled, schedule_enabled, issue_policy, policy_level, severity_gate, confidence_gate
        FROM repo_settings WHERE repository_id = ?
        """,
        (repo_id,),
    ).fetchone()
    if not row:
        return {
            "filing_enabled": True,
            "issue_policy": "all",
            "policy_level": "issue_only",
            "severity_gate": "high",
            "confidence_gate": 0.5,
        }
    enabled, _, issue_policy, policy_level, severity_gate, confidence_gate = row
    scan_on = enabled is None or enabled == 1
    filing = scan_on and issue_policy != "off" and policy_level != "monitor_only"
    return {
        "filing_enabled": filing,
        "issue_policy": issue_policy or "all",
        "policy_level": policy_level or "issue_only",
        "severity_gate": severity_gate or "high",
        "confidence_gate": float(confidence_gate if confidence_gate is not None else 0.5),
    }


def latest_scan_meta(conn: sqlite3.Connection, repo_id: int) -> tuple[bool, str]:
    row = conn.execute(
        """
        SELECT summary_json FROM scans
        WHERE repository_id = ? ORDER BY started_at DESC LIMIT 1
        """,
        (repo_id,),
    ).fetchone()
    if not row or not row[0]:
        return False, ""
    try:
        summary = json.loads(row[0])
    except json.JSONDecodeError:
        return False, ""
    dry = bool(summary.get("dry_run_report_only"))
    sync = str(summary.get("issue_sync_status") or "")
    return dry, sync


def fetch_unmapped_findings(conn: sqlite3.Connection, repo_id: int | None = None) -> list[dict]:
    where = "WHERE f.status = 'open'"
    args: list = []
    if repo_id is not None:
        where += " AND f.repository_id = ?"
        args.append(repo_id)
    rows = conn.execute(
        f"""
        SELECT f.id, f.repository_id, r.full_name, f.rule_id, f.severity, f.confidence,
               f.title, f.status, f.file_path, f.line,
               EXISTS(SELECT 1 FROM external_issues e WHERE e.finding_id = f.id AND e.state = 'open')
        FROM findings f
        JOIN repositories r ON r.id = f.repository_id
        {where}
        """,
        args,
    ).fetchall()
    out = []
    for row in rows:
        fid, rid, full_name, rule_id, severity, confidence, title, status, file_path, line, has_mapped = row
        if has_mapped:
            continue
        settings = repo_settings(conn, rid)
        dry, sync = latest_scan_meta(conn, rid)
        reason = classify_unmapped(
            status,
            False,
            dry,
            sync,
            settings["filing_enabled"],
            settings["policy_level"],
            settings["issue_policy"],
            severity,
            settings["severity_gate"],
            confidence or 0.0,
            settings["confidence_gate"],
        )
        out.append(
            {
                "finding_id": fid,
                "repository_id": rid,
                "full_name": full_name,
                "rule_id": rule_id,
                "severity": severity,
                "confidence": confidence,
                "title": title,
                "file_path": file_path,
                "line": line,
                "reason": reason,
                "eligible_to_file": reason in ("unknown",),
            }
        )
    return out


def forge_errors(conn: sqlite3.Connection, limit: int = 20) -> list[dict]:
    rows = conn.execute(
        """
        SELECT r.full_name, s.id, s.error, s.finished_at
        FROM scans s
        JOIN repositories r ON r.id = s.repository_id
        WHERE s.error IS NOT NULL AND TRIM(s.error) != ''
        ORDER BY s.finished_at DESC
        LIMIT ?
        """,
        (limit,),
    ).fetchall()
    return [
        {"repo": r[0], "scan_id": r[1], "error": r[2], "finished_at": r[3]}
        for r in rows
    ]


def resolve_repo(conn: sqlite3.Connection, spec: str) -> tuple[int, str]:
    if "/" not in spec:
        raise ValueError(f"expected owner/repo, got {spec!r}")
    owner, name = spec.split("/", 1)
    row = conn.execute(
        "SELECT id, full_name FROM repositories WHERE owner = ? AND name = ?",
        (owner, name),
    ).fetchone()
    if not row:
        raise ValueError(f"repository not found: {spec}")
    return int(row[0]), str(row[1])


def write_report(path: Path, title: str, body: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(f"# {title}\n\nGenerated: {iso_now()}\n\n{body}\n")


def iso_now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def cmd_summary(conn: sqlite3.Connection) -> int:
    findings = fetch_unmapped_findings(conn)
    mapped = conn.execute(
        """
        SELECT COUNT(1) FROM external_issues e
        INNER JOIN findings f ON f.id = e.finding_id
        WHERE e.state = 'open' AND f.status = 'open'
        """
    ).fetchone()[0]
    open_total = conn.execute("SELECT COUNT(1) FROM findings WHERE status = 'open'").fetchone()[0]
    reasons = Counter(f["reason"] for f in findings)
    by_repo = Counter(f["full_name"] for f in findings)
    by_rule = Counter(f["rule_id"] for f in findings)
    payload = {
        "open_findings": open_total,
        "mapped_forge_issues": mapped,
        "unmapped_findings": len(findings),
        "reason_counts": dict(reasons),
        "top_repos_by_unmapped": by_repo.most_common(10),
        "top_rules_by_unmapped": by_rule.most_common(10),
        "recent_forge_errors": forge_errors(conn),
    }
    body = json.dumps(payload, indent=2)
    write_report(REPORT_DIR / "issue-filing-reconcile-summary.md", "Issue filing reconcile summary", f"```json\n{body}\n```")
    print(body)
    return 0


def cmd_repo_dry_run(conn: sqlite3.Connection, spec: str, limit: int) -> int:
    repo_id, full_name = resolve_repo(conn, spec)
    settings = repo_settings(conn, repo_id)
    dry, sync = latest_scan_meta(conn, repo_id)
    findings = fetch_unmapped_findings(conn, repo_id)
    eligible = [f for f in findings if f["eligible_to_file"]][:limit]
    blocked = [f for f in findings if not f["eligible_to_file"]]
    blocked_counts = Counter(f["reason"] for f in blocked)
    lines = [
        f"Repository: **{full_name}** (id {repo_id})",
        "",
        f"- Filing enabled (effective): **{settings['filing_enabled']}**",
        f"- Latest scan report-only: **{dry}**",
        f"- Latest issue sync: **{sync or '—'}**",
        f"- Unmapped open findings: **{len(findings)}**",
        f"- Eligible to file (heuristic): **{len(eligible)}** (showing up to {limit})",
        "",
        "## Blocked reason counts",
        "",
        "```json",
        json.dumps(dict(blocked_counts), indent=2),
        "```",
        "",
        "## Eligible preview",
        "",
        "| Severity | Rule | Title | File |",
        "|----------|------|-------|------|",
    ]
    for f in eligible:
        lines.append(
            f"| {f['severity']} | `{f['rule_id']}` | {f['title'][:60]} | `{f['file_path']}:{f['line']}` |"
        )
    report_path = REPORT_DIR / f"issue-filing-dry-run-{full_name.replace('/', '-')}.md"
    write_report(report_path, f"Issue filing dry-run — {full_name}", "\n".join(lines))
    print(json.dumps({"repo": full_name, "eligible": len(eligible), "blocked": len(blocked), "report": str(report_path)}, indent=2))
    return 0


def cmd_apply(conn: sqlite3.Connection, spec: str, limit: int) -> int:
    repo_id, full_name = resolve_repo(conn, spec)
    before = REPORT_DIR / f"issue-filing-apply-before-{full_name.replace('/', '-')}.md"
    cmd_repo_dry_run(conn, spec, limit)
    findings = fetch_unmapped_findings(conn, repo_id)
    eligible = [f for f in findings if f["eligible_to_file"]]
    settings = repo_settings(conn, repo_id)
    if not settings["filing_enabled"]:
        print("ABORT: filing disabled for repo", file=sys.stderr)
        return 2
    if len(eligible) == 0:
        print("ABORT: no eligible unmapped findings for canary apply", file=sys.stderr)
        return 2
    if len(eligible) > limit:
        print(f"ABORT: {len(eligible)} eligible findings exceeds --limit {limit}", file=sys.stderr)
        return 2
    # Safety: never bulk-close; reconcile only updates tracked issues
    owner, name = full_name.split("/", 1)
    write_report(
        before,
        f"Issue filing apply BEFORE — {full_name}",
        f"Eligible: {len(eligible)}\nLimit: {limit}\n",
    )
    results: dict = {"repo": full_name, "steps": []}
    try:
        preview = api_request("GET", f"/api/v1/repos/{repo_id}/reconcile-issues/preview")
        results["steps"].append({"reconcile_preview": preview})
    except urllib.error.HTTPError as exc:
        results["steps"].append({"reconcile_preview_error": exc.read().decode()[:500]})
    try:
        analyze = api_request(
            "POST",
            "/api/v1/analyze",
            {
                "owner": owner,
                "repository": name,
                "ref": "main",
                "report_only_dry_run": False,
            },
        )
        results["steps"].append({"analyze_started": analyze})
    except urllib.error.HTTPError as exc:
        results["steps"].append({"analyze_error": exc.read().decode()[:500]})
        write_report(
            REPORT_DIR / f"issue-filing-apply-after-{full_name.replace('/', '-')}.md",
            f"Issue filing apply AFTER — {full_name}",
            f"```json\n{json.dumps(results, indent=2)}\n```",
        )
        return 1
    time.sleep(5)
    try:
        reconcile = api_request("POST", f"/api/v1/repos/{repo_id}/reconcile-issues", {})
        results["steps"].append({"reconcile_apply": reconcile})
    except urllib.error.HTTPError as exc:
        results["steps"].append({"reconcile_apply_error": exc.read().decode()[:500]})
    after = REPORT_DIR / f"issue-filing-apply-after-{full_name.replace('/', '-')}.md"
    write_report(after, f"Issue filing apply AFTER — {full_name}", f"```json\n{json.dumps(results, indent=2)}\n```")
    print(json.dumps(results, indent=2))
    return 0


def main(argv: list[str] | None = None) -> int:
    load_env()
    p = argparse.ArgumentParser(description="Issue filing reconciliation audit")
    p.add_argument("--db-path", type=Path, default=DEFAULT_DB)
    p.add_argument("--summary", action="store_true", help="Fleet-wide unmapped reason summary")
    p.add_argument("--repo", help="owner/repo for repo-scoped dry-run or apply")
    p.add_argument("--dry-run", action="store_true", help="Repo filing dry-run preview")
    p.add_argument("--apply", action="store_true", help="One-repo canary: reconcile + analyze with filing")
    p.add_argument("--limit", type=int, default=DEFAULT_LIMIT, help="Max eligible findings for apply (default 10)")
    args = p.parse_args(argv)

    if not args.db_path.exists():
        print(f"database not found: {args.db_path}", file=sys.stderr)
        return 2

    conn = sqlite3.connect(args.db_path)
    try:
        if args.apply:
            if not args.repo:
                print("--apply requires --repo owner/name", file=sys.stderr)
                return 2
            return cmd_apply(conn, args.repo, args.limit)
        if args.dry_run:
            if not args.repo:
                print("--dry-run requires --repo owner/name", file=sys.stderr)
                return 2
            return cmd_repo_dry_run(conn, args.repo, args.limit)
        if args.summary or not any([args.apply, args.dry_run]):
            return cmd_summary(conn)
        return 0
    finally:
        conn.close()


if __name__ == "__main__":
    raise SystemExit(main())
