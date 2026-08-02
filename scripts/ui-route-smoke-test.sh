#!/usr/bin/env bash
# Crawl Repository Detective UI routes for HTTP status and basic content checks.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RD_PORT:-8081}"
BASE="${RD_BASE_URL:-http://127.0.0.1:${PORT}/ui}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY}"
REPORT="${RD_UI_SMOKE_REPORT:-docs/dogfood-reports/ui-route-smoke-report.md}"

pass=0
fail=0
warn=0

log() { printf '==> %s\n' "$*"; }
record() { printf '%s\n' "$1" >>"$REPORT.tmp"; }

mkdir -p "$(dirname "$REPORT")"
cat >"$REPORT.tmp" <<EOF
# UI route smoke report

Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Base: ${BASE}

| Route | Status | Notes |
|-------|--------|-------|
EOF

check_route() {
  local path=$1
  local expect=${2:-}
  local url="${BASE}${path}"
  local code body
  local curl_args=(-sS -o /tmp/rd-ui-body.html -w '%{http_code}' -m 45 --max-redirs 3)
  if [ -n "$API_KEY" ]; then
    curl_args+=(-H "X-Repository-Detective-API-Key: ${API_KEY}" -H "Authorization: Bearer ${API_KEY}")
  fi
  code=$(curl "${curl_args[@]}" "$url" || echo "000")
  body=$(head -c 4000 /tmp/rd-ui-body.html 2>/dev/null || true)
  local note="ok"
  if [ "$code" = "401" ] || [ "$code" = "403" ]; then
    note="auth required (set REPOSITORY_DETECTIVE_API_KEY)"
    ((warn++)) || true
  elif [ "$code" = "404" ]; then
    note="not found (route or resource may need deploy)"
    ((warn++)) || true
  elif [ "$code" != "200" ]; then
    note="unexpected status"
    ((fail++)) || true
  elif echo "$body" | grep -qiE 'panic|runtime error|stack trace'; then
    note="error page content"
    ((fail++)) || true
  elif [ -n "$expect" ] && ! echo "$body" | grep -qi "$expect"; then
    note="missing expected text: $expect"
    ((warn++)) || true
  else
    ((pass++)) || true
  fi
  record "| \`${path}\` | ${code} | ${note} |"
  log "${path} -> ${code} (${note})"
}

ROUTES=(
  "/"
  "/repos"
  "/repos/1"
  "/repos/1/settings"
  "/repos/1/containers"
  "/repos/1/sbom"
  "/repos/1/graph"
  "/repos/1/report"
  "/repos/1/reconcile"
  "/repos/1/scan"
  "/scans"
  "/findings"
  "/findings?repo_id=1&status=open"
  "/findings/1"
  "/configure"
  "/learning"
  "/health"
  "/preinstall"
  "/reports"
  "/projects"
)

for r in "${ROUTES[@]}"; do
  check_route "$r" "Repository Detective"
done

{
  echo
  echo "## Summary"
  echo "- Pass: ${pass}"
  echo "- Warn: ${warn}"
  echo "- Fail: ${fail}"
} >>"$REPORT.tmp"

mv "$REPORT.tmp" "$REPORT"
log "Report written to ${REPORT}"
if [ "$fail" -gt 0 ]; then exit 1; fi
if [ -z "$API_KEY" ]; then
  log "WARN: no API key — auth routes recorded as warn only"
fi
