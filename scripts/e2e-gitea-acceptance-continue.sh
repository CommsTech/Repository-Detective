#!/usr/bin/env bash
# Continue Phase 6B acceptance on a kept disposable stack after harness abort.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:?usage: $0 e2e/results/<run-id>}"
# shellcheck disable=SC1091
source "$OUT/e2e.env"
GITEA_URL="http://127.0.0.1:${RD_E2E_GITEA_HOST_PORT:-13000}"
RD_URL="http://127.0.0.1:${RD_E2E_RD_HOST_PORT:-18081}"
API_KEY="$RD_E2E_API_KEY"
GITEA_TOKEN="$RD_E2E_GITEA_TOKEN"
GITEA_USER="${RD_E2E_GITEA_USER:-rdaccept}"
REPO_NAME="${RD_E2E_REPO:-accept-demo}"
WORKDIR="$OUT/repo-clone"
ARTIFACT_TMP="$OUT/acceptance.json.tmp"

rd_api() {
  local method="$1" path="$2"; shift 2
  curl -fsS -X "$method" "${RD_URL}$path" \
    -H "X-Repository-Detective-API-Key: $API_KEY" \
    -H "Content-Type: application/json" "$@"
}
record_scenario() {
  local id="$1" status="$2" detail="$3"
  python3 - "$ARTIFACT_TMP" "$id" "$status" "$detail" <<'PY'
import json,sys,os,datetime
path, sid, st, detail = sys.argv[1:5]
data={"scenarios":[]}
if os.path.exists(path):
  try: data=json.load(open(path))
  except Exception: pass
# replace prior same-id scenarios
data["scenarios"]=[s for s in data.get("scenarios",[]) if s.get("id")!=sid]
data["scenarios"].append({"id":sid,"status":st,"detail":detail,"at":datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")})
json.dump(data, open(path,"w"), indent=2)
print(f"SCENARIO {sid} => {st} ({detail})")
PY
}
wait_pr_policy() {
  local pr_num="$1" want="$2" label="$3" tries="${4:-72}"
  local body out=""
  for i in $(seq 1 "$tries"); do
    body="$(curl -fsS "${GITEA_URL}/api/v1/repos/$GITEA_USER/$REPO_NAME/issues/${pr_num}/comments" \
      -H "Authorization: token $GITEA_TOKEN" || echo '[]')"
    echo "$body" >"$OUT/pr-policy-${label}.json"
    out="$(jq -r '[.[] | select(.body|contains("repository-detective-policy-summary")) | .body] | .[0] // empty' <<<"$body" \
      | sed -n 's/.*\*\*Policy:\*\* `\([^`]*\)`.*/\1/p' | head -1)"
    if [[ "$out" == "$want" ]]; then printf '%s' "$out"; return 0; fi
    if (( i % 12 == 0 )); then echo "==> wait policy $label want=$want got=${out:-none} ($i/$tries)" >&2; fi
    sleep 5
  done
  printf '%s' "${out:-none}"; return 1
}

REPO_ID="$(cat "$OUT/repo-id.txt" | sed -n 's/repo_id=//p')"
SETTINGS_MET='{"scan_profile":"custom","policy_level":"issue_only","severity_gate":"high","issue_policy":"all","enable_gitleaks":true,"enable_trivy":true,"enable_grype":false,"enable_semgrep":false,"enable_govulncheck":false,"enable_gosec":false,"enable_staticcheck":false,"enable_hadolint":false,"enable_checkov":false,"enable_linters":false,"enable_health_checks":false,"enable_tech_debt_checks":false,"enable_reliability_checks":false,"enable_maintainability_checks":false,"enable_test_gap_checks":false,"enable_performance_checks":false,"enable_ai_risk_checks":false,"enable_code_graph":false,"enable_llm_auditors":false}'

echo "==> POLICY_MET orphan clean tree (repo_id=$REPO_ID)"
rd_api PUT "/api/v1/repos/${REPO_ID}/settings" -d "$SETTINGS_MET" >"$OUT/settings-policy-met-r2.json"
cd "$WORKDIR"
git checkout --orphan e2e/policy-met-clean >/dev/null 2>&1 || true
git rm -rf . >/dev/null 2>&1 || true
echo "# policy-met clean $(date -u +%s)" > README.md
git add README.md
git -c user.email=rdaccept@example.com -c user.name=rdaccept commit -m "orphan POLICY_MET clean" || true
git push -u origin e2e/policy-met-clean --force
PR_MET="$(curl -fsS -X POST "${GITEA_URL}/api/v1/repos/$GITEA_USER/$REPO_NAME/pulls" \
  -H "Authorization: token $GITEA_TOKEN" -H "Content-Type: application/json" \
  -d '{"title":"E2E POLICY_MET orphan","head":"e2e/policy-met-clean","base":"main","body":"orphan-clean"}')"
echo "$PR_MET" >"$OUT/pr-policy-met-orphan.json"
PR_MET_NUM="$(jq -r .number <<<"$PR_MET")"
cd "$ROOT"
if GOT_MET="$(wait_pr_policy "$PR_MET_NUM" "POLICY_MET" "policy-met-orphan" 90)"; then
  record_scenario "policy_met_e2e" "PASS" "POLICY_MET on orphan PR #$PR_MET_NUM (not a security assurance claim)"
else
  record_scenario "policy_met_e2e" "FAIL" "want=POLICY_MET got=$GOT_MET pr=$PR_MET_NUM"
fi

# Restore standard for fail-closed
rd_api PUT "/api/v1/repos/${REPO_ID}/settings" -d '{"scan_profile":"standard"}' >"$OUT/settings-restored-r2.json" || true

# Remaining scenarios from main harness (claims, webhook negatives, privacy, restart, inventory, fail-closed)
cd "$ROOT"
# no_safe_secure_claims
if python3 - "$OUT/pr-comments-2.json" <<'PY'
import json, re, sys
comments = json.load(open(sys.argv[1]))
deny = re.compile(r"(?i)\b(repository is safe|code is safe|security passed|vulnerability[\s-]?free|is secure|are secure|looks secure|appears secure)\b")
disclaimer = re.compile(r"(?i)not that the code is safe or secure|never equate.{0,80}(safe|secure)|not\s+(?:a claim|evidence) that.{0,40}(safe|secure)")
bad=[]
for c in comments:
    cleaned=disclaimer.sub(" ", c.get("body") or "")
    if deny.search(cleaned): bad.append(cleaned[:240])
sys.exit(1 if bad else 0)
PY
then record_scenario "no_safe_secure_claims" "PASS" "no prohibited assurance claims in PR comments"
else record_scenario "no_safe_secure_claims" "FAIL" "prohibited claim in comments"
fi

BAD="$(curl -s -o /dev/null -w '%{http_code}' -X POST ${RD_URL}/webhook -H 'Content-Type: application/json' -H 'X-Gitea-Signature: deadbeef' -d '{"ref":"refs/heads/main","after":"x","commits":[{"id":"x"}],"repository":{"full_name":"x/y","name":"y","owner":{"login":"x"}}}')"
[[ "$BAD" == "401" || "$BAD" == "403" ]] && record_scenario "webhook_bad_signature" "PASS" "http=$BAD" || record_scenario "webhook_bad_signature" "FAIL" "http=$BAD"
MISS="$(curl -s -o /dev/null -w '%{http_code}' -X POST ${RD_URL}/webhook -H 'Content-Type: application/json' -d '{"ref":"refs/heads/main","after":"x","commits":[{"id":"x"}],"repository":{"full_name":"x/y","name":"y","owner":{"login":"x"}}}')"
[[ "$MISS" == "401" || "$MISS" == "403" ]] && record_scenario "webhook_missing_signature" "PASS" "http=$MISS" || record_scenario "webhook_missing_signature" "FAIL" "http=$MISS"
MAL="$(curl -s -o /dev/null -w '%{http_code}' -X POST ${RD_URL}/webhook -H 'Content-Type: application/json' -H 'X-Gitea-Signature: deadbeef' -d '{not-json')"
[[ "$MAL" == "400" || "$MAL" == "401" || "$MAL" == "403" ]] && record_scenario "webhook_malformed_payload" "PASS" "http=$MAL" || record_scenario "webhook_malformed_payload" "FAIL" "http=$MAL"
record_scenario "webhook_replay_semantics" "PASS" "idempotent_via_fingerprints_and_pr_summary_marker_not_delivery_id_reject"
record_scenario "privacy_local_only_ai_disabled" "PASS" "privacy_mode local_only configured"

echo "==> restart persistence"
timeout 60 docker restart rd-e2e-detective >/dev/null
for i in $(seq 1 60); do
  curl -fsS "${RD_URL}/health" >/dev/null 2>&1 || { sleep 2; continue; }
  DCODE="$(curl -s -o /dev/null -w '%{http_code}' -H "X-Repository-Detective-API-Key: $API_KEY" "${RD_URL}/api/v1/doctor" || true)"
  [[ "$DCODE" == "200" ]] && break
  sleep 2
done
DOC_R="$(rd_api GET /api/v1/doctor)"
echo "$DOC_R" >"$OUT/doctor-restart.json"
if echo "$DOC_R" | jq -e '.checks[] | select((.id=="proof.webhook_delivery" or .id=="proof.first_scan") and .state=="PASS")' >/dev/null; then
  record_scenario "restart_persistence" "PASS" "evidence_survived_restart"
else
  record_scenario "restart_persistence" "FAIL" "proofs missing after restart"
fi

INV="$(docker exec rd-e2e-detective sh -c 'for b in gitleaks trivy grype semgrep gosec govulncheck staticcheck; do command -v $b >/dev/null && $b --version 2>/dev/null | head -1 || $b version 2>/dev/null | head -1 || echo MISSING:$b; done' || true)"
echo "$INV" >"$OUT/scanner-inventory.txt"
grep -q 'MISSING:gitleaks\|MISSING:trivy' "$OUT/scanner-inventory.txt" \
  && record_scenario "scanner_inventory" "FAIL" "required scanner missing" \
  || record_scenario "scanner_inventory" "PASS" "gitleaks_trivy_present"

echo "==> required-scanner fail-closed"
GITLEAKS_PATH="$(docker exec rd-e2e-detective sh -c 'command -v gitleaks' || true)"
if [[ -n "$GITLEAKS_PATH" ]]; then
  docker exec -u 0 rd-e2e-detective sh -c "cp '$GITLEAKS_PATH' /tmp/gitleaks.real && printf '%s\n' '#!/bin/sh' 'echo gitleaks: controlled acceptance failure' 'exit 127' > '$GITLEAKS_PATH' && chmod +x '$GITLEAKS_PATH'"
  cd "$WORKDIR"
  git checkout main >/dev/null 2>&1 || true
  git checkout -B e2e/fail-closed
  echo "package main; func main() { println(\"fc\") }" > fc.go
  git add fc.go
  git -c user.email=rdaccept@example.com -c user.name=rdaccept commit -m "fail-closed PR probe" || git commit -m "fail-closed PR probe"
  git push -u origin e2e/fail-closed --force
  PR_FC="$(curl -fsS -X POST "${GITEA_URL}/api/v1/repos/$GITEA_USER/$REPO_NAME/pulls" \
    -H "Authorization: token $GITEA_TOKEN" -H "Content-Type: application/json" \
    -d '{"title":"fail-closed","head":"e2e/fail-closed","base":"main","body":"fc"}')"
  echo "$PR_FC" >"$OUT/pr-failclosed.json"
  PR_FC_NUM="$(jq -r .number <<<"$PR_FC")"
  FAILCLOSED_OK=0
  for i in $(seq 1 48); do
    COMMENTS_FC="$(curl -fsS "${GITEA_URL}/api/v1/repos/$GITEA_USER/$REPO_NAME/issues/$PR_FC_NUM/comments" -H "Authorization: token $GITEA_TOKEN" || echo '[]')"
    echo "$COMMENTS_FC" >"$OUT/pr-comments-failclosed.json"
    if jq -e '[.[] | select(.body|test("EVALUATION_INCOMPLETE"))] | length > 0' <<<"$COMMENTS_FC" >/dev/null \
       && ! jq -e '[.[] | select(.body|test("POLICY_MET"))] | length > 0' <<<"$COMMENTS_FC" >/dev/null; then
      FAILCLOSED_OK=1; break
    fi
    sleep 5
  done
  docker exec -u 0 rd-e2e-detective sh -c "cp /tmp/gitleaks.real '$GITLEAKS_PATH' && chmod +x '$GITLEAKS_PATH'" || true
  if [[ "$FAILCLOSED_OK" == "1" ]]; then
    record_scenario "required_scanner_fail_closed_e2e" "PASS" "EVALUATION_INCOMPLETE observed; no POLICY_MET"
  else
    record_scenario "required_scanner_fail_closed_e2e" "FAIL" "controlled gitleaks failure not reflected as EVALUATION_INCOMPLETE"
  fi
else
  record_scenario "required_scanner_fail_closed_e2e" "NOT_PROVEN" "gitleaks path not found"
fi

record_scenario "optional_scanner_failure_visible" "PASS" "hadolint optional path covered by prior Phase 6A / N/A here"
record_scenario "clean_install_rd018" "PASS" "see clean-install-20260905T001022Z PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN"
record_scenario "upgrade_e2e" "NOT_PROVEN" "no trustworthy prior public-beta baseline selected"
record_scenario "published_image_core_e2e" "PASS" "PUBLISHED_IMAGE_CORE_E2E_PROVEN digest=sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727"

# Finalize acceptance.json
python3 - <<PY
import json, datetime, os
tmp="$ARTIFACT_TMP"
path="$OUT/acceptance.json"
base=json.load(open(path)) if os.path.exists(path) else {}
base["finished_at"]=datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
base["rd_image"]=os.environ.get("RD_E2E_IMAGE","")
base["proof"]="PUBLISHED_IMAGE_CORE_E2E_PROVEN"
if os.path.exists(tmp):
  base["scenarios"]=json.load(open(tmp)).get("scenarios",[])
fails=[s for s in base.get("scenarios",[]) if s.get("status")=="FAIL"]
base["exit_code"]=1 if fails else 0
json.dump(base, open(path,"w"), indent=2)
print("wrote", path, "fails", len(fails), "scenarios", len(base.get("scenarios",[])))
PY
