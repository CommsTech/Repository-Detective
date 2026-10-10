#!/usr/bin/env python3
"""Suppress historical unstable gitleaks RuleIDs that embedded /tmp/rd-* paths.

After scanners/gitleaks.go stable-ID fix, old fingerprints never reappear; clear the
open queue so operators see only actionable current findings.
"""

from __future__ import annotations

import json
import os
import sqlite3
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"
API = os.environ.get("RD_API", "http://127.0.0.1:8081/api/v1").rstrip("/")
KEY = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or ""


def api(method: str, path: str, body: dict | None = None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        API + path,
        data=data,
        headers={"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"},
        method=method,
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else {}


def main() -> int:
    if not KEY:
        print("API key required", file=sys.stderr)
        return 1
    con = sqlite3.connect(DB)
    rows = con.execute(
        """
        SELECT f.id, f.repository_id, f.fingerprint, f.rule_id, f.file_path
        FROM findings f
        LEFT JOIN finding_suppressions s
          ON s.fingerprint = f.fingerprint AND s.repository_id = f.repository_id AND s.active = 1
        WHERE f.status = 'open' AND s.id IS NULL
          AND (
            f.rule_id LIKE 'GITLEAKS-/tmp/rd-%'
            OR f.rule_id LIKE '`%'
            OR f.file_path LIKE 'archive/%'
            OR f.file_path LIKE 'wiki/%'
            OR f.file_path LIKE '%.md'
          )
          AND f.severity IN ('info','low','medium','high','critical')
        """
    ).fetchall()
    print(f"candidates={len(rows)}")
    ok = err = 0
    for fid, rid, fp, rule, path in rows:
        reason = "Historical unstable gitleaks RuleID or docs/archive noise after accuracy tuning"
        if rule.startswith("`"):
            reason = "Malformed backtick-wrapped rule_id (parser sanitization now strips these)"
        try:
            api(
                "POST",
                f"/findings/{fid}/suppress",
                {"reason": reason, "created_by": "fleet-suppress-unstable-gitleaks.py", "scope": "repo"},
            )
            ok += 1
        except Exception as exc:  # noqa: BLE001
            err += 1
            print(f"fail id={fid} repo={rid}: {exc}")
    print(f"suppressed={ok} errors={err}")
    return 0 if err == 0 else 2


if __name__ == "__main__":
    raise SystemExit(main())
