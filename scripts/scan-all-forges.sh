#!/usr/bin/env bash
# Queue scans for every repo visible to configured Gitea and/or GitHub tokens.
# Uses POST /api/v1/analyze/all (forge=all by default).
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
FORGE="${FORGE:-all}"

if [[ -z "${API_KEY}" ]]; then
  echo "Set REPOSITORY_DETECTIVE_API_KEY in .env" >&2
  exit 1
fi

payload=$(python3 - <<PY
import json, os

def orgs_from_env(*keys):
    for key in keys:
        raw = os.environ.get(key, "")
        if raw.strip():
            return [o.strip() for o in raw.split(",") if o.strip()]
    return []

# Prefer explicit ORGS; otherwise merge Gitea/GitHub org env vars for the bulk endpoint.
orgs = orgs_from_env("ORGS", "SCAN_ORGS")
if not orgs:
    seen = set()
    for key in ("GITEA_SCAN_ORGS", "GITHUB_SCAN_ORGS"):
        for o in orgs_from_env(key):
            if o not in seen:
                seen.add(o)
                orgs.append(o)

print(json.dumps({
    "dry_run": os.environ.get("DRY_RUN", "false").lower() in ("1", "true", "yes"),
    "scan_profile": os.environ.get("SCAN_PROFILE") or "maintainer_deep",
    "forge": os.environ.get("FORGE", "all"),
    "orgs": orgs,
}))
PY
)

echo "==> Health"
curl -sf "${BASE}/health" | head -c 200
echo ""

echo "==> Bulk analyze (forge=${FORGE}, dry_run=${DRY_RUN}, profile=${PROFILE})"
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
