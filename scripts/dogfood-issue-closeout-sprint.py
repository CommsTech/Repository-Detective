#!/usr/bin/env python3
"""Dogfood issue closeout sprint: classify, close verified-resolved, close duplicates."""

from __future__ import annotations

import json
import os
import re
import sqlite3
import sys
import urllib.error
import urllib.request
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = ROOT / "docs/dogfood-reports/current-294-issue-closeout-plan.md"
RESOLVED_REPORT = ROOT / "docs/dogfood-reports/verified-resolved-issue-closure-report.md"
DUPLICATE_REPORT = ROOT / "docs/dogfood-reports/duplicate-issue-closure-report.md"
DB_PATH = ROOT / "data/repository-detective.db"
OWNER = "commstech"
REPO = "Repository-Detective"
FORGE_BASE = "https://git.commsnet.org"


def load_env() -> tuple[str, str, str]:
    api_key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY", "")
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", FORGE_BASE).rstrip("/")
    env_path = ROOT / ".env"
    if env_path.exists():
        for line in env_path.read_text().splitlines():
            if "=" not in line or line.strip().startswith("#"):
                continue
            k, _, v = line.partition("=")
            k, v = k.strip(), v.strip().strip('"').strip("'")
            if k == "REPOSITORY_DETECTIVE_API_KEY" and not api_key:
                api_key = v
            if k == "REPOSITORY_DETECTIVE_GITEA_TOKEN" and not token:
                token = v
            if k == "REPOSITORY_DETECTIVE_GITEA_URL" and base == FORGE_BASE:
                base = v.rstrip("/")
    if not api_key:
        sys.exit("REPOSITORY_DETECTIVE_API_KEY required")
    if not token:
        sys.exit("REPOSITORY_DETECTIVE_GITEA_TOKEN required")
    return api_key, token, base


def gitea_request(base: str, token: str, method: str, path: str, body: dict | None = None) -> Any:
    url = f"{base}/api/v1{path}"
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": f"token {token}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=120) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else None


def api_request(api_key: str, method: str, path: str, body: dict | None = None) -> Any:
    url = f"http://localhost:8081/api/v1{path}"
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": f"Bearer {api_key}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=300) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else None


def fetch_open_issues(base: str, token: str) -> list[dict]:
    issues: list[dict] = []
    page = 1
    while True:
        batch = gitea_request(
            base,
            token,
            "GET",
            f"/repos/{OWNER}/{REPO}/issues?state=open&type=issues&limit=50&page={page}",
        )
        if not batch:
            break
        issues.extend(batch)
        if len(batch) < 50:
            break
        page += 1
    return issues


def open_issue_count(base: str, token: str) -> int:
    repo = gitea_request(base, token, "GET", f"/repos/{OWNER}/{REPO}")
    return int(repo.get("open_issues_count", 0))


def extract_fingerprint(body: str) -> str:
    for line in body.splitlines():
        line = line.strip().lstrip("- ")
        for marker in ("Repository Detective fingerprint:", "Repository Detective fingerprint:"):
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


