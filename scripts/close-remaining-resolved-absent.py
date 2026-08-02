#!/usr/bin/env python3
"""Close remaining resolved-absent Gitea issues with scan evidence."""

from __future__ import annotations

import json
import os
import sqlite3
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPORT = ROOT / "docs/dogfood-reports/batch4b-resolved-absent-closeout-report.md"
DB = ROOT / "data/repository-detective.db"
SCAN_ID = "db2d7061eaac8eb0"
OWNER, REPO = "commstech", "Repository-Detective"


def load_env() -> tuple[str, str]:
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", "https://git.commsnet.org").rstrip("/")
    for line in (ROOT / ".env").read_text().splitlines() if (ROOT / ".env").exists() else []:
        if "=" not in line or line.strip().startswith("#"):
            continue
        k, _, v = line.partition("=")
        k, v = k.strip(), v.strip().strip('"').strip("'")
        if k == "REPOSITORY_DETECTIVE_GITEA_TOKEN" and not token:
            token = v
        if k == "REPOSITORY_DETECTIVE_GITEA_URL":
            base = v.rstrip("/")
    if not token:
        sys.exit("REPOSITORY_DETECTIVE_GITEA_TOKEN required")
    return token, base


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


def extract_fingerprint(body: str) -> str:
    for line in (body or "").splitlines():
        line = line.strip().lstrip("- ")
        for m in ("Repository Detective fingerprint:", "Repository Detective fingerprint:"):
            if line.startswith(m):
                return line[len(m) :].strip()
    return ""


def scanner_for(source: str) -> str:
    s = (source or "").lower()
    for key in ("gosec", "gitleaks", "semgrep", "staticcheck", "govulncheck", "health", "hadolint", "checkov", "static", "preinstall"):
        if key in s:
            return key
    return s.split("-")[0] if s else ""


def main() -> int:
    token, base = load_env()
    open_before = int(gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}")["open_issues_count"])

    conn = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    cur = conn.cursor()
    cur.execute(
        "SELECT DISTINCT f.fingerprint FROM finding_instances fi JOIN findings f ON f.id=fi.finding_id WHERE fi.scan_id=? AND f.repository_id=1",
        (SCAN_ID,),
    )
    in_scan = {r[0] for r in cur.fetchall()}
    cur.execute("SELECT scanner_name, status FROM scanner_results WHERE scan_id=?", (SCAN_ID,))
    scanners = {n.lower(): s for n, s in cur.fetchall()}
    cur.execute(
        """
        SELECT ei.issue_number, f.fingerprint, f.source, f.rule_id, ei.finding_id
        FROM external_issues ei
        JOIN findings f ON f.id = ei.finding_id
        WHERE f.repository_id=1 AND ei.state='open'
        """
    )
    mapped = {row[0]: row for row in cur.fetchall()}

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

    candidates = []
    for issue in issues:
        num = issue["number"]
        fp = extract_fingerprint(issue.get("body") or "")
        labels = " ".join(lb.get("name", "") for lb in issue.get("labels", []))
        if "needs-human-review" in labels:
            continue
        if num not in mapped or not fp or fp in in_scan:
            continue
        src = mapped[num][2]
        sc = scanner_for(src)
        st = scanners.get(sc, "")
        if sc and st and st.lower() in ("failed", "error", "timeout", "skipped"):
            continue
        if "needs-human-review" in labels or "repository-detective/needs-human-review" in labels:
            continue
        candidates.append((num, fp, sc, st or "ok", mapped[num][3]))

    closed, skipped = [], []
    for num, fp, sc, st, rule in candidates:
        body = (
            f"Repository Detective **evidence closure** (resolved absent).\n\n"
            f"- Scan ID: `{SCAN_ID}`\n"
            f"- Scanner/check: `{sc}` (status: {st})\n"
            f"- Rule: `{rule}`\n"
            f"- Fingerprint `{fp}` absent from latest persisted scan\n"
            f"- Lifecycle: `external_issue_closed_resolved_verified`\n"
        )
        try:
            gitea(base, token, "POST", f"/repos/{OWNER}/{REPO}/issues/{num}/comments", {"body": body})
            gitea(base, token, "PATCH", f"/repos/{OWNER}/{REPO}/issues/{num}", {"state": "closed"})
            gitea(base, token, "POST", f"/repos/{OWNER}/{REPO}/issues/{num}/labels", {"labels": ["repository-detective/resolved-verified"]})
            closed.append(num)
        except urllib.error.HTTPError as e:
            skipped.append((num, e.read().decode(errors="replace")[:200]))

    open_after = int(gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}")["open_issues_count"])
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    REPORT.parent.mkdir(parents=True, exist_ok=True)
    REPORT.write_text(
        "\n".join(
            [
                "# Batch 4b resolved-absent closeout report\n",
                f"Generated: {now}\n",
                f"Scan: `{SCAN_ID}`\n",
                f"Open before: {open_before}\n",
                f"Open after: {open_after}\n",
                f"Candidates found: {len(candidates)}\n",
                f"Closed: {len(closed)}\n",
                f"Skipped: {len(skipped)}\n",
                "## Closed\n",
                ", ".join(f"#{n}" for n in sorted(closed)) or "(none)",
                "\n## Skipped\n",
                "\n".join(f"- #{n}: {r}" for n, r in skipped) or "(none)",
            ]
        )
        + "\n"
    )
    print(json.dumps({"open_before": open_before, "open_after": open_after, "closed": len(closed), "candidates": len(candidates)}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
