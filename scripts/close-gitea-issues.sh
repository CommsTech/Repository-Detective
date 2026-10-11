#!/usr/bin/env bash
# Post closeout comments and close Gitea issues #38-#47 (except already closed).
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env ]]; then set -a; source .env; set +a; fi
TOKEN="${GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}"
API="https://git.commsnet.org/api/v1/repos/commstech/Repository-Detective"
SHA="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

if [[ -z "${TOKEN}" ]]; then
  echo "No GITEA_TOKEN — cannot close issues via API." >&2
  exit 1
fi

auth=(-H "Authorization: token ${TOKEN}" -H "Content-Type: application/json")

close_issue() {
  local num="$1" body="$2"
  echo "==> #$num"
  curl -sf -X POST "${API}/issues/${num}/comments" "${auth[@]}" \
    -d "$(python3 -c "import json,sys; print(json.dumps({'body': sys.stdin.read()}))" <<<"${body}")"
  curl -sf -X PATCH "${API}/issues/${num}" "${auth[@]}" \
    -d '{"state":"closed"}' >/dev/null
}

close_issue 38 "Closed in ${SHA}: **False positive** — \`analyzers/static.go\` is now excluded from self-scan (rule definitions). SEC-CMD-EXEC was matching regex pattern literals, not runtime command execution. See \`skipStaticAnalysisPath\` and tests in \`analyzers/static_test.go\`."

close_issue 39 "Closed in ${SHA}: **Shipped** — \`ComputeOverallScore\` (\`analyzers/scoring.go\`) drives 0–1 scan scores; documented 10-check matrix in \`docs/SECURITY_CHECK_MATRIX.md\`."

close_issue 40 "Closed in ${SHA}: **Shipped** — \`analyzers/deterministic_test.go\` uses discard logger, \`context.WithTimeout\`, and asserts \`proven[0].ID == trivy-1\` per review."

close_issue 41 "Closed in ${SHA}: **Documented** — integrated tools in \`docs/SBOM.md\`; OpenSCAP and additional scanners remain roadmap (not required for current profiles)."

close_issue 43 "Closed in ${SHA}: **Shipped (advisory)** — optimization static rules \`OPT-*\` + \`docs/OPTIMIZATION_CHECKS.md\`; health performance checks when enabled."

close_issue 45 "Closed in ${SHA}: **Documented** — \`docs/PRE_PUBLISH_CHECKS.md\` covers history scrub, internal refs, PHI/PII manual review, infra refs; gitleaks + static rules automated."

close_issue 46 "Closed in ${SHA}: **Documented** — \`docs/PIPELINE_GOVERNANCE.md\`, static workflow rules \`GOV-*\`, runner guidance in \`RUNNERS.md\`."

close_issue 47 "Closed in ${SHA}: **Review complete** — \`docs/DOC_DETECTIVE_REVIEW.md\`; no Doc Detective dependency added to image."

echo "All issues commented and closed."
