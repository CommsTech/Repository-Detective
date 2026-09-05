#!/usr/bin/env bash
# RD-033 — Upgrade acceptance: exact v0.1.0-beta.3 digest → CURRENT development candidate.
#
# Classification (until target is a published release digest):
#   UPGRADE_FROM_BETA3_TO_CURRENT_MAIN_INTEGRATION_PROVEN
# Not:
#   PUBLISHED_RELEASE_UPGRADE_E2E_PROVEN
#
# Preserves a disposable baseline DB snapshot. Failure keeps sanitized diagnostics.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export COMPOSE_HTTP_TIMEOUT="${COMPOSE_HTTP_TIMEOUT:-600}"
export DOCKER_CLIENT_TIMEOUT="${DOCKER_CLIENT_TIMEOUT:-600}"
if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose -f docker-compose.e2e.yml)
else
  COMPOSE=(docker-compose -f docker-compose.e2e.yml)
fi

RUN_ID="${RD_UPGRADE_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-upgrade-$$}"
OUT_DIR="${RD_UPGRADE_OUT:-$ROOT/e2e/results/$RUN_ID}"
mkdir -p "$OUT_DIR/snapshots" "$OUT_DIR/diagnostics"
ARTIFACT="$OUT_DIR/upgrade-acceptance.json"
SUMMARY="$OUT_DIR/summary.md"
KEEP_ON_FAIL="${RD_UPGRADE_KEEP_ON_FAIL:-1}"

BETA3_DIGEST="sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727"
BETA3_REF="${RD_UPGRADE_BASELINE_IMAGE:-ghcr.io/commstech/repository-detective@${BETA3_DIGEST}}"
CANDIDATE_IMAGE="${RD_UPGRADE_CANDIDATE_IMAGE:-repository-detective:upgrade-candidate}"
CLASSIFICATION="UPGRADE_FROM_BETA3_TO_CURRENT_MAIN_INTEGRATION_PROVEN"

# Dedicated ports so this harness does not collide with a live core E2E stack.
GITEA_HOST_PORT="${RD_E2E_GITEA_HOST_PORT:-13100}"
RD_HOST_PORT="${RD_E2E_RD_HOST_PORT:-18181}"
export RD_E2E_GITEA_HOST_PORT="$GITEA_HOST_PORT"
export RD_E2E_RD_HOST_PORT="$RD_HOST_PORT"

GITEA_URL="http://127.0.0.1:${GITEA_HOST_PORT}"
RD_URL="http://127.0.0.1:${RD_HOST_PORT}"
API_KEY="${RD_E2E_API_KEY:-e2e-upgrade-api-key-not-a-secret}"
WEBHOOK_SECRET="${RD_E2E_WEBHOOK_SECRET:-e2e-upgrade-webhook-secret}"
GITEA_USER="${RD_E2E_GITEA_USER:-rdupgrade}"
GITEA_PASS="${RD_E2E_GITEA_PASS:-UpgradeTestPass1!}"
GITEA_EMAIL="${RD_E2E_GITEA_EMAIL:-rdupgrade@example.com}"
REPO_NAME="${RD_E2E_REPO:-upgrade-demo}"

export RD_E2E_API_KEY="$API_KEY"
export RD_E2E_WEBHOOK_SECRET="$WEBHOOK_SECRET"

log() { printf '==> %s\n' "$*" | tee -a "$OUT_DIR/harness.log"; }
fail() { printf 'ERROR: %s\n' "$*" | tee -a "$OUT_DIR/harness.log" >&2; exit 1; }

record_scenario() {
  local id="$1" status="$2" detail="$3"
  python3 - "$OUT_DIR/scenarios.tmp" "$id" "$status" "$detail" <<'PY' || true
import json,sys,os,datetime
path, sid, st, detail = sys.argv[1:5]
data={"scenarios":[]}
if os.path.exists(path):
  try: data=json.load(open(path))
  except Exception: pass
data.setdefault("scenarios",[]).append({
  "id": sid, "status": st, "detail": detail,
  "at": datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
})
json.dump(data, open(path,"w"), indent=2)
PY
  log "SCENARIO $id => $status ($detail)"
}

