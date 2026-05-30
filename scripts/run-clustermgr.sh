#!/usr/bin/env bash
# Run Bugbot natively on clustermgr (no Docker).
# Logs to stdout — redirect if needed: ./scripts/run-clustermgr.sh 2>&1 | tee bugbot.log
set -euo pipefail

cd "$(dirname "$0")/.."

export BUGBOT_PORT="${BUGBOT_PORT:-8081}"
export BUGBOT_LISTEN_HOST="${BUGBOT_LISTEN_HOST:-0.0.0.0}"
export BUGBOT_SKIP_STARTUP_CHECKS="${BUGBOT_SKIP_STARTUP_CHECKS:-true}"
export BUGBOT_STARTUP_CHECK_TIMEOUT="${BUGBOT_STARTUP_CHECK_TIMEOUT:-10}"
export BUGBOT_LOG_LEVEL="${BUGBOT_LOG_LEVEL:-info}"

: "${BUGBOT_GITEA_URL:?Set BUGBOT_GITEA_URL}"
: "${BUGBOT_GITEA_TOKEN:?Set BUGBOT_GITEA_TOKEN}"
: "${BUGBOT_API_KEY:?Set BUGBOT_API_KEY}"

echo "Starting Bugbot on ${BUGBOT_LISTEN_HOST}:${BUGBOT_PORT}"
echo "  Health: curl http://127.0.0.1:${BUGBOT_PORT}/health"
echo ""

if [[ -x ./gitea-bugbot ]]; then
  exec ./gitea-bugbot
fi

exec go run .
