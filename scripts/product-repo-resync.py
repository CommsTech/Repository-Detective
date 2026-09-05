#!/usr/bin/env python3
"""Product repo full rescan + external_issues reconciliation (single repo only)."""

from __future__ import annotations

import json
import os
import sqlite3
import subprocess
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"
OWNER, REPO = "commstech", "Repository-Detective"
REPO_ID = 1
API = os.environ.get("RD_API_BASE", "http://127.0.0.1:8081/api/v1")


def load_env() -> tuple[str, str, str]:
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", "https://git.commsnet.org").rstrip("/")
    api_key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("REPOSITORY_DETECTIVE_API_KEY", "")
    env_file = ROOT / ".env"
    if env_file.exists():
        for line in env_file.read_text().splitlines():
            if "=" not in line or line.strip().startswith("#"):
                continue
            k, _, v = line.partition("=")
            k, v = k.strip(), v.strip().strip('"').strip("'")
            if k == "REPOSITORY_DETECTIVE_GITEA_TOKEN" and not token:
                token = v
            if k == "REPOSITORY_DETECTIVE_GITEA_URL":
                base = v.rstrip("/")
            if k in ("REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_API_KEY") and not api_key:
                api_key = v
    if not token:
        sys.exit("REPOSITORY_DETECTIVE_GITEA_TOKEN required")
    if not api_key:
        sys.exit("REPOSITORY_DETECTIVE_API_KEY required")
    return token, base, api_key


def gitea(base: str, token: str, method: str, path: str, body: dict | None = None):
    url = f"{base}/api/v1{path}"
    data = json.dumps(body).encode() if body else None
    headers = {"Authorization": f"token {token}"}
    if data:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=120) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else None


def api(api_key: str, method: str, path: str, body: dict | None = None):
    url = f"{API}{path}"
    data = json.dumps(body).encode() if body else None
    headers = {"X-Repository-Detective-API-Key": api_key}
    if data:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=300) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else None


def recon(api_key: str) -> dict:
    return api(api_key, "GET", f"/repos/{REPO_ID}/reconciliation")


def fetch_gitea_open(base: str, token: str) -> list[dict]:
    issues: list[dict] = []
    page = 1
    while True:
        batch = gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}/issues?state=open&type=issues&limit=50&page={page}")
        if not batch:
            break
        issues.extend(batch)
        if len(batch) < 50:
            break
        page += 1
    return issues


def sync_external_issues(base: str, token: str) -> int:
    """Mark DB external_issues closed when Gitea issue is closed or missing."""
    open_gitea = {i["number"] for i in fetch_gitea_open(base, token)}
    repaired = 0
    sql_close = """
    UPDATE external_issues SET state='closed', updated_at=datetime('now')
    WHERE forge_type='gitea' AND issue_number=? AND state='open'
    """
    try:
        conn = sqlite3.connect(DB)
    except sqlite3.OperationalError:
        out = subprocess.run(
            ["docker", "exec", "repository-detective", "sqlite3", "/app/data/repository-detective.db",
             "SELECT issue_number FROM external_issues WHERE state='open'"],
            capture_output=True, text=True, timeout=30,
        )
        nums = [int(x) for x in out.stdout.split() if x.strip().isdigit()]
        for num in nums:
            if num not in open_gitea:
                subprocess.run(
                    ["docker", "exec", "repository-detective", "sqlite3", "/app/data/repository-detective.db", sql_close, str(num)],
                    check=False, timeout=30,
                )
                repaired += 1
        return repaired

    cur = conn.cursor()
    cur.execute("SELECT issue_number FROM external_issues WHERE state='open'")
    for (num,) in cur.fetchall():
        if num not in open_gitea:
            cur.execute(sql_close, (num,))
            repaired += 1
    conn.commit()
    conn.close()
    return repaired


def wait_scan(api_key: str, want_id: str, timeout: int = 1800) -> dict:
    start = time.time()
    while time.time() - start < timeout:
        if want_id:
            try:
                s = api(api_key, "GET", f"/scans/{want_id}")
                if s.get("status") == "completed":
                    return s
            except urllib.error.HTTPError:
                pass
        else:
            scans = api(api_key, "GET", f"/repos/{REPO_ID}/scans?limit=5")
            items = scans if isinstance(scans, list) else scans.get("scans", scans.get("items", []))
            for s in items:
                if s.get("status") == "completed":
                    return s
        time.sleep(15)
    raise TimeoutError("scan did not complete in time")


def main() -> int:
    execute = "--execute" in sys.argv
    token, gitea_base, api_key = load_env()
    before_recon = recon(api_key)
    gitea_before = len(fetch_gitea_open(gitea_base, token))
    prev_scan = before_recon.get("latest_scan_id", "")

    print(json.dumps({"phase": "baseline", "gitea_open": gitea_before, "recon": before_recon}, indent=2))

    if not execute:
        print("Dry run — pass --execute to trigger rescan and sync")
        return 0

    body = {
        "owner": OWNER,
        "repository": REPO,
        "ref": "main",
        "scan_profile": "maintainer_deep",
        "report_only_dry_run": False,
    }
    trigger = api(api_key, "POST", "/analyze", body)
    print("triggered:", json.dumps(trigger, indent=2))
    triggered_id = trigger.get("scan_id", "")

    completed = wait_scan(api_key, triggered_id or prev_scan)
    scan_id = triggered_id or completed.get("id") or completed.get("scan_id")
    time.sleep(10)

    repaired = sync_external_issues(gitea_base, token)
    try:
        api(api_key, "POST", f"/repos/{REPO_ID}/reconcile-issues", {})
    except urllib.error.HTTPError as e:
        print("reconcile warning:", e)

    after_recon = recon(api_key)
    gitea_after = len(fetch_gitea_open(gitea_base, token))

    report = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "scan_id": scan_id,
        "gitea_open_before": gitea_before,
        "gitea_open_after": gitea_after,
        "active_present_before": before_recon.get("active_present_open"),
        "active_present_after": after_recon.get("active_present_open"),
        "forge_open_db_before": before_recon.get("forge_open_issues"),
        "forge_open_db_after": after_recon.get("forge_open_issues"),
        "stale_rows_repaired": repaired,
    }
    out = ROOT / "docs/dogfood-reports/product-rescan-and-issue-sync-report.md"
    out.write_text(
        "# Product rescan and issue sync report\n\n"
        f"Generated: {report['timestamp']}\n\n"
        + "\n".join(f"- **{k}:** {v}" for k, v in report.items())
        + "\n"
    )
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