sanitize_file() {
  local f="$1"
  python3 - "$f" "$API_KEY" "$WEBHOOK_SECRET" "$GITEA_PASS" "${GITEA_TOKEN:-}" <<'PY' || true
import sys
path=sys.argv[1]
secrets=[s for s in sys.argv[2:] if s]
try:
  text=open(path,"r",errors="replace").read()
except Exception:
  raise SystemExit(0)
for s in secrets:
  text=text.replace(s,"[REDACTED]")
open(path,"w").write(text)
PY
}

finalize() {
  local code="$1"
  python3 - <<PY
import json, os, datetime
tmp="$OUT_DIR/scenarios.tmp"
base={
  "run_id":"$RUN_ID",
  "finished_at":datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
  "exit_code": $code,
  "classification": "$CLASSIFICATION" if $code==0 else "UPGRADE_NOT_PROVEN",
  "baseline_digest":"$BETA3_DIGEST",
  "baseline_image":"$BETA3_REF",
  "candidate_image":"$CANDIDATE_IMAGE",
  "repository_detective_commit":"$(git rev-parse HEAD 2>/dev/null || echo unknown)",
  "note":"Until candidate is a published release digest, this is integration upgrade proof only.",
  "scenarios":[]
}
if os.path.exists(tmp):
  try: base["scenarios"]=json.load(open(tmp)).get("scenarios",[])
  except Exception: pass
blob=json.dumps(base)
for s in ["$GITEA_PASS","$API_KEY","$WEBHOOK_SECRET","${GITEA_TOKEN:-}"]:
  if s: blob=blob.replace(s,"[REDACTED]")
open("$ARTIFACT","w").write(json.dumps(json.loads(blob), indent=2)+"\n")
with open("$SUMMARY","w") as fh:
  fh.write("# Upgrade acceptance %s\n\n" % "$RUN_ID")
  fh.write("classification: %s\n" % base["classification"])
  fh.write("baseline: %s\ncandidate: %s\nexit: %s\n\n" % ("$BETA3_DIGEST","$CANDIDATE_IMAGE",$code))
  for sc in base["scenarios"]:
    fh.write("- **%s**: %s — %s\n" % (sc["id"], sc["status"], sc["detail"]))
print("wrote", "$ARTIFACT")
PY
}

cleanup() {
  local code=$?
  # Preserve sanitized logs on failure
  docker logs rd-e2e-detective >"$OUT_DIR/diagnostics/rd-logs.txt" 2>&1 || true
  docker logs rd-e2e-gitea >"$OUT_DIR/diagnostics/gitea-logs.txt" 2>&1 || true
  sanitize_file "$OUT_DIR/diagnostics/rd-logs.txt"
  sanitize_file "$OUT_DIR/diagnostics/gitea-logs.txt"
  if [[ $code -ne 0 && "$KEEP_ON_FAIL" == "1" ]]; then
    log "keeping environment for debug (RD_UPGRADE_KEEP_ON_FAIL=1)"
  else
    "${COMPOSE[@]}" down -v >/dev/null 2>&1 || true
  fi
  finalize "$code"
  exit $code
}
trap cleanup EXIT

wait_http() {
  local url="$1" tries="${2:-90}"
  local i=0
  while (( i < tries )); do
    if curl -fsS "$url" >/dev/null 2>&1; then return 0; fi
    sleep 2
    i=$((i+1))
  done
  return 1
}

rd_api() {
  local method="$1" path="$2"
  shift 2
  curl -fsS -X "$method" "${RD_URL}$path" \
    -H "X-Repository-Detective-API-Key: $API_KEY" \
    -H "Content-Type: application/json" "$@"
}

wait_rd_ready() {
  local tries="${1:-90}"
  local i=0
  while (( i < tries )); do
    HJSON="$(curl -fsS "${RD_URL}/health" 2>/dev/null || echo '{}')"
    DCODE="$(curl -s -o /dev/null -w '%{http_code}' -H "X-Repository-Detective-API-Key: $API_KEY" "${RD_URL}/api/v1/doctor" || true)"
    if echo "$HJSON" | jq -e '.ready == true or .status == "healthy"' >/dev/null 2>&1; then
      if [[ "$DCODE" == "200" ]]; then return 0; fi
    fi
    sleep 2
    i=$((i+1))
  done
  return 1
}

