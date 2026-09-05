#!/usr/bin/env bash
# Operator smoke test against a running Repository Detective instance.
# Usage:
#   export REPOSITORY_DETECTIVE_API_KEY=your-key
#   ./scripts/operator-smoke-test.sh
#
# Optional:
#   RD_BASE_URL=http://127.0.0.1:8081
#   RD_PORT=8081
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RD_PORT:-8081}"
BASE="${RD_BASE_URL:-http://127.0.0.1:${PORT}}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY}"
HEADER="X-Repository-Detective-API-Key"
LEGACY_HEADER="X-Bugbot-API-Key"

log() { printf '==> %s\n' "$*"; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

truncate() {
  local text=$1
  local max=${2:-400}
  printf '%s' "${text:0:${max}}"
}

curl_json() {
  local path=$1
  shift
  curl -sfS -m 30 "$@" "${BASE}${path}"
}

curl_auth() {
  local path=$1
  if [ -z "$API_KEY" ]; then
    fail "Set REPOSITORY_DETECTIVE_API_KEY for authenticated endpoints"
  fi
  curl_json "$path" -H "${HEADER}: ${API_KEY}"
}

check_health() {
  log "GET /health"
  local body
  body=$(curl_json /health) || fail "/health unreachable at ${BASE}"
  truncate "$body" 400
  echo
  if ! echo "$body" | grep -q '"status"'; then
    fail "/health response missing status field"
  fi
  if echo "$body" | grep -q '"status":"healthy"'; then
    log "/health status=healthy"
  elif echo "$body" | grep -q '"status":"starting"'; then
    warn "/health still starting — wait and retry"
  else
    warn "/health not healthy — check logs"
  fi
}

check_about() {
  log "GET /api/v1/about"
  local body
  body=$(curl_auth /api/v1/about) || fail "/api/v1/about failed"
  truncate "$body" 300
  echo
  if ! echo "$body" | grep -qi 'repository.detective\|Repository Detective'; then
    warn "/api/v1/about may not show product name"
  fi
}

check_status() {
  log "GET /api/v1/status"
  local body
  body=$(curl_auth /api/v1/status) || fail "/api/v1/status failed"
  truncate "$body" 500
  echo
  if echo "$body" | grep -qiE 'gitea_token|api_key|password|secret'; then
    fail "/api/v1/status may leak secrets in response"
  fi
}

check_dashboard() {
  log "GET /api/v1/dashboard/summary"
  local body
  body=$(curl_auth /api/v1/dashboard/summary) || fail "/api/v1/dashboard/summary failed"
  truncate "$body" 400
  echo
}

check_scanner_availability() {
  log "scanner availability (from /health tools_summary)"
  local body
  body=$(curl_json /health)
  if echo "$body" | grep -q 'tools_summary'; then
    local avail
    avail=$(echo "$body" | sed -n 's/.*"available_count":\([0-9]*\).*/\1/p' | head -1)
    local configured
    configured=$(echo "$body" | sed -n 's/.*"configured_count":\([0-9]*\).*/\1/p' | head -1)
    log "scanners available=${avail:-?} configured=${configured:-?}"
    if [ -n "${avail:-}" ] && [ "$avail" -lt 1 ]; then
      warn "no scanners reported available — all-in-one image may be misbuilt"
    fi
  else
    warn "tools_summary not in /health — use /api/v1/status for scanner list"
  fi
}

check_database_healthy() {
  log "database healthy (from /health features)"
  local body
  body=$(curl_json /health)
  if echo "$body" | grep -q '"database_healthy":true'; then
    log "database_healthy=true"
  elif echo "$body" | grep -q '"database_enabled":false'; then
    warn "database disabled"
  else
    warn "database_healthy not true — check DB path and permissions"
  fi
}

check_legacy_header_rejected() {
  if [ -z "$API_KEY" ]; then
    return 0
  fi
  log "legacy header ${LEGACY_HEADER} must be rejected"
  code=$(curl -sS -o /dev/null -w '%{http_code}' -m 15 -H "${LEGACY_HEADER}: ${API_KEY}" "${BASE}/api/v1/status" || echo 000)
  if [ "$code" = "401" ] || [ "$code" = "403" ]; then
    log "legacy API header correctly rejected (HTTP ${code})"
  else
    fail "legacy API header should be rejected, got HTTP ${code}"
  fi
}

main() {
  log "Repository Detective operator smoke test"
  log "base URL: ${BASE}"
  if [ -z "$API_KEY" ]; then
    warn "REPOSITORY_DETECTIVE_API_KEY not set — authenticated checks will fail"
  fi

  check_health
  check_database_healthy
  check_scanner_availability
  check_about
  check_status
  check_dashboard
  check_legacy_header_rejected

  log "operator-smoke-test: completed"
}

main "$@"
