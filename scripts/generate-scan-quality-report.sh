#!/usr/bin/env bash
# Generate scan-quality JSON from live SQLite (offline from running API).
# Usage: ./scripts/generate-scan-quality-report.sh [path-to-repository-detective.db] [output.json]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB="${1:-$ROOT/data/repository-detective.db}"
OUT="${2:-$ROOT/docs/dogfood-reports/scan-quality-first-39-repos.json}"

if [ ! -f "$DB" ]; then
  echo "database not found: $DB" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"

# Build consolidated JSON with python (works with or without sqlite3 CLI on host)
python3 - "$DB" "$OUT" <<'PY'
import json, sqlite3, sys
from pathlib import Path

db, out = sys.argv[1], sys.argv[2]
conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row

def counts(sql, col=1):
    return {r[0] or "unknown": r[col] for r in conn.execute(sql)}

def table_exists(name):
    return conn.execute(
        "SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?", (name,)
    ).fetchone()[0] > 0

repos_scanned = conn.execute(
    "SELECT COUNT(DISTINCT repository_id) FROM scans WHERE status='completed'"
).fetchone()[0]
total_scans = conn.execute("SELECT COUNT(*) FROM scans").fetchone()[0]
total_findings = conn.execute("SELECT COUNT(*) FROM findings").fetchone()[0]
open_findings = conn.execute("SELECT COUNT(*) FROM findings WHERE status='open'").fetchone()[0]
suppressed = conn.execute(
    "SELECT COUNT(*) FROM findings WHERE status IN ('suppressed','false_positive')"
).fetchone()[0]
false_pos = conn.execute(
    "SELECT COUNT(*) FROM findings WHERE status='false_positive'"
).fetchone()[0]
actionable = open_findings
ratio = (actionable / total_findings) if total_findings else 0.0

report = {
    "repos_scanned": repos_scanned,
    "total_scans": total_scans,
    "total_findings": total_findings,
    "open_findings": open_findings,
    "suppressed_findings": suppressed,
    "false_positive_findings": false_pos,
    "findings_by_severity": counts(
        "SELECT LOWER(severity), COUNT(*) FROM findings GROUP BY LOWER(severity)"
    ),
    "findings_by_category": counts(
        "SELECT LOWER(category), COUNT(*) FROM findings GROUP BY LOWER(category)"
    ),
    "findings_by_source": counts(
        "SELECT LOWER(source), COUNT(*) FROM findings GROUP BY LOWER(source)"
    ),
    "external_issues_open": conn.execute(
        "SELECT COUNT(*) FROM external_issues WHERE state='open'"
    ).fetchone()[0],
    "remediation_plans_generated": conn.execute(
        "SELECT COUNT(*) FROM remediation_plans"
    ).fetchone()[0],
    "patch_attempts_opened": conn.execute(
        "SELECT COUNT(*) FROM patch_attempts WHERE pull_request_number IS NOT NULL AND pull_request_number > 0"
    ).fetchone()[0],
    "patch_attempts_verified": conn.execute(
        "SELECT COUNT(*) FROM closure_evidence WHERE status='resolved_verified'"
    ).fetchone()[0],
    "scanner_failures": conn.execute(
        "SELECT COUNT(*) FROM scanner_results WHERE status IN ('failed','error','timeout')"
    ).fetchone()[0],
    "failed_scans": conn.execute(
        "SELECT COUNT(*) FROM scans WHERE status='failed'"
    ).fetchone()[0],
    "active_suppressions": conn.execute(
        "SELECT COUNT(*) FROM finding_suppressions WHERE active=1"
    ).fetchone()[0] if table_exists("finding_suppressions") else 0,
    "schema_migrations": [r[0] for r in conn.execute("SELECT version FROM schema_migrations ORDER BY version")],
    "repos_with_no_findings": conn.execute("""
        SELECT COUNT(*) FROM (
          SELECT repository_id FROM scans WHERE status='completed'
          GROUP BY repository_id
          HAVING SUM(CAST(json_extract(summary_json,'$.issues_found') AS INTEGER)) = 0
        )
    """).fetchone()[0],
    "repos_with_critical_high": conn.execute("""
        SELECT COUNT(DISTINCT repository_id) FROM findings
        WHERE status='open' AND LOWER(severity) IN ('critical','high')
    """).fetchone()[0],
    "actionable_findings": actionable,
    "actionable_ratio": round(ratio, 4),
    "top_rules": [
        dict(r)
        for r in conn.execute("""
            SELECT rule_id, source, COUNT(*) AS cnt FROM findings
            WHERE rule_id != '' GROUP BY rule_id, source ORDER BY cnt DESC LIMIT 15
        """)
    ],
    "top_sources": [
        dict(r)
        for r in conn.execute("""
            SELECT source, COUNT(*) AS cnt FROM findings
            WHERE source != '' GROUP BY source ORDER BY cnt DESC LIMIT 10
        """)
    ],
    "scanner_failure_by_tool": [
        dict(r)
        for r in conn.execute("""
            SELECT scanner_name, status, COUNT(*) AS cnt FROM scanner_results
            WHERE status IN ('failed','error','timeout','binary_missing')
            GROUP BY scanner_name, status ORDER BY cnt DESC LIMIT 20
        """)
    ],
}
Path(out).parent.mkdir(parents=True, exist_ok=True)
Path(out).write_text(json.dumps(report, indent=2) + "\n")
print(f"wrote {out}")
PY
