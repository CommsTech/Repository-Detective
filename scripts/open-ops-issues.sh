#!/usr/bin/env bash
# Open operator/deployment tracking issues on commstech/Repository-Detective (requires GITEA_TOKEN in .env).
set -euo pipefail
cd "$(dirname "$0")/.."
[[ -f .env ]] && set -a && source .env && set +a
TOKEN="${GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}"
OWNER="${GITEA_ISSUE_OWNER:-commstech}"
REPO="${GITEA_ISSUE_REPO:-repository-detective}"
API="https://git.commsnet.org/api/v1/repos/${OWNER}/${REPO}"

if [[ -z "${TOKEN}" ]]; then
  echo "Set GITEA_TOKEN in .env" >&2
  exit 1
fi

create_issue() {
  local title="$1"
  local body="$2"
  curl -sf -X POST "${API}/issues" \
    -H "Authorization: token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$(python3 -c 'import json,sys; print(json.dumps({"title":sys.argv[1],"body":sys.argv[2]}))' "$title" "$body")" \
    | python3 -c 'import sys,json; i=json.load(sys.stdin); print(i.get("html_url", i))'
}

create_issue "Ops: Docker Trivy install when GitHub CDN blocked" "$(cat docs/issues/P2-docker-trivy-github-blocked.md)"
