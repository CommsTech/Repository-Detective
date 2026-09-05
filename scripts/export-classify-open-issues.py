#!/usr/bin/env python3
"""Export and classify open Gitea issues for commstech/Repository-Detective."""
from __future__ import annotations

import json
import os
import re
import sqlite3
import sys
import urllib.request
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXPORT_PATH = ROOT / "docs/dogfood-reports/current-open-issues-export.md"
RECON_PATH = ROOT / "docs/dogfood-reports/current-open-issues-reconciliation.md"
BACKLOG_PATH = ROOT / "docs/dogfood-reports/real-active-backlog-report.md"
DB_PATH = ROOT / "data/repository-detective.db"


def load_env(path: Path) -> dict[str, str]:
    env: dict[str, str] = {}
    if not path.exists():
        return env
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def gitea_get(url: str, token: str) -> list | dict:
    req = urllib.request.Request(url, headers={"Authorization": f"token {token}"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.load(resp)


def fetch_open_issues(base: str, token: str) -> list[dict]:
    issues: list[dict] = []
    page = 1
    while True:
        url = f"{base}/api/v1/repos/commstech/Repository-Detective/issues?state=open&type=issues&limit=50&page={page}"
        batch = gitea_get(url, token)
        if not batch:
            break
        issues.extend(batch)
        if len(batch) < 50:
            break
        page += 1
    return issues


def extract_fingerprint(body: str) -> str:
    for line in body.splitlines():
        line = line.strip().lstrip("- ")
        for marker in (
            "Repository Detective fingerprint:",
            "Repository Detective fingerprint:",
        ):
            if line.startswith(marker):
                return line[len(marker) :].strip()
    return ""


def extract_field(body: str, label: str) -> str:
    for line in body.splitlines():
        line = line.strip().lstrip("- ")
        if line.startswith(label + ":"):
            return line.split(":", 1)[1].strip()
    return ""


def db_connect() -> sqlite3.Connection:
    return sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)


def latest_reconcilable_scan(conn: sqlite3.Connection) -> tuple[str, int, dict]:
    cur = conn.cursor()
    cur.execute(
        """
        SELECT id, summary_json FROM scans
        WHERE repository_id = 1 AND status IN ('completed','analysis_complete','persistence_incomplete')
        ORDER BY started_at DESC LIMIT 5
        """
    )
    for scan_id, summary_raw in cur.fetchall():
        summary = json.loads(summary_raw or "{}")
        expected = summary.get("persistence_expected_count") or summary.get("issues_found") or 0
        persisted = summary.get("persistence_persisted_count")
        ps = summary.get("persistence_status")
        cur.execute("SELECT COUNT(1) FROM finding_instances WHERE scan_id = ?", (scan_id,))
        inst = cur.fetchone()[0]
        if ps == "complete" or (expected and inst >= expected):
            return scan_id, inst, summary
        if not ps and expected and inst >= expected:
            return scan_id, inst, summary
    return "", 0, {}


def fingerprints_in_scan(conn: sqlite3.Connection, scan_id: str) -> set[str]:
    cur = conn.cursor()
    cur.execute(
        """
        SELECT DISTINCT f.fingerprint
        FROM finding_instances fi
        JOIN findings f ON f.id = fi.finding_id
        WHERE fi.scan_id = ? AND f.repository_id = 1
        """,
        (scan_id,),
    )
    return {row[0] for row in cur.fetchall()}


def external_maps(conn: sqlite3.Connection) -> dict[int, dict]:
    cur = conn.cursor()
    cur.execute(
        """
        SELECT ei.issue_number, f.fingerprint, ei.state, f.id
        FROM external_issues ei
        JOIN findings f ON f.id = ei.finding_id
        WHERE f.repository_id = 1
        """
    )
    out: dict[int, dict] = {}
    for num, fp, state, fid in cur.fetchall():
        out[num] = {"fingerprint": fp, "state": state, "finding_id": fid}
    return out


def scanner_status(conn: sqlite3.Connection, scan_id: str) -> dict[str, str]:
    cur = conn.cursor()
    cur.execute(
        "SELECT scanner_name, status FROM scanner_results WHERE scan_id = ?",
        (scan_id,),
    )
    return {name.lower(): status for name, status in cur.fetchall()}


def classify_issue(issue: dict, fp: str, fps_in_scan: set[str], ext_by_num: dict, fp_to_issues: dict, scan_id: str) -> dict:
    num = issue["number"]
    title = issue.get("title", "")
    body = issue.get("body") or ""
    source = extract_field(body, "Source")
    rule = extract_field(body, "Rule ID")
    severity = extract_field(body, "Severity")
    labels = [lb.get("name", "") for lb in issue.get("labels", [])]

    classification = "needs_human_review"
    canonical = ""
    action = "none"
    evidence = "unverified"
    latest_presence = "unknown"

    if not fp:
        classification = "needs_human_review"
        action = "review_untracked"
    elif num not in ext_by_num:
        classification = "missing_local_mapping_backfilled"
        action = "backfill_mapping"
    elif fp not in fps_in_scan and scan_id:
        classification = "resolved_absent_from_latest_scan"
        latest_presence = "absent"
        action = "evidence_closure"
        evidence = "pending_verify"
    elif fp in fps_in_scan:
        classification = "active_present_in_latest_scan"
        latest_presence = "present"
        action = "fix_in_code_batch"
    else:
        classification = "needs_human_review"

    dupes = fp_to_issues.get(fp, [])
    if fp and len(dupes) > 1:
        canonical = str(min(dupes))
        if num != min(dupes):
            classification = "duplicate_existing_fingerprint"
            action = "link_canonical"
            latest_presence = "present" if fp in fps_in_scan else "absent"

    if "repository-detective/summary" in " ".join(labels) or title.startswith("Code Review Summary"):
        classification = "out_of_scope_for_current_batch"
        action = "ignore_summary"

    if "resolved-verified" in labels or "lifecycle/resolved-verified" in labels:
        if classification == "resolved_absent_from_latest_scan":
            classification = "resolved_verified_open_by_policy"
            action = "keep_open_by_policy"
            evidence = "verified"

    return {
        "issue_number": num,
        "title": title,
        "fingerprint": fp,
        "source": source,
        "rule_id": rule,
        "severity": severity,
        "latest_scan_presence": latest_presence,
        "classification": classification,
        "canonical_issue": canonical,
        "action_taken": action,
        "evidence_status": evidence,
    }


def main() -> int:
    env = load_env(ROOT / ".env")
    gitea_url = env.get("REPOSITORY_DETECTIVE_GITEA_URL") or env.get("REPOSITORY_DETECTIVE_GITEA_URL", "")
    token = env.get("REPOSITORY_DETECTIVE_GITEA_TOKEN") or env.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    if not gitea_url or not token:
        print("missing gitea env", file=sys.stderr)
        return 1

    issues = fetch_open_issues(gitea_url.rstrip("/"), token)
    conn = db_connect()
    scan_id, inst_count, summary = latest_reconcilable_scan(conn)
    fps = fingerprints_in_scan(conn, scan_id) if scan_id else set()
    ext_by_num = external_maps(conn)
    fp_to_issues: dict[str, list[int]] = defaultdict(list)

    rows: list[dict] = []
    for issue in issues:
        fp = extract_fingerprint(issue.get("body") or "")
        if fp:
            fp_to_issues[fp].append(issue["number"])
        rows.append(issue)

    classified = [
        classify_issue(issue, extract_fingerprint(issue.get("body") or ""), fps, ext_by_num, fp_to_issues, scan_id)
        for issue in rows
    ]
    counts = Counter(c["classification"] for c in classified)

    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    EXPORT_PATH.parent.mkdir(parents=True, exist_ok=True)
    EXPORT_PATH.write_text(
        f"# Open issues export — commstech/Repository-Detective\n\nGenerated: {now}\n\nTotal open: {len(issues)}\n\n"
        + "| # | Title | Fingerprint | Labels |\n|---|-------|-------------|--------|\n"
        + "\n".join(
            f"| #{i['number']} | {i.get('title','').replace('|','/')[:80]} | {extract_fingerprint(i.get('body') or '')[:20]} | "
            f"{', '.join(l.get('name','') for l in i.get('labels', [])[:3])} |"
            for i in issues
        )
        + "\n"
    )

    recon_lines = [
        f"# Current open issues reconciliation — commstech/Repository-Detective\n",
        f"Generated: {now}\n",
        f"Reconciled against scan **`{scan_id or 'none'}`** ({inst_count} finding instances).\n",
        "## Summary\n",
        "| Metric | Count |",
        "|--------|------:|",
        f"| Gitea open issues exported | {len(issues)} |",
    ]
    for k, v in sorted(counts.items()):
        recon_lines.append(f"| {k} | {v} |")
    recon_lines += [
        "\n## Detail\n",
        "| issue | title | fingerprint | source | severity | scan presence | classification | canonical | action | evidence |",
        "|------:|-------|-------------|--------|----------|---------------|----------------|-----------|--------|----------|",
    ]
    for c in sorted(classified, key=lambda x: x["issue_number"]):
        recon_lines.append(
            f"| #{c['issue_number']} | {c['title'][:50].replace('|','/')} | `{c['fingerprint'][:24]}` | {c['source']} | {c['severity']} | "
            f"{c['latest_scan_presence']} | {c['classification']} | {c['canonical_issue']} | {c['action_taken']} | {c['evidence_status']} |"
        )
    RECON_PATH.write_text("\n".join(recon_lines) + "\n")

    active = [c for c in classified if c["classification"] == "active_present_in_latest_scan"]
    health_ignored = [c for c in active if c.get("rule_id") == "HEALTH-IGNORED-ERROR"]
    backlog_lines = [
        f"# Real active backlog — commstech/Repository-Detective\n",
        f"Generated: {now}\n",
        f"Scan: **`{scan_id or 'none'}`**\n",
        "## Summary\n",
        "| Metric | Count |",
        "|--------|------:|",
        f"| Gitea open (exported) | {len(issues)} |",
        f"| active_present_in_latest_scan | {counts.get('active_present_in_latest_scan', 0)} |",
        f"| resolved_absent_from_latest_scan | {counts.get('resolved_absent_from_latest_scan', 0)} |",
        f"| resolved_verified_open_by_policy | {counts.get('resolved_verified_open_by_policy', 0)} |",
        f"| duplicate_existing_fingerprint | {counts.get('duplicate_existing_fingerprint', 0)} |",
        f"| out_of_scope_for_current_batch | {counts.get('out_of_scope_for_current_batch', 0)} |",
        f"| needs_human_review | {counts.get('needs_human_review', 0)} |",
        f"| HEALTH-IGNORED-ERROR (active) | {len(health_ignored)} |",
        "\n## Why open count grows\n",
        "- Scanner coverage expanded (more rules/tools per scan).\n",
        "- Evidence closure keeps issues open when `close_issues=false`.\n",
        "- Duplicates are labeled, not deleted.\n",
        "- New scanner availability creates variance findings.\n",
        "\n## Active code-fix queue (top 30 by issue #)\n",
        "| # | Rule | Source | Title |",
        "|--:|------|--------|-------|",
    ]
    for c in sorted(active, key=lambda x: x["issue_number"])[:30]:
        backlog_lines.append(
            f"| #{c['issue_number']} | {c.get('rule_id','')} | {c.get('source','')} | {c['title'][:55].replace('|','/')} |"
        )
    BACKLOG_PATH.write_text("\n".join(backlog_lines) + "\n")

    print(json.dumps({"open": len(issues), "scan_id": scan_id, "instances": inst_count, "counts": dict(counts), "active": len(active)}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
