#!/usr/bin/env bash
# Dogfood loop: scan Repository-Detective, file Gitea issues, close out calibrated noise.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BASE="${REPOSITORY_DETECTIVE_PUBLIC_URL:-http://127.0.0.1:8081}"
BASE="${BASE%/}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-${BUGBOT_API_KEY:-}}"
OWNER="${DOGFOOD_OWNER:-commstech}"
REPO="${DOGFOOD_REPO:-Repository-Detective}"
REPORT_ONLY="${REPORT_ONLY:-false}"

if [[ -z "$API_KEY" ]]; then
  echo "Set REPOSITORY_DETECTIVE_API_KEY in .env" >&2
  exit 1
fi

auth=(-H "X-Repository-Detective-API-Key: ${API_KEY}" -H "Authorization: Bearer ${API_KEY}")

echo "==> Reap orphaned started scans for repo 1"
python3 <<'PY'
import sqlite3, time
db = "data/repository-detective.db"
conn = sqlite3.connect(db, timeout=30)
now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
cur = conn.execute(
    """
    UPDATE scans SET status='failed', finished_at=?, error=?
    WHERE repository_id=1 AND status='started' AND finished_at IS NULL
    """,
    (now, "reaped before dogfood release scan"),
)
conn.commit()
print("reaped", cur.rowcount)
PY

payload=$(REPORT_ONLY="$REPORT_ONLY" python3 - <<'PY'
import json, os
report_only = os.environ.get("REPORT_ONLY", "false").lower() in ("1", "true", "yes")
print(json.dumps({
    "owner": os.environ.get("DOGFOOD_OWNER", "commstech"),
    "repository": os.environ.get("DOGFOOD_REPO", "Repository-Detective"),
    "ref": "main",
    "report_only_dry_run": report_only,
}))
PY
)

echo "==> Trigger analyze (report_only=${REPORT_ONLY})"
code=$(curl -sS -o /tmp/rd-dogfood-analyze.json -w "%{http_code}" \
  -H "Content-Type: application/json" "${auth[@]}" \
  -d "$payload" "${BASE}/api/v1/analyze")
echo "HTTP ${code}"
cat /tmp/rd-dogfood-analyze.json
echo ""

scan_id=$(python3 -c "import json; print(json.load(open('/tmp/rd-dogfood-analyze.json')).get('scan_id',''))")
[[ -n "$scan_id" ]] || { echo "no scan_id"; exit 1; }

echo "==> Poll scan ${scan_id}"
for i in $(seq 1 120); do
  st=$(python3 -c "import sqlite3; print(sqlite3.connect('data/repository-detective.db').execute('SELECT status FROM scans WHERE id=?',('${scan_id}',)).fetchone()[0])")
  echo "  poll $i status=$st"
  if [[ "$st" != "started" ]]; then
    break
  fi
  sleep 15
done

python3 <<PY
import sqlite3
r = sqlite3.connect("data/repository-detective.db").execute(
    "SELECT status, error FROM scans WHERE id=?", ("${scan_id}",)
).fetchone()
print("final", r)
open_count = sqlite3.connect("data/repository-detective.db").execute(
    "SELECT count(*) FROM findings WHERE repository_id=1 AND status='open'"
).fetchone()[0]
print("open_findings", open_count)
PY

echo "==> Closeout calibrated noise"
export REPOSITORY_DETECTIVE_API_KEY="$API_KEY"
python3 scripts/closeout-repo1-findings.py

python3 <<'PY'
import sqlite3
conn = sqlite3.connect("data/repository-detective.db")
open_count = conn.execute("SELECT count(*) FROM findings WHERE repository_id=1 AND status='open'").fetchone()[0]
high = conn.execute("SELECT count(*) FROM findings WHERE repository_id=1 AND status='open' AND severity IN ('critical','high')").fetchone()[0]
print(f"remaining open={open_count} high+critical={high}")
PY