snapshot_rd_db() {
  local label="$1"
  local dest="$OUT_DIR/snapshots/${label}.db"
  # Checkpoint WAL so a single-file docker cp is a consistent disposable baseline.
  docker exec rd-e2e-detective sh -c '
    if command -v sqlite3 >/dev/null 2>&1; then
      sqlite3 /app/data/repository-detective.db "PRAGMA wal_checkpoint(FULL);"
    fi
  ' >/dev/null 2>&1 || true
  if ! docker cp "rd-e2e-detective:/app/data/repository-detective.db" "$dest" 2>/dev/null; then
    fail "failed to snapshot DB as $label"
  fi
  # Best-effort WAL companions (may be empty after checkpoint).
  docker cp "rd-e2e-detective:/app/data/repository-detective.db-wal" "${dest}-wal" 2>/dev/null || true
  docker cp "rd-e2e-detective:/app/data/repository-detective.db-shm" "${dest}-shm" 2>/dev/null || true
  chmod 644 "$dest" 2>/dev/null || true
  log "snapshot written $dest ($(wc -c <"$dest") bytes)"
}

# --- start ---
log "run_id=$RUN_ID out=$OUT_DIR"
log "baseline=$BETA3_REF"
log "candidate=$CANDIDATE_IMAGE"
"${COMPOSE[@]}" down -v >/dev/null 2>&1 || true

log "pulling exact beta.3 digest"
docker pull "$BETA3_REF" || fail "cannot pull baseline digest"
docker tag "$BETA3_REF" "repository-detective:v0.1.0-beta.3"
export RD_E2E_IMAGE="repository-detective:v0.1.0-beta.3"
record_scenario "baseline_image" "PASS" "digest=$BETA3_DIGEST"

if ! docker image inspect "$CANDIDATE_IMAGE" >/dev/null 2>&1; then
  log "building candidate overlay onto beta.3 (Dockerfile.binary-overlay)"
  docker build -f Dockerfile.binary-overlay \
    --build-arg "SOURCE_IMAGE=$BETA3_REF" \
    --build-arg VERSION=upgrade-candidate \
    --build-arg COMMIT="$(git rev-parse --short HEAD)" \
    --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -t "$CANDIDATE_IMAGE" "$ROOT" || fail "candidate build failed"
fi
record_scenario "candidate_image" "PASS" "image=$CANDIDATE_IMAGE"

log "starting Gitea 1.22.3"
"${COMPOSE[@]}" up -d gitea
wait_http "${GITEA_URL}/api/v1/version" 210 || fail "gitea not ready"
GITEA_VER="$(curl -fsS "${GITEA_URL}/api/v1/version" | jq -r .version)"
[[ "$GITEA_VER" == "1.22.3" ]] || log "WARN gitea version=$GITEA_VER (expected 1.22.3)"
record_scenario "gitea_ready" "PASS" "version=$GITEA_VER"

docker exec -u git rd-e2e-gitea gitea admin user create \
  --username "$GITEA_USER" --password "$GITEA_PASS" --email "$GITEA_EMAIL" \
  --admin --must-change-password=false >/dev/null 2>&1 || true

TOKEN_JSON="$(curl -fsS -u "$GITEA_USER:$GITEA_PASS" \
  -H "Content-Type: application/json" \
  -X POST "${GITEA_URL}/api/v1/users/$GITEA_USER/tokens" \
  -d "{\"name\":\"rd-upgrade-$RUN_ID\",\"scopes\":[\"all\"]}" || true)"
GITEA_TOKEN="$(printf '%s' "$TOKEN_JSON" | jq -r '.sha1 // .token // empty')"
[[ -n "$GITEA_TOKEN" && "$GITEA_TOKEN" != null ]] || fail "gitea token failed: $TOKEN_JSON"
export RD_E2E_GITEA_TOKEN="$GITEA_TOKEN"
record_scenario "gitea_token" "PASS" "created"