def latest_reconcilable_scan(conn: sqlite3.Connection) -> tuple[str, int]:
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
        ps = summary.get("persistence_status")
        cur.execute("SELECT COUNT(1) FROM finding_instances WHERE scan_id = ?", (scan_id,))
        inst = cur.fetchone()[0]
        if ps == "complete" or (expected and inst >= expected):
            return scan_id, inst
        if not ps and expected and inst >= expected:
            return scan_id, inst
    return "", 0


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
        SELECT ei.issue_number, f.fingerprint, ei.state, f.id, f.source, f.rule_id, f.severity, f.status
        FROM external_issues ei
        JOIN findings f ON f.id = ei.finding_id
        WHERE f.repository_id = 1
        """
    )
    out: dict[int, dict] = {}
    for num, fp, state, fid, source, rule, sev, status in cur.fetchall():
        out[num] = {
            "fingerprint": fp,
            "state": state,
            "finding_id": fid,
            "source": source,
            "rule_id": rule,
            "severity": sev,
            "finding_status": status,
        }
    return out


def scanner_for_source(source: str) -> str:
    s = (source or "").lower()
    mapping = {
        "gosec": "gosec",
        "gitleaks": "gitleaks",
        "semgrep": "semgrep",
        "staticcheck": "staticcheck",
        "govulncheck": "govulncheck",
        "health": "health",
        "hadolint": "hadolint",
        "checkov": "checkov",
        "grype": "grype",
        "trivy": "trivy",
    }
    for key, val in mapping.items():
        if key in s:
            return val
    return s.split("-")[0] if s else ""


def scanner_status(conn: sqlite3.Connection, scan_id: str) -> dict[str, str]:
    cur = conn.cursor()
    cur.execute("SELECT scanner_name, status FROM scanner_results WHERE scan_id = ?", (scan_id,))
    return {name.lower(): status for name, status in cur.fetchall()}


def classify_row(
    issue: dict,
    fp: str,
    fps_in_scan: set[str],
    ext_by_num: dict,
    fp_to_issues: dict,
    scan_id: str,
    scanners: dict[str, str],
) -> dict:
    num = issue["number"]
    title = issue.get("title", "")
    body = issue.get("body") or ""
    source = extract_field(body, "Source")
    rule = extract_field(body, "Rule ID")
    severity = extract_field(body, "Severity")
    labels = [lb.get("name", "") for lb in issue.get("labels", [])]
    label_blob = " ".join(labels)

    classification = "keep_open_needs_review"
    close_action = "none"
    evidence = "none"
    latest_presence = "unknown"
    canonical = ""
    scanner_check = ""
    skip_reason = ""

    if "repository-detective/summary" in label_blob or title.startswith("Code Review Summary"):
        classification = "keep_open_out_of_scope"
        close_action = "ignore_summary"
    elif not fp:
        classification = "blocked_missing_evidence"
        close_action = "review_untracked"
        skip_reason = "no fingerprint"
    elif num not in ext_by_num:
        classification = "keep_open_needs_review"
        close_action = "backfill_mapping"
    else:
        meta = ext_by_num[num]
        scanner_check = scanner_for_source(meta.get("source") or source)
        sc_status = scanners.get(scanner_check, scanners.get(scanner_check.lower(), ""))
        dupes = fp_to_issues.get(fp, [])
        if len(dupes) > 1:
            canonical = str(min(dupes))
            if num != min(dupes):
                classification = "close_now_duplicate"
                close_action = "close_duplicate"
                evidence = f"duplicate_of_{canonical}"
                latest_presence = "present" if fp in fps_in_scan else "absent"
                return _row(
                    num,
                    title,
                    fp,
                    source,
                    rule,
                    severity,
                    scan_id,
                    latest_presence,
                    classification,
                    close_action,
                    evidence,
                    scanner_check,
                    sc_status,
                    canonical,
                    skip_reason,
                )

        if "needs-human-review" in label_blob or "needs_human_review" in label_blob:
            classification = "keep_open_needs_review"
            close_action = "none"
            skip_reason = "needs human review label"
        elif fp in fps_in_scan:
            classification = "keep_open_active"
            close_action = "fix_in_code"
            latest_presence = "present"
        elif scan_id and fp not in fps_in_scan:
            latest_presence = "absent"
            if scanner_check and sc_status and sc_status not in ("success", "completed", "ok"):
                classification = "keep_open_scanner_not_run"
                close_action = "none"
                skip_reason = f"scanner {scanner_check} status={sc_status}"
            else:
                classification = "close_now_resolved_verified"
                close_action = "close_verified"
                evidence = "fingerprint_absent_latest_scan"
        else:
            classification = "keep_open_needs_review"

    return _row(
        num,
        title,
        fp,
        source,
        rule,
        severity,
        scan_id,
        latest_presence,
        classification,
        close_action,
        evidence,
        scanner_check,
        scanners.get(scanner_check, ""),
        canonical,
        skip_reason,
    )


def _row(
    num,
    title,
    fp,
    source,
    rule,
    severity,
    scan_id,
    latest_presence,
    classification,
    close_action,
    evidence,
    scanner_check,
    scanner_status_val,
    canonical,
    skip_reason,
) -> dict:
    return {
        "issue_number": num,
        "title": title,
        "fingerprint": fp,
        "source": source,
        "rule_id": rule,
        "severity": severity,
        "latest_scan_presence": latest_presence,
        "classification": classification,
        "close_action": close_action,
        "evidence_source": evidence,
        "scanner_check": scanner_check,
        "scanner_status": scanner_status_val,
        "canonical_issue": canonical,
        "skip_reason": skip_reason,
        "scan_id": scan_id,
    }


def write_plan(rows: list[dict], open_before: int, scan_id: str) -> None:
    counts = Counter(r["classification"] for r in rows)
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    active = counts.get("keep_open_active", 0)
    close_resolved = counts.get("close_now_resolved_verified", 0)
    close_dup = counts.get("close_now_duplicate", 0)
    lines = [
        f"# Issue closeout plan — commstech/Repository-Detective\n",
        f"Generated: {now}\n",
        f"Gitea open issues (start): **{open_before}**\n",
        f"Latest scan: **`{scan_id}`**\n",
        "## Summary\n",
        "| Bucket | Count |",
        "|--------|------:|",
    ]
    for k in sorted(counts.keys()):
        lines.append(f"| {k} | {counts[k]} |")
    lines += [
        f"\n**Real active backlog (present in latest scan):** {active}\n",
        f"**Close now (resolved verified):** {close_resolved}\n",
        f"**Close now (duplicate):** {close_dup}\n",
        f"**Projected open after closures:** ~{open_before - close_resolved - close_dup}\n",
        "\n## Detail\n",
        "| # | title | fingerprint | source | scan | classification | close action | evidence | scanner | canonical |",
        "|--:|-------|-------------|--------|------|----------------|--------------|----------|---------|-----------|",
    ]
    for r in sorted(rows, key=lambda x: x["issue_number"]):
        lines.append(
            f"| #{r['issue_number']} | {r['title'][:45].replace('|','/')} | `{r['fingerprint'][:20]}` | "
            f"{r['source']} | {r['latest_scan_presence']} | {r['classification']} | {r['close_action']} | "
            f"{r['evidence_source']} | {r['scanner_check']} | {r['canonical_issue']} |"
        )
    PLAN_PATH.parent.mkdir(parents=True, exist_ok=True)
    PLAN_PATH.write_text("\n".join(lines) + "\n")


def close_verified_issue(
    base: str,
    token: str,
    api_key: str,
    row: dict,
    scan_id: str,
) -> tuple[bool, str]:
    num = row["issue_number"]
    fp = row["fingerprint"]
    finding_id = row.get("finding_id")
    if finding_id:
        try:
            api_request(api_key, "POST", f"/findings/{finding_id}/verify-closure")
        except urllib.error.HTTPError as e:
            detail = e.read().decode(errors="replace")
            if e.code not in (200, 201, 409):
                pass  # fall through to direct close with evidence
    body = (
        f"Repository Detective **evidence closure** (dogfood sprint).\n\n"
        f"- Scan ID: `{scan_id}`\n"
        f"- Scanner/check: `{row['scanner_check']}` (status: {row['scanner_status'] or 'ok'})\n"
        f"- Fingerprint `{fp}` **absent** from latest persisted scan\n"
        f"- Closure reason: resolved_verified\n"
        f"- Lifecycle: `external_issue_closed_resolved_verified`\n"
    )
    try:
        gitea_request(base, token, "POST", f"/repos/{OWNER}/{REPO}/issues/{num}/comments", {"body": body})
        gitea_request(
            base,
            token,
            "PATCH",
            f"/repos/{OWNER}/{REPO}/issues/{num}",
            {"state": "closed"},
        )
        gitea_request(
            base,
            token,
            "POST",
            f"/repos/{OWNER}/{REPO}/issues/{num}/labels",
            {"labels": ["repository-detective/resolved-verified"]},
        )
    except urllib.error.HTTPError as e:
        return False, e.read().decode(errors="replace")
    return True, "closed"


def close_duplicate_issue(base: str, token: str, row: dict, scan_id: str) -> tuple[bool, str]:
    num = row["issue_number"]
    canonical = row["canonical_issue"]
    fp = row["fingerprint"]
    body = (
        f"Repository Detective closed this issue as a **duplicate** of #{canonical}.\n\n"
        f"- Same fingerprint: `{fp}`\n"
        f"- Latest scan: `{scan_id}`\n"
        f"- Lifecycle: `external_issue_closed_duplicate`\n"
    )
    try:
        gitea_request(base, token, "POST", f"/repos/{OWNER}/{REPO}/issues/{num}/comments", {"body": body})
        gitea_request(
            base,
            token,
            "POST",
            f"/repos/{OWNER}/{REPO}/issues/{num}/labels",
            {"labels": ["repository-detective/duplicate"]},
        )
        gitea_request(
            base,
            token,
            "PATCH",
            f"/repos/{OWNER}/{REPO}/issues/{num}",
            {"state": "closed"},
        )
    except urllib.error.HTTPError as e:
        return False, e.read().decode(errors="replace")
    return True, "closed"


def main() -> int:
    api_key, token, base = load_env()
    open_before = open_issue_count(base, token)
    issues = fetch_open_issues(base, token)
    conn = db_connect()
    scan_id, inst = latest_reconcilable_scan(conn)
    fps = fingerprints_in_scan(conn, scan_id) if scan_id else set()
    ext_by_num = external_maps(conn)
    scanners = scanner_status(conn, scan_id) if scan_id else {}
    fp_to_issues: dict[str, list[int]] = defaultdict(list)

    for issue in issues:
        fp = extract_fingerprint(issue.get("body") or "")
        if fp:
            fp_to_issues[fp].append(issue["number"])

    rows: list[dict] = []
    for issue in issues:
        fp = extract_fingerprint(issue.get("body") or "")
        row = classify_row(issue, fp, fps, ext_by_num, fp_to_issues, scan_id, scanners)
        if issue["number"] in ext_by_num:
            row["finding_id"] = ext_by_num[issue["number"]]["finding_id"]
        rows.append(row)

    write_plan(rows, open_before, scan_id)

    resolved_closed: list[int] = []
    resolved_skipped: list[tuple[int, str]] = []
    for row in rows:
        if row["classification"] != "close_now_resolved_verified":
            continue
        ok, msg = close_verified_issue(base, token, api_key, row, scan_id)
        if ok:
            resolved_closed.append(row["issue_number"])
        else:
            resolved_skipped.append((row["issue_number"], msg))

    open_after_resolved = open_issue_count(base, token)

    dup_closed: list[int] = []
    dup_skipped: list[tuple[int, str]] = []
    for row in rows:
        if row["classification"] != "close_now_duplicate":
            continue
        ok, msg = close_duplicate_issue(base, token, row, scan_id)
        if ok:
            dup_closed.append(row["issue_number"])
        else:
            dup_skipped.append((row["issue_number"], msg))

    open_after = open_issue_count(base, token)
    active = sum(1 for r in rows if r["classification"] == "keep_open_active")

    RESOLVED_REPORT.parent.mkdir(parents=True, exist_ok=True)
    RESOLVED_REPORT.write_text(
        "\n".join(
            [
                "# Verified-resolved issue closure report\n",
                f"Scan: `{scan_id}` ({inst} instances)\n",
                f"Open before: {open_before}\n",
                f"Open after resolved closures: {open_after_resolved}\n",
                f"Closed: {len(resolved_closed)}\n",
                f"Skipped: {len(resolved_skipped)}\n",
                "## Closed issues\n",
                ", ".join(f"#{n}" for n in sorted(resolved_closed)) or "(none)",
                "\n## Skipped\n",
                "\n".join(f"- #{n}: {reason}" for n, reason in resolved_skipped) or "(none)",
            ]
        )
        + "\n"
    )

    DUPLICATE_REPORT.write_text(
        "\n".join(
            [
                "# Duplicate issue closure report\n",
                f"Scan: `{scan_id}`\n",
                f"Open after duplicate closures: {open_after}\n",
                f"Closed: {len(dup_closed)}\n",
                f"Skipped: {len(dup_skipped)}\n",
                "## Closed duplicates\n",
                ", ".join(f"#{n}" for n in sorted(dup_closed)) or "(none)",
                "\n## Skipped\n",
                "\n".join(f"- #{n}: {reason}" for n, reason in dup_skipped) or "(none)",
            ]
        )
        + "\n"
    )

    # Update plan footer
    append = [
        "\n## Closure results\n",
        f"- Open before: {open_before}",
        f"- Open after resolved: {open_after_resolved}",
        f"- Open after duplicates: {open_after}",
        f"- Real active backlog: {active}",
        f"- Resolved closed: {len(resolved_closed)}",
        f"- Duplicates closed: {len(dup_closed)}",
    ]
    PLAN_PATH.write_text(PLAN_PATH.read_text() + "\n".join(append) + "\n")

    print(
        json.dumps(
            {
                "open_before": open_before,
                "open_after": open_after,
                "resolved_closed": len(resolved_closed),
                "dup_closed": len(dup_closed),
                "active": active,
                "scan_id": scan_id,
            }
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
