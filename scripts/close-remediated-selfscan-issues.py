#!/usr/bin/env python3
"""Close remediations-linked Repository Detective forge issues (self-scan FP / calibrated / fixed)."""

from __future__ import annotations

import json
import os
import sqlite3
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PLAN_PATH = Path("/tmp/rd_close_plan.json")
DB_PATH = ROOT / "data/repository-detective.db"
OWNER = "commstech"
REPO = "repository-detective"
FORGE_BASE = "https://git.commsnet.org"


def load_token() -> tuple[str, str]:
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", FORGE_BASE).rstrip("/")
    env_path = ROOT / ".env"
    if env_path.exists():
        for line in env_path.read_text().splitlines():
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
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": f"token {token}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=120) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else None


def label_ids(base: str, token: str) -> dict[str, int]:
    out: dict[str, int] = {}
    page = 1
    while True:
        batch = gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}/labels?limit=50&page={page}")
        if not batch:
            break
        for lab in batch:
            out[lab["name"]] = int(lab["id"])
        if len(batch) < 50:
            break
        page += 1
    return out


def close_one(
    base: str,
    token: str,
    row: dict,
    labels: dict[str, int],
) -> tuple[bool, str]:
    num = row["num"]
    comment = row.get("comment") or "Closing: remediations-linked Repository Detective finding."
    try:
        gitea(
            base,
            token,
            "POST",
            f"/repos/{OWNER}/{REPO}/issues/{num}/comments",
            {"body": comment},
        )
        gitea(
            base,
            token,
            "PATCH",
            f"/repos/{OWNER}/{REPO}/issues/{num}",
            {"state": "closed"},
        )
        add_names = list(row.get("labels_add") or [])
        # Always add status/done when available.
        if "status/done" not in add_names:
            add_names.append("status/done")
        ids = [labels[n] for n in add_names if n in labels]
        if ids:
            gitea(
                base,
                token,
                "POST",
                f"/repos/{OWNER}/{REPO}/issues/{num}/labels",
                {"labels": ids},
            )
        return True, "ok"
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode(errors="replace")
        return False, f"HTTP {exc.code}: {detail[:300]}"
    except Exception as exc:  # noqa: BLE001
        return False, str(exc)


def update_external_issues(issue_numbers: list[int]) -> int:
    """Mark matching repo_id=1 external_issues rows closed."""
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    conn = sqlite3.connect(str(DB_PATH))
    cur = conn.cursor()
    updated = 0
    for num in issue_numbers:
        cur.execute(
            """
            UPDATE external_issues
            SET state = 'closed', updated_at = ?
            WHERE state = 'open'
              AND issue_number = ?
              AND finding_id IN (SELECT id FROM findings WHERE repository_id = 1)
            """,
            (now, num),
        )
        updated += cur.rowcount
    conn.commit()
    conn.close()
    return updated


def sync_stale_fp_mappings() -> list[int]:
    """Close DB mappings for FP/suppressed findings whose forge issues are already gone."""
    conn = sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)
    cur = conn.cursor()
    cur.execute(
        """
        SELECT DISTINCT ei.issue_number
        FROM external_issues ei
        JOIN findings f ON f.id = ei.finding_id
        WHERE f.repository_id = 1
          AND ei.state = 'open'
          AND f.status IN ('false_positive', 'suppressed')
        """
    )
    nums = [int(r[0]) for r in cur.fetchall()]
    conn.close()
    return nums


def main() -> int:
    if not PLAN_PATH.exists():
        sys.exit(f"missing plan {PLAN_PATH}")
    rows = json.loads(PLAN_PATH.read_text())
    to_close = [r for r in rows if r.get("decision") == "CLOSE"]
    token, base = load_token()
    labels = label_ids(base, token)

    repo = gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}")
    open_before = int(repo.get("open_issues_count", 0))
    print(f"open_before={open_before} planned_close={len(to_close)}")

    closed: list[dict] = []
    errors: list[dict] = []
    for i, row in enumerate(sorted(to_close, key=lambda r: r["num"])):
        ok, msg = close_one(base, token, row, labels)
        if ok:
            closed.append(row)
            print(f"CLOSED #{row['num']} ({row['reason']})")
        else:
            errors.append({"num": row["num"], "error": msg, "reason": row.get("reason")})
            print(f"ERROR #{row['num']}: {msg}")
        # gentle pacing for Gitea
        if (i + 1) % 10 == 0:
            time.sleep(0.5)

    closed_nums = [r["num"] for r in closed]
    stale = sync_stale_fp_mappings()
    db_nums = sorted(set(closed_nums) | set(stale))
    db_updated = update_external_issues(db_nums)
    print(f"db_external_issues_updated={db_updated} (nums={len(db_nums)} incl stale FP {stale})")

    repo = gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}")
    open_after = int(repo.get("open_issues_count", 0))

    # recount labeled open
    labeled_open = 0
    page = 1
    while True:
        batch = gitea(
            base,
            token,
            "GET",
            f"/repos/{OWNER}/{REPO}/issues?state=open&type=issues&labels=repository-detective&limit=50&page={page}",
        )
        if not batch:
            break
        labeled_open += len(batch)
        if len(batch) < 50:
            break
        page += 1

    report = {
        "open_before": open_before,
        "planned_close": len(to_close),
        "closed_count": len(closed),
        "open_after": open_after,
        "labeled_open_after": labeled_open,
        "db_updated": db_updated,
        "stale_fp_issue_nums": stale,
        "sample_closed_urls": [r["html"] for r in closed[:8]],
        "errors": errors,
        "kept_count": sum(1 for r in rows if r.get("decision") == "KEEP"),
        "kept_sample": [
            {"num": r["num"], "rule": r["rule"], "reason": r["reason"], "html": r["html"]}
            for r in rows
            if r.get("decision") == "KEEP"
        ][:12],
    }
    Path("/tmp/rd_close_report.json").write_text(json.dumps(report, indent=2))
    print(json.dumps(report, indent=2))
    return 0 if not errors else 1


if __name__ == "__main__":
    raise SystemExit(main())