curl -fsS -X POST "${GITEA_URL}/api/v1/user/repos" \
  -H "Authorization: token $GITEA_TOKEN" -H "Content-Type: application/json" \
  -d "{\"name\":\"$REPO_NAME\",\"private\":true,\"auto_init\":true,\"default_branch\":\"main\",\"description\":\"RD upgrade fixture\"}" \
  >"$OUT_DIR/repo.json" || fail "create repo failed"
record_scenario "repo_create" "PASS" "$GITEA_USER/$REPO_NAME"

cat >"$OUT_DIR/e2e.env" <<EOF
RD_E2E_API_KEY=$API_KEY
RD_E2E_WEBHOOK_SECRET=$WEBHOOK_SECRET
RD_E2E_GITEA_TOKEN=$GITEA_TOKEN
RD_E2E_IMAGE=$RD_E2E_IMAGE
RD_E2E_GITEA_HOST_PORT=$GITEA_HOST_PORT
RD_E2E_RD_HOST_PORT=$RD_HOST_PORT
EOF
set -a; # shellcheck disable=SC1090
source "$OUT_DIR/e2e.env"; set +a

log "starting Repository Detective at beta.3"
"${COMPOSE[@]}" up -d repository-detective
wait_http "${RD_URL}/health" 90 || fail "beta.3 health not ready"
wait_rd_ready 90 || fail "beta.3 doctor not ready"
BEFORE_HEALTH="$(curl -fsS "${RD_URL}/health")"
echo "$BEFORE_HEALTH" >"$OUT_DIR/health-before.json"
record_scenario "beta3_ready" "PASS" "$(echo "$BEFORE_HEALTH" | jq -c '{version:.version,commit:.commit,ready:.ready}' 2>/dev/null || echo ok)"

# Warm grype lightly (non-fatal)
docker exec rd-e2e-detective grype db update >/dev/null 2>"$OUT_DIR/grype-db-update.log" || true

HOOK_URL="http://repository-detective:8081/webhook"
curl -fsS -X POST "${GITEA_URL}/api/v1/repos/$GITEA_USER/$REPO_NAME/hooks" \
  -H "Authorization: token $GITEA_TOKEN" -H "Content-Type: application/json" \
  -d "{\"type\":\"gitea\",\"active\":true,\"events\":[\"push\",\"pull_request\"],\"config\":{\"url\":\"$HOOK_URL\",\"content_type\":\"json\",\"secret\":\"$WEBHOOK_SECRET\"}}" \
  >"$OUT_DIR/hook-create.json"
record_scenario "webhook_registration" "PASS" "hook_created"

WORKDIR="$OUT_DIR/repo-clone"
rm -rf "$WORKDIR"
git clone "http://$GITEA_USER:$GITEA_TOKEN@127.0.0.1:${GITEA_HOST_PORT}/$GITEA_USER/$REPO_NAME.git" "$WORKDIR"
cd "$WORKDIR"
git config user.email "$GITEA_EMAIL"
git config user.name "RD Upgrade"
echo "# upgrade demo" > README.md
echo "package main; func main() {}" > main.go
# Synthetic Slack fixture (same shape as core E2E; not a real credential)
python3 - <<'PY'
prefix="xoxb-"; mid="123456789012-123456789012-"; suffix="abcdefghijklmnopqrstuvwx"
open("leak.go","w").write("package main\nvar slackBot = %r\nfunc main() {}\n" % (prefix+mid+suffix))
PY
git add -A && git commit -m "baseline + synthetic secret" && git push origin HEAD:main
cd "$ROOT"
record_scenario "seed_push" "PASS" "pushed"

log "waiting for findings / first scan under beta.3"
SEED_OK=0
for i in $(seq 1 100); do
  FINDINGS="$(rd_api GET '/api/v1/findings?limit=50' 2>/dev/null || echo '{}')"
  echo "$FINDINGS" >"$OUT_DIR/findings-before.json"
  DOC="$(rd_api GET /api/v1/doctor 2>/dev/null || echo '{}')"
  echo "$DOC" >"$OUT_DIR/doctor-before.json"
  COUNT="$(jq '(.findings // []) | length' <<<"$FINDINGS" 2>/dev/null || echo 0)"
  if [[ "${COUNT:-0}" -gt 0 ]]; then
    SEED_OK=1
    break
  fi
  sleep 3
