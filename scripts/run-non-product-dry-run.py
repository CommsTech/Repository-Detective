#!/usr/bin/env python3
"""Run report-only dry-run scan and collect metrics."""

from __future__ import annotations

import json
import os
import sqlite3
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"


def load_env() -> tuple[str, str]:
    api_key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("REPOSITORY_DETECTIVE_API_KEY", "")
    for line in (ROOT / ".env").read_text().splitlines() if (ROOT / ".env").exists() else []:
        if "=" not in line or line.strip().startswith("#"):
            continue
        k, _, v = line.partition("=")
        k, v = k.strip(), v.strip().strip('"').strip("'")
        if k in ("REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_API_KEY") and not api_key:
            api_key = v
    if not api_key:
        sys.exit("API key required")
    return api_key, "http://localhost:8081"


def gitea_count(owner: str, repo: str) -> int:
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", "https://git.commsnet.org").rstrip("/")
    for line in (ROOT / ".env").read_text().splitlines() if (ROOT / ".env").exists() else []:
        if "=" not in line or line.strip().startswith("#"):
            continue
        k, _, v = line.partition("=")
        if k == "REPOSITORY_DETECTIVE_GITEA_TOKEN" and not token:
            token = v.strip().strip('"').strip("'")
        if k == "REPOSITORY_DETECTIVE_GITEA_URL":
            base = v.strip().strip('"').strip("'").rstrip("/")
    req = urllib.request.Request(
        f"{base}/api/v1/repos/{owner}/{repo}",
        headers={"Authorization": f"token {token}"},
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        return int(json.load(resp)["open_issues_count"])


def trigger_scan(api_key: str, base: str, owner: str, repo: str) -> None:
    body = json.dumps(
        {
            "owner": owner,
            "repository": repo,
            "ref": "main",
            "scan_profile": "standard_deterministic",
            "report_only_dry_run": True,
        }
    ).encode()
    req = urllib.request.Request(
        f"{base}/api/v1/analyze",
        data=body,
        headers={
            "X-Repository-Detective-API-Key": api_key,
            "Content-Type": "application/json",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        print(json.load(resp))


def wait_scan(owner: str, repo: str, prev_scans: set[str], timeout: int = 900) -> dict:
    full = f"{owner}/{repo}"
    start = time.time()
    while time.time() - start < timeout:
        conn = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
        row = conn.execute(
            """
            SELECT s.id, s.status, s.started_at, s.finished_at, s.summary_json
            FROM scans s JOIN repositories r ON r.id = s.repository_id
            WHERE r.full_name = ? ORDER BY s.started_at DESC LIMIT 1
            """,
            (full,),
        ).fetchone()
        conn.close()
        if row and row[0] not in prev_scans and row[1] == "completed":
            scan_id, _, started, finished, summary_raw = row
            summary = json.loads(summary_raw or "{}")
            conn = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
            inst = conn.execute("SELECT COUNT(*) FROM finding_instances WHERE scan_id=?", (scan_id,)).fetchone()[0]
            scanners = conn.execute(
                "SELECT scanner_name, status, findings_count FROM scanner_results WHERE scan_id=?",
                (scan_id,),
            ).fetchall()
            ext_new = conn.execute(
                """
                SELECT COUNT(*) FROM external_issues ei
                JOIN finding_instances fi ON fi.finding_id = ei.finding_id
                WHERE fi.scan_id = ?
                """,
                (scan_id,),
            ).fetchone()[0]
            conn.close()
            duration = ""
            if started and finished:
                duration = f"{started} -> {finished}"
            return {
                "scan_id": scan_id,
                "issues_found": summary.get("issues_found", 0),
                "persistence_status": summary.get("persistence_status"),
                "issue_sync_status": summary.get("issue_sync_status"),
                "dry_run_report_only": summary.get("dry_run_report_only"),
                "instances": inst,
                "scanners": scanners,
                "external_issues_linked": ext_new,
                "duration": duration,
                "graph_nodes": summary.get("graph_nodes"),
                "graph_edges": summary.get("graph_edges"),
            }
        time.sleep(15)
    raise SystemExit(f"timeout waiting for {full}")


def prev_scan_ids(owner: str, repo: str) -> set[str]:
    conn = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    ids = {
        r[0]
        for r in conn.execute(
            """
            SELECT s.id FROM scans s JOIN repositories r ON r.id = s.repository_id
            WHERE r.full_name = ?
            """,
            (f"{owner}/{repo}",),
        )
    }
    conn.close()
    return ids


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: run-non-product-dry-run.py OWNER REPO", file=sys.stderr)
        return 1
    owner, repo = sys.argv[1], sys.argv[2]
    api_key, base = load_env()
    open_before = gitea_count(owner, repo)
    prev = prev_scan_ids(owner, repo)
    print(f"open_issues_before={open_before}")
    trigger_scan(api_key, base, owner, repo)
    metrics = wait_scan(owner, repo, prev)
    open_after = gitea_count(owner, repo)
    metrics["open_issues_before"] = open_before
    metrics["open_issues_after"] = open_after
    metrics["issues_created"] = open_after - open_before
    print(json.dumps(metrics, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
