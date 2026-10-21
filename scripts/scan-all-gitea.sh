#!/usr/bin/env bash
# Queue scans for every Gitea repo (legacy wrapper — prefer scripts/scan-all-forges.sh).
# Uses POST /api/v1/analyze/all with forge=gitea.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BASE="${REPOSITORY_DETECTIVE_PUBLIC_URL:-http://127.0.0.1:8081}}"
BASE="${BASE%/}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY}"
PROFILE="${SCAN_PROFILE:-maintainer_deep}"
DRY_RUN="${DRY_RUN:-false}"

if [[ -z "${API_KEY}" ]]; then
  echo "Set REPOSITORY_DETECTIVE_API_KEY in .env" >&2
  exit 1
fi

payload=$(python3 - <<PY
import json, os
orgs = [o.strip() for o in os.environ.get("GITEA_SCAN_ORGS", "").split(",") if o.strip()]
print(json.dumps({
    "forge": "gitea",
    "dry_run": os.environ.get("DRY_RUN", "false").lower() in ("1", "true", "yes"),
    "scan_profile": os.environ.get("SCAN_PROFILE") or "maintainer_deep",
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
