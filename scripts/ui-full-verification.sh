#!/usr/bin/env bash
# Full UI verification: Go handler tests, route smoke, feature matrix, Playwright object audit.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

PORT="${REPOSITORY_DETECTIVE_PORT:-8081}"
BASE="${RD_BASE_URL:-http://127.0.0.1:${PORT}}"
UI_BASE="${BASE}/ui"
KEY="${REPOSITORY_DETECTIVE_API_KEY:-}"
OUT="${RD_VERIFY_OUT:-/tmp/rd-ui-verify}"
REPORT_DIR="${RD_VERIFY_REPORT_DIR:-docs/dogfood-reports}"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
MASTER="${REPORT_DIR}/ui-full-verification-${TS}.md"

mkdir -p "$OUT" "$REPORT_DIR"

log() { printf '==> %s\n' "$*"; }
section() { printf '\n## %s\n\n' "$1" >>"$MASTER"; }

if [[ -z "$KEY" ]]; then
  echo "REPOSITORY_DETECTIVE_API_KEY required in .env" >&2
  exit 2
fi

# Resolve real IDs from SQLite when available
DB="${REPOSITORY_DETECTIVE_DATABASE_PATH:-./data/repository-detective.db}"
if [[ ! -f "$DB" && -f "./data/repository-detective.db" ]]; then
  DB="./data/repository-detective.db"
fi
if [[ -f "$DB" ]]; then
  export RD_VERIFY_REPO_ID="$(python3 -c "
import sqlite3
c=sqlite3.connect('$DB')
r=c.execute(\"SELECT id FROM repositories WHERE full_name LIKE '%Repository-Detective%' ORDER BY id LIMIT 1\").fetchone()
print(r[0] if r else 1)
")"
  export RD_VERIFY_SCAN_ID="$(python3 -c "
import sqlite3
c=sqlite3.connect('$DB')
r=c.execute(\"SELECT id FROM scans WHERE status='completed' ORDER BY finished_at DESC LIMIT 1\").fetchone()
print(r[0] if r else '')
")"
  export RD_VERIFY_FINDING_ID="$(python3 -c "
import sqlite3
c=sqlite3.connect('$DB')
r=c.execute(\"SELECT id FROM findings WHERE status='open' ORDER BY id DESC LIMIT 1\").fetchone()
print(r[0] if r else 1)
")"
fi

log "IDs: repo=${RD_VERIFY_REPO_ID:-1} scan=${RD_VERIFY_SCAN_ID:-} finding=${RD_VERIFY_FINDING_ID:-1}"

{
  echo "# UI full verification"
  echo
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "Base: ${UI_BASE}"
  echo "Image: $(docker inspect --format '{{.Config.Image}}' repository-detective 2>/dev/null || echo unknown)"
} >"$MASTER"

PASS=0
FAIL=0

run_step() {
  local name=$1
  shift
  log "$name"
  if "$@"; then
    echo "- **${name}**: PASS" >>"$MASTER"
    ((PASS++)) || true
  else
    echo "- **${name}**: FAIL" >>"$MASTER"
    ((FAIL++)) || true
  fi
}

section "Go UI handler tests"
if docker run --rm -v "$ROOT:/src" -w /src \
  -e GOFLAGS=-mod=mod -e GOCACHE=/tmp/gocache -e GOPATH=/tmp/gopath -e CGO_ENABLED=0 \
  --entrypoint sh git.commsnet.org/commstech/repository-detective:v0.1.0-beta.2 \
  -c 'go test -buildvcs=false ./ui/... -count=1 2>&1 | tail -20' >>"$MASTER" 2>&1; then
  echo "- **go test ./ui/...**: PASS" >>"$MASTER"
  ((PASS++)) || true
else
  echo "- **go test ./ui/...**: FAIL (see log)" >>"$MASTER"
  ((FAIL++)) || true
fi

section "Route smoke"
export REPOSITORY_DETECTIVE_API_KEY="$KEY"
export RD_BASE_URL="$UI_BASE"
export RD_UI_SMOKE_REPORT="${OUT}/ui-route-smoke.md"
if ./scripts/ui-route-smoke-test.sh >>"$MASTER" 2>&1; then
  echo "- **ui-route-smoke-test.sh**: PASS" >>"$MASTER"
  ((PASS++)) || true
else
  echo "- **ui-route-smoke-test.sh**: FAIL" >>"$MASTER"
  ((FAIL++)) || true
fi

section "Feature matrix"
export REPOSITORY_DETECTIVE_PUBLIC_URL="$BASE"
if ./scripts/feature-matrix-smoke.sh >>"$MASTER" 2>&1; then
  echo "- **feature-matrix-smoke.sh**: PASS" >>"$MASTER"
  ((PASS++)) || true
else
  echo "- **feature-matrix-smoke.sh**: FAIL" >>"$MASTER"
  ((FAIL++)) || true
fi

section "Operator smoke"
export RD_BASE_URL="$BASE"
if ./scripts/operator-smoke-test.sh >>"$MASTER" 2>&1; then
  echo "- **operator-smoke-test.sh**: PASS" >>"$MASTER"
  ((PASS++)) || true
else
  echo "- **operator-smoke-test.sh**: FAIL" >>"$MASTER"
  ((FAIL++)) || true
fi

section "Playwright full object audit"
if docker run --rm --network host \
  -v "$ROOT/scripts:/scripts:ro" \
  -v "$OUT:/out" \
  -e REPOSITORY_DETECTIVE_API_KEY="$KEY" \
  -e RD_BASE_URL="$UI_BASE" \
  -e RD_API_BASE="$BASE" \
  -e RD_VERIFY_REPO_ID="${RD_VERIFY_REPO_ID:-1}" \
  -e RD_VERIFY_SCAN_ID="${RD_VERIFY_SCAN_ID:-}" \
  -e RD_VERIFY_FINDING_ID="${RD_VERIFY_FINDING_ID:-1}" \
  -e RD_VERIFY_OUT=/out \
  mcr.microsoft.com/playwright:v1.49.0-jammy \
  sh -c 'npm install --prefix /tmp/pw playwright@1.49.0 --silent && NODE_PATH=/tmp/pw/node_modules node /scripts/ui-full-verification.js' >>"$MASTER" 2>&1; then
  echo "- **ui-full-verification.js**: PASS" >>"$MASTER"
  ((PASS++)) || true
else
  echo "- **ui-full-verification.js**: FAIL" >>"$MASTER"
  ((FAIL++)) || true
fi

if [[ -f "${OUT}/ui-full-verification.md" ]]; then
  section "Playwright detail"
  cat "${OUT}/ui-full-verification.md" >>"$MASTER"
fi

section "Overall"
{
  echo "- Pass steps: ${PASS}"
  echo "- Fail steps: ${FAIL}"
} >>"$MASTER"

log "Master report: ${MASTER}"
if [[ "$FAIL" -gt 0 ]]; then
  log "FAILED (${FAIL} steps)"
  exit 1
fi
log "ALL PASSED (${PASS} steps)"
