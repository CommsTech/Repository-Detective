#!/usr/bin/env bash
# Create Gitea labels, milestones, and optional backlog issues for commstech/Repository-Detective.
# Does nothing without GITEA_TOKEN (or REPOSITORY_DETECTIVE_GITEA_TOKEN) in environment or .env.
set -euo pipefail
cd "$(dirname "$0")/.."

GITEA_URL="${GITEA_URL:-${REPOSITORY_DETECTIVE_GITEA_URL:-${REPOSITORY_DETECTIVE_GITEA_URL:-https://git.commsnet.org}}}"
GITEA_URL="${GITEA_URL%/}"
OWNER="${GITEA_OWNER:-commstech}"
REPO="${GITEA_REPO:-Repository-Detective}"
API="${GITEA_URL}/api/v1/repos/${OWNER}/${REPO}"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

TOKEN="${GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}}"
if [[ -z "${TOKEN}" ]]; then
  echo "No Gitea token — export GITEA_TOKEN or add to .env. Prepared issues: docs/issues/"
  exit 0
fi

auth=(-H "Authorization: token ${TOKEN}" -H "Content-Type: application/json")

create_label() {
  local name="$1" color="$2"
  curl -sf -X POST "${API}/labels" "${auth[@]}" \
    -d "{\"name\":\"${name}\",\"color\":\"${color}\"}" >/dev/null 2>&1 \
    || echo "  label exists or failed: ${name}"
}

create_milestone() {
  local title="$1"
  curl -sf -X POST "${API}/milestones" "${auth[@]}" \
    -d "{\"title\":\"${title}\",\"state\":\"open\"}" >/dev/null 2>&1 \
    || echo "  milestone exists or failed: ${title}"
}

LABELS_ONLY=false
CREATE_ISSUES=false
for arg in "$@"; do
  case "$arg" in
    --labels-only) LABELS_ONLY=true ;;
    --issues) CREATE_ISSUES=true ;;
  esac
done

if [[ "$LABELS_ONLY" == true ]] || [[ "$CREATE_ISSUES" == true ]] || [[ $# -eq 0 ]]; then
  echo "==> Labels"
  for pair in \
    "type/bug,e11d21" "type/feature,1d76db" "type/docs,5319e7" \
    "type/compliance,6f42c1" "type/privacy,6f42c1" "type/accessibility,0e8a16" \
    "type/security,d93f0b" "type/scanner,fbc04d" "type/false-positive,c5def5" \
    "type/ui,1d76db" "type/api,1d76db" "type/reporting,1d76db" \
    "severity/critical,b60205" "severity/high,d93f0b" "severity/medium,fbca04" "severity/low,0e8a16" \
    "status/needs-triage,ededed" "status/accepted,0e8a16" "status/in-progress,1d76db" \
    "status/blocked,e11d21" "status/ready-for-test,1d76db" "status/done,0e8a16" \
    "priority/p0,b60205" "priority/p1,d93f0b" "priority/p2,fbca04" "priority/p3,c5def5"
  do
    create_label "${pair%%,*}" "${pair##*,}"
  done

  echo "==> Milestones"
  for m in \
    "Sprint 1 - Issue and Feature Backlog" \
    "Sprint 2 - Accessibility" \
    "Sprint 3 - Privacy and Compliance Readiness" \
    "Sprint 4 - Repository Detective Self-Scan" \
    "Sprint 5 - Bug and Feature Closeout" \
    "Sprint 6 - Release Readiness"
  do
    create_milestone "$m"
  done
fi

if [[ "$CREATE_ISSUES" == true ]]; then
  echo "==> Issues from docs/issues/*.md (manual titles — review before bulk create)"
  echo "    Parse docs/issues/README.md and open issues via Gitea UI or extend this script."
  echo "    No automatic issue creation in this script (avoids duplicates)."
fi

echo "Done."