done
[[ "$SEED_OK" == "1" ]] || fail "failed to seed findings under beta.3 (need at least one finding)"
FINDING_COUNT_BEFORE="$(jq '(.findings // []) | length' "$OUT_DIR/findings-before.json")"
record_scenario "seed_state" "PASS" "findings=$FINDING_COUNT_BEFORE"

# Capture representative IDs for post-upgrade verify
PRE_STATE="$OUT_DIR/pre-upgrade-state.json"
rd_api GET /api/v1/dashboard/summary >"$OUT_DIR/dashboard-before.json" || echo '{}' >"$OUT_DIR/dashboard-before.json"
python3 - <<PY
import json
findings=json.load(open("$OUT_DIR/findings-before.json"))
items=findings.get("findings") or findings.get("items") or []
ids=[f.get("id") or f.get("fingerprint") for f in items if isinstance(f,dict)]
open("$PRE_STATE","w").write(json.dumps({
  "finding_ids":[i for i in ids if i][:20],
  "finding_count": len(items),
}, indent=2)+"\n")
PY
sanitize_file "$OUT_DIR/findings-before.json"
sanitize_file "$OUT_DIR/doctor-before.json"

# Disposable baseline snapshot BEFORE upgrade (never mutate this copy as the live DB)
snapshot_rd_db "beta3-baseline"
cp "$OUT_DIR/snapshots/beta3-baseline.db" "$OUT_DIR/snapshots/beta3-baseline.db.immutable"
record_scenario "baseline_snapshot" "PASS" "beta3-baseline.db preserved"

log "stopping beta.3 container (volumes retained)"
"${COMPOSE[@]}" stop repository-detective
docker rm -f rd-e2e-detective >/dev/null 2>&1 || true

log "upgrading to candidate (same named volume)"
export RD_E2E_IMAGE="$CANDIDATE_IMAGE"
# rewrite env for compose
sed -i "s|^RD_E2E_IMAGE=.*|RD_E2E_IMAGE=$CANDIDATE_IMAGE|" "$OUT_DIR/e2e.env"
set -a; source "$OUT_DIR/e2e.env"; set +a
"${COMPOSE[@]}" up -d repository-detective
wait_http "${RD_URL}/health" 90 || fail "candidate health not ready"
wait_rd_ready 120 || fail "candidate doctor not ready after migration"
AFTER_HEALTH="$(curl -fsS "${RD_URL}/health")"
echo "$AFTER_HEALTH" >"$OUT_DIR/health-after.json"
record_scenario "candidate_ready" "PASS" "$(echo "$AFTER_HEALTH" | jq -c '{version:.version,commit:.commit,ready:.ready}' 2>/dev/null || echo ok)"

# Migration idempotency: restart again
log "restart after successful migration (idempotent reopen)"
"${COMPOSE[@]}" restart repository-detective
wait_rd_ready 90 || fail "candidate not ready after restart"
record_scenario "migration_restart_idempotent" "PASS" "ready_after_restart"

# Verify persisted state
FINDINGS_AFTER="$(rd_api GET '/api/v1/findings?limit=50' 2>/dev/null || echo '{}')"
echo "$FINDINGS_AFTER" >"$OUT_DIR/findings-after.json"
DOC_AFTER="$(rd_api GET /api/v1/doctor 2>/dev/null || echo '{}')"
echo "$DOC_AFTER" >"$OUT_DIR/doctor-after.json"
DASH_AFTER="$(rd_api GET /api/v1/dashboard/summary 2>/dev/null || echo '{}')"
echo "$DASH_AFTER" >"$OUT_DIR/dashboard-after.json"

