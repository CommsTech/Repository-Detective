#!/usr/bin/env bash
# Queue Repository Detective scans for every Gitea repo the configured token can see.
# Uses POST /api/v1/analyze/all (user repos + optional orgs via GITEA_SCAN_ORGS).
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BASE="${REPOSITORY_DETECTIVE_PUBLIC_URL:-${BUGBOT_PUBLIC_URL:-http://127.0.0.1:8081}}"
BASE="${BASE%/}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-${BUGBOT_API_KEY:-}}"
PROFILE="${SCAN_PROFILE:-${BUGBOT_SCAN_PROFILE:-standard_deterministic}}"
DRY_RUN="${DRY_RUN:-false}"

if [[ -z "${API_KEY}" ]]; then
  echo "Set REPOSITORY_DETECTIVE_API_KEY or BUGBOT_API_KEY in .env" >&2
  exit 1
fi

payload=$(python3 - <<PY
import json, os
orgs = [o.strip() for o in os.environ.get("GITEA_SCAN_ORGS", "").split(",") if o.strip()]
print(json.dumps({
    "dry_run": os.environ.get("DRY_RUN", "false").lower() in ("1", "true", "yes"),
    "scan_profile": os.environ.get("SCAN_PROFILE") or os.environ.get("BUGBOT_SCAN_PROFILE") or "standard_deterministic",
    "orgs": orgs,
}))
PY
)

echo "==> Health"
curl -sf "${BASE}/health" | head -c 200
echo ""

echo "==> Bulk analyze (dry_run=${DRY_RUN}, profile=${PROFILE})"
resp=$(mktemp)
code=$(curl -sS -o "${resp}" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d "${payload}" \
  "${BASE}/api/v1/analyze/all")

echo "HTTP ${code}"
cat "${resp}"
echo ""

if [[ "${code}" != "200" ]]; then
  exit 1
fi

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "Dry run only — set DRY_RUN=false to queue scans."
  exit 0
fi

echo "Scans queued. Monitor: docker logs repository-detective -f --tail 50"
echo "Dashboard: ${BASE}/ui/"
