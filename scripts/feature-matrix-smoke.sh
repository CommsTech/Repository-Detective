#!/usr/bin/env bash
# Full feature matrix smoke for Repository Detective (release gate helper).
# Exercises APIs/UI for major capabilities without filing forge issues.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BASE="${REPOSITORY_DETECTIVE_PUBLIC_URL:-http://127.0.0.1:8081}"
BASE="${BASE%/}"
KEY="${REPOSITORY_DETECTIVE_API_KEY}"
REPO_ID="${RD_MATRIX_REPO_ID:-1}"
REPORT="${RD_FEATURE_MATRIX_REPORT:-docs/dogfood-reports/feature-matrix-$(date -u +%Y%m%dT%H%M%SZ).md}"

pass=0; fail=0; warn=0; skip=0
mkdir -p "$(dirname "$REPORT")"
{
  echo "# Feature matrix report"
  echo
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "Base: ${BASE}"
  echo "Repo: ${REPO_ID}"
  echo
  echo "| Check | Result | Notes |"
  echo "|-------|--------|-------|"
} >"$REPORT.tmp"

auth=(-H "Authorization: Bearer ${KEY}" -H "X-Repository-Detective-API-Key: ${KEY}")

record() {
  printf '| `%s` | %s | %s |\n' "$1" "$2" "$3" >>"$REPORT.tmp"
  printf '==> %s -> %s (%s)\n' "$1" "$2" "$3"
}

check_http() {
  local name=$1 url=$2 expect=${3:-200}
  local code
  code=$(curl -sS -o /tmp/rd-matrix-body -w '%{http_code}' -m 30 "${auth[@]}" "$url" || echo 000)
  if [[ "$code" == "$expect" ]]; then
    ((pass++)) || true
    record "$name" "pass" "HTTP $code"
  else
    ((fail++)) || true
    record "$name" "fail" "HTTP $code (want $expect)"
  fi
}

json_field() {
  python3 - "$1" <<'PY'
import json,sys
path=sys.argv[1]
d=json.load(sys.stdin)
cur=d
for part in path.split('.'):
  if isinstance(cur, dict):
    cur=cur.get(part)
  else:
    cur=None
    break
print(cur if cur is not None else '')
PY
}

echo "==> Health"
check_http "health" "${BASE}/health"
ready=$(python3 -c "import json; d=json.load(open('/tmp/rd-matrix-body')); print(d.get('ready',''))" 2>/dev/null || true)
tools=$(python3 -c "import json; d=json.load(open('/tmp/rd-matrix-body')); print(d.get('tools_summary',{}).get('available_count',0))" 2>/dev/null || echo 0)
if [[ "$ready" == "True" || "$ready" == "true" ]]; then
  ((pass++)) || true; record "ready" "pass" "ready=$ready tools=$tools"
else
  ((fail++)) || true; record "ready" "fail" "ready=$ready"
fi

echo "==> Core UI"
for path in /ui /ui/repos /ui/repos/${REPO_ID} /ui/repos/${REPO_ID}/settings /ui/repos/${REPO_ID}/containers /ui/repos/${REPO_ID}/sbom /ui/repos/${REPO_ID}/graph /ui/repos/${REPO_ID}/report /ui/repos/${REPO_ID}/reconcile /ui/repos/${REPO_ID}/scan /ui/findings /ui/configure /ui/learning /ui/health /ui/preinstall /ui/reports /ui/projects; do
  check_http "ui:${path}" "${BASE}${path}"
done

echo "==> API surfaces"
check_http "api:repos" "${BASE}/api/v1/repos"
check_http "api:findings_open" "${BASE}/api/v1/findings?repo_id=${REPO_ID}&status=open&suppressed=false&limit=5"
check_http "api:scans" "${BASE}/api/v1/repos/${REPO_ID}/scans?limit=5"
check_http "api:status" "${BASE}/api/v1/status"

if [[ -z "$KEY" ]]; then
  ((skip++)) || true
  record "manual_scan" "skip" "no API key"
else
  echo "==> Manual report-only scan"
  payload=$(printf '{"owner":"commstech","repository":"Repository-Detective","ref":"main","report_only_dry_run":true}')
  code=$(curl -sS -o /tmp/rd-matrix-scan.json -w '%{http_code}' -m 30 \
    -H "Content-Type: application/json" "${auth[@]}" \
    -d "$payload" "${BASE}/api/v1/analyze" || echo 000)
  if [[ "$code" == "200" ]]; then
    scan_id=$(python3 -c "import json; print(json.load(open('/tmp/rd-matrix-scan.json')).get('scan_id',''))" 2>/dev/null || true)
    ((pass++)) || true
    record "manual_scan_start" "pass" "HTTP 200 scan_id=${scan_id}"
  else
    ((fail++)) || true
    record "manual_scan_start" "fail" "HTTP $code"
  fi
fi

{
  cat "$REPORT.tmp"
  echo
  echo "## Summary"
  echo "- Pass: ${pass}"
  echo "- Fail: ${fail}"
  echo "- Warn: ${warn}"
  echo "- Skip: ${skip}"
} >"$REPORT"
rm -f "$REPORT.tmp"
echo "Report: $REPORT"
[[ "$fail" -eq 0 ]]
