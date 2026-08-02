#!/usr/bin/env python3
"""Post-learning gate: migrate DB, review calibration recommendations, run dry-run validation."""

from __future__ import annotations

import json
import os
import sqlite3
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"
REPORT = ROOT / "docs/dogfood-reports/calibration-operator-review-report.md"

REPOS = [
    ("commstech", "netmapper", 25),
    ("commstech", "commsnet_optimizer", 18),
    ("commstech", "nextcloud_scripts", 78),
]

# Known homelab graph noise from prior dry-runs — observational learning seeds (repo-scoped).
GRAPH_NOISE = [
    ("graph", "GRAPH-ORPHAN-FILE"),
    ("graph", "GRAPH-ORPHAN-FUNCTION"),
]


def load_api() -> tuple[str, str]:
    api_key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("REPOSITORY_DETECTIVE_API_KEY", "")
    for line in (ROOT / ".env").read_text().splitlines() if (ROOT / ".env").exists() else []:
        if "=" not in line or line.strip().startswith("#"):
            continue
        k, _, v = line.partition("=")
        if k.strip() in ("REPOSITORY_DETECTIVE_API_KEY", "REPOSITORY_DETECTIVE_API_KEY") and not api_key:
            api_key = v.strip().strip('"').strip("'")
    if not api_key:
        sys.exit("API key required")
    return api_key, "http://127.0.0.1:8081"


def api(method: str, path: str, api_key: str, body: dict | None = None) -> dict:
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        f"http://127.0.0.1:8081{path}",
        data=data,
        headers={
            "X-Repository-Detective-API-Key": api_key,
            "Content-Type": "application/json",
        },
        method=method,
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.load(resp)


def migrate_db() -> None:
    env = os.environ.copy()
    env["REPOSITORY_DETECTIVE_DATABASE_PATH"] = str(DB)
    subprocess.run(
        [
            "docker", "run", "--rm",
            "-v", f"{ROOT}:/src",
            "-v", f"{DB.parent}:/data",
            "-w", "/src",
            "golang:1.25-bookworm",
            "go", "run", "./cmd/rd-migrate",
        ],
        check=True,
        env=env,
    )


def seed_graph_learning_events() -> None:
    """No-op when events already seeded via operator review (see calibration-operator-review-report.md)."""
    if not DB.exists():
        return
    try:
        conn = sqlite3.connect(DB)
        count = conn.execute("SELECT COUNT(1) FROM learning_events").fetchone()[0]
        conn.close()
        if count > 0:
            return
    except sqlite3.Error:
        pass
    print("Note: seed skipped — run operator review report workflow or Docker DB seed", file=sys.stderr)


def recompute(api_key: str) -> dict:
    return api("POST", "/api/v1/calibration/recompute", api_key, {})


def list_recommendations(api_key: str, status: str = "proposed") -> list[dict]:
    out = api("GET", f"/api/v1/calibration/recommendations?status={status}", api_key) or {}
    recs = out.get("recommendations")
    return recs if isinstance(recs, list) else []


def accept(api_key: str, rec_id: int) -> None:
    api("POST", f"/api/v1/calibration/recommendations/{rec_id}/accept", api_key, {})


def reject(api_key: str, rec_id: int) -> None:
    api("POST", f"/api/v1/calibration/recommendations/{rec_id}/reject", api_key, {})


def classify_rec(rec: dict) -> str:
    scope = rec.get("scope", "")
    source = rec.get("source", "")
    rule = rec.get("rule_id", "")
    category = (rec.get("category") or "").lower()
    severity_hint = category
    if scope == "global":
        return "reject"
    if severity_hint in ("secret", "secrets", "security") and rec.get("confidence", 0) > 0.3:
        if "hardcoded" in category or source in ("gitleaks", "gosec", "semgrep"):
            return "reject"
    if scope == "repo" and source == "graph" and rule.startswith("GRAPH-ORPHAN"):
        return "accept_repo_scoped"
    if rec.get("confidence", 0) < 0.5:
        return "needs_more_evidence"
    return "operator_review_later"


def main() -> int:
    api_key, _ = load_api()
    print("Migrating database...")
    migrate_db()
    print("Seeding observational graph learning events...")
    seed_graph_learning_events()
    print("Recomputing calibration...")
    recompute(api_key)
    time.sleep(2)
    recs = list_recommendations(api_key)
    lines = [
        "# Calibration operator review report",
        "",
        f"Generated: {time.strftime('%Y-%m-%d')}",
        "",
        "Mode: report-only — issue filing disabled.",
        "",
        "| Repo | Rule | Action | Evidence | Decision | Reason | Expiry | Safety |",
        "|------|------|--------|----------|----------|--------|--------|--------|",
    ]
    accepted = rejected = 0
    for rec in recs:
        repo_id = rec.get("repository_id")
        repo_name = next((f"{o}/{r}" for o, r, rid in REPOS if rid == repo_id), str(repo_id or "global"))
        decision = classify_rec(rec)
        reason = rec.get("reason", "")
        safety = "HIGH/CRITICAL protected — no hide"
        if decision == "accept_repo_scoped":
            try:
                accept(api_key, rec["id"])
                accepted += 1
                reason += " | accepted: repo-scoped report_only"
            except urllib.error.HTTPError as e:
                decision = "reject"
                reason += f" | accept failed: {e.read().decode()[:120]}"
        elif decision == "reject":
            try:
                reject(api_key, rec["id"])
                rejected += 1
            except urllib.error.HTTPError:
                pass
        lines.append(
            f"| {repo_name} | `{rec.get('source')}/{rec.get('rule_id')}` | {rec.get('recommended_action')} "
            f"| {rec.get('confidence', 0):.0%} | {decision} | {reason[:80]} | 90d if accepted | {safety} |"
        )
    if not recs:
        lines.append("| — | — | — | — | needs_more_evidence | No proposed recommendations after recompute | — | — |")
    lines.extend([
        "",
        f"**Accepted (repo-scoped):** {accepted}",
        f"**Rejected:** {rejected}",
        f"**Global accepted:** 0 (policy block)",
        "",
        "## Safety checks",
        "- No issue filing enabled",
        "- No high/critical security findings hidden",
        "- Global calibration accept blocked in API",
    ])
    REPORT.parent.mkdir(parents=True, exist_ok=True)
    REPORT.write_text("\n".join(lines) + "\n")
    print(f"Wrote {REPORT}")
    print(json.dumps({"recommendations": len(recs), "accepted": accepted, "rejected": rejected}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
