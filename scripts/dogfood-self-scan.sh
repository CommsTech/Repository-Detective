#!/usr/bin/env bash
# Trigger a safe self-scan of commstech/Repository-Detective (dogfood). Does not create issues unless configured.
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

echo "==> Health"
curl -sf "${BASE}/health" | head -c 400
echo ""

if [[ -z "${API_KEY}" ]]; then
  echo "No API key — set REPOSITORY_DETECTIVE_API_KEY in .env to run analyze."
  echo "Manual steps: docs/DOGFOODING.md"
  exit 0
fi

echo "==> Status (scanner tools)"
curl -sf -H "Authorization: Bearer ${API_KEY}" "${BASE}/api/v1/status" 2>/dev/null | head -c 600 || \
  curl -sf "${BASE}/api/v1/status?api_key=${API_KEY}" | head -c 600
echo ""

echo "==> Analyze (connected repo must exist in DB)"
payload=$(cat <<EOF
{"owner":"${OWNER}","repository":"${REPO}","ref":"main","report_only_dry_run":true}
EOF
)
code=$(curl -s -o /tmp/rd-dogfood-analyze.json -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -H "X-Repository-Detective-API-Key: ${API_KEY}" \
  -d "${payload}" \
  "${BASE}/api/v1/analyze" 2>/dev/null || echo "000")

if [[ "${code}" == "000" || "${code}" == "401" ]]; then
  code=$(curl -s -o /tmp/rd-dogfood-analyze.json -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -d "${payload}" \
    "${BASE}/api/v1/analyze?api_key=${API_KEY}")
fi

echo "HTTP ${code}"
head -c 500 /tmp/rd-dogfood-analyze.json 2>/dev/null || true
echo ""
echo "Review findings: ${BASE}/ui/findings"
echo "Fill report: docs/DOGFOOD_REPORT_TEMPLATE.md"
