#!/usr/bin/env bash
# Scan repository-detective container logs for operational health signals.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CONTAINER="${RD_CONTAINER_NAME:-repository-detective}"
LINES="${RD_LOG_LINES:-500}"
REPORT="${RD_LOG_HEALTH_REPORT:-docs/dogfood-reports/container-log-health-report.md}"

mkdir -p "$(dirname "$REPORT")"

log() { printf '==> %s\n' "$*"; }

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  log "Container ${CONTAINER} not running — skipping log scrape"
  {
    echo "# Container log health report"
    echo
    echo "Container \`${CONTAINER}\` not running at $(date -u +"%Y-%m-%dT%H:%M:%SZ")."
  } >"$REPORT"
  exit 0
fi

RAW=$(docker logs --tail "$LINES" "$CONTAINER" 2>&1 || true)

count_pattern() {
  echo "$RAW" | grep -ciE "$1" || true
}

PANICS=$(count_pattern 'panic:|fatal error')
FATALS=$(count_pattern 'level=fatal|FATAL')
DBLOCK=$(count_pattern 'database is locked|deadlock')
AUTHF=$(count_pattern 'auth failed|unauthorized')
AIFAIL=$(count_pattern 'ai recommendation|openclaw|non-JSON|strict json')
SCANF=$(count_pattern 'scanner.*fail|trivy.*error|grype.*error')
SECRETS=$(echo "$RAW" | grep -ciE 'api_key=|password=|AKIA[0-9A-Z]{16}' || true)

{
  echo "# Container log health report"
  echo
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "Container: \`${CONTAINER}\` (last ${LINES} lines)"
  echo
  echo "## Pattern counts"
  echo "- Panics: ${PANICS}"
  echo "- Fatals: ${FATALS}"
  echo "- DB lock/deadlock: ${DBLOCK}"
  echo "- Auth failures: ${AUTHF}"
  echo "- AI recommendation failures: ${AIFAIL}"
  echo "- Scanner failures: ${SCANF}"
  echo "- Possible secrets in logs: ${SECRETS}"
  echo
  echo "## Known expected"
  echo "- OpenClaw/provider non-JSON responses when strict JSON not configured on provider side"
  echo
  echo "## Recent errors (sample)"
  echo '```'
  echo "$RAW" | grep -iE 'error|panic|fatal' | tail -n 20 || echo "(none)"
  echo '```'
} >"$REPORT"

log "Report written to ${REPORT}"
if [ "$PANICS" -gt 0 ] || [ "$SECRETS" -gt 0 ]; then
  exit 1
fi