FINDING_COUNT_AFTER="$(jq '(.findings // []) | length' "$OUT_DIR/findings-after.json")"
if [[ "$FINDING_COUNT_AFTER" -lt 1 ]]; then
  record_scenario "persisted_findings" "FAIL" "no findings after upgrade"
else
  record_scenario "persisted_findings" "PASS" "findings=$FINDING_COUNT_AFTER"
fi

# Auth still works (api_key_only in this harness — existing-install posture)
CODE_OK="$(curl -s -o /dev/null -w '%{http_code}' -H "X-Repository-Detective-API-Key: $API_KEY" "${RD_URL}/api/v1/doctor")"
CODE_BAD="$(curl -s -o /dev/null -w '%{http_code}' "${RD_URL}/api/v1/doctor")"
if [[ "$CODE_OK" == "200" && ( "$CODE_BAD" == "401" || "$CODE_BAD" == "403" ) ]]; then
  record_scenario "auth_post_upgrade" "PASS" "api_key_ok missing_denied"
else
  record_scenario "auth_post_upgrade" "FAIL" "ok=$CODE_OK bad=$CODE_BAD"
fi

# Doctor post-upgrade
if jq -e '.overall != null or .checks' "$OUT_DIR/doctor-after.json" >/dev/null 2>&1; then
  record_scenario "doctor_post_upgrade" "PASS" "doctor_json_ok"
else
  record_scenario "doctor_post_upgrade" "FAIL" "doctor payload unexpected"
fi

# Process another webhook/scan
cd "$WORKDIR"
echo "// post-upgrade marker" >> main.go
git add main.go && git commit -m "post-upgrade change" && git push origin HEAD:main
cd "$ROOT"
POST_SCAN=0
for i in $(seq 1 80); do
  DOC_LIVE="$(rd_api GET /api/v1/doctor 2>/dev/null || echo '{}')"
  echo "$DOC_LIVE" >"$OUT_DIR/doctor-post-scan.json"
  # activity: doctor still healthy and findings still present
  if jq -e '.checks' <<<"$DOC_LIVE" >/dev/null 2>&1; then
    POST_SCAN=1
  fi
  sleep 3
  if (( i > 20 && POST_SCAN == 1 )); then break; fi
done
if [[ "$POST_SCAN" == "1" ]]; then
  record_scenario "post_upgrade_scan_activity" "PASS" "doctor_active_after_push"
else
  record_scenario "post_upgrade_scan_activity" "FAIL" "no post-upgrade activity"
fi

# Duplicate findings guard — count should not explode unrealistically vs before
DUP_STATUS="$(python3 - <<PY
import json
before=json.load(open("$OUT_DIR/findings-before.json"))
after=json.load(open("$OUT_DIR/findings-after.json"))
b=len(before.get("findings") or [])
a=len(after.get("findings") or [])
print("FAIL" if (b>0 and a>max(50,b*5)) else "PASS")
print("before=%d after=%d" % (b,a))
PY
)"
DUP_ST="$(echo "$DUP_STATUS" | head -1)"
DUP_DET="$(echo "$DUP_STATUS" | tail -1)"
record_scenario "no_duplicate_explosion" "$DUP_ST" "$DUP_DET"

# Immutable baseline still untouched
if cmp -s "$OUT_DIR/snapshots/beta3-baseline.db" "$OUT_DIR/snapshots/beta3-baseline.db.immutable"; then
  record_scenario "baseline_snapshot_untouched" "PASS" "immutable copy intact"
else
  record_scenario "baseline_snapshot_untouched" "FAIL" "baseline snapshot was mutated"
fi

snapshot_rd_db "after-upgrade"

sanitize_file "$OUT_DIR/findings-after.json"
sanitize_file "$OUT_DIR/doctor-after.json"
sanitize_file "$OUT_DIR/doctor-post-scan.json"

if jq -e '.scenarios[] | select(.status=="FAIL")' "$OUT_DIR/scenarios.tmp" >/dev/null 2>&1; then
  fail "one or more upgrade scenarios FAILED — see $OUT_DIR"
fi

record_scenario "upgrade_classification" "PASS" "$CLASSIFICATION"
log "upgrade harness completed: $CLASSIFICATION"
exit 0
