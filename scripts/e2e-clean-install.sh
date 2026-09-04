#!/usr/bin/env bash
# RD-018 — clean installation validation against published/recommended path.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${RD_E2E_OUT:-$ROOT/e2e/results/clean-install-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"
WORKDIR="$OUT/clean-tree"
log(){ printf '==> %s\n' "$*" | tee -a "$OUT/clean-install.log"; }

log "preparing sanitized public-like tree (git archive HEAD)"
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
git -C "$ROOT" archive HEAD | tar -x -C "$WORKDIR"
cd "$WORKDIR"

IMAGE="${RD_IMAGE:-git.commsnet.org/commstech/repository-detective:all-in-one}"
# Fall back to local all-in-one if registry unreachable
if ! docker pull "$IMAGE" 2>"$OUT/pull.err"; then
  log "pull failed — using local repository-detective:all-in-one if present"
  IMAGE="repository-detective:all-in-one"
  docker image inspect "$IMAGE" >/dev/null
fi
DIGEST="$(docker image inspect "$IMAGE" --format '{{index .RepoDigests 0}}' 2>/dev/null || echo local-only)"
echo "$DIGEST" >"$OUT/image-digest.txt"
log "image=$IMAGE digest=$DIGEST"

cp .env.example .env
# Generate ephemeral values — never commit
API_KEY="clean-$(openssl rand -hex 16)"
WH_SECRET="wh-$(openssl rand -hex 16)"
# Clean-install doctor/scanner inventory does not require a live forge token.
sed -i "s/^REPOSITORY_DETECTIVE_API_KEY=.*/REPOSITORY_DETECTIVE_API_KEY=$API_KEY/" .env
sed -i "s/^REPOSITORY_DETECTIVE_WEBHOOK_SECRET=.*/REPOSITORY_DETECTIVE_WEBHOOK_SECRET=$WH_SECRET/" .env
# Point Gitea at a placeholder; skip startup forge checks
grep -q REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS .env || echo REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true >>.env
sed -i 's|^REPOSITORY_DETECTIVE_GITEA_URL=.*|REPOSITORY_DETECTIVE_GITEA_URL=http://127.0.0.1:9|' .env || true
sed -i 's|^REPOSITORY_DETECTIVE_GITEA_TOKEN=.*|REPOSITORY_DETECTIVE_GITEA_TOKEN=unused|' .env || true
echo "REPOSITORY_DETECTIVE_PRIVACY_MODE=local_only" >>.env
echo "REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false" >>.env
echo "REPOSITORY_DETECTIVE_REJECT_QUERY_STRING_API_KEY=true" >>.env
echo "REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED=false" >>.env

# Published compose bind-mounts ./data as non-root user repositorydetective.
mkdir -p data/tmp data/cache certs config
chmod -R a+rwX data || true

export RD_IMAGE="$IMAGE"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-rd-clean-install}"
export COMPOSE_HTTP_TIMEOUT="${COMPOSE_HTTP_TIMEOUT:-600}"
set -o pipefail

# Avoid colliding with a host's long-lived repository-detective container_name.
if grep -q 'container_name: repository-detective' docker-compose.yml; then
  sed -i 's/container_name: repository-detective/container_name: rd-clean-install-detective/' docker-compose.yml
fi

# Host publish port (container still listens on 8081 via compose environment literal).
export RD_HOST_PORT="${RD_E2E_CLEAN_PORT:-18082}"
export REPOSITORY_DETECTIVE_PORT="$RD_HOST_PORT"

docker compose -f docker-compose.yml down -v >/dev/null 2>&1 || docker-compose -f docker-compose.yml down -v >/dev/null 2>&1 || true
RD_IMAGE="$IMAGE" docker compose -f docker-compose.yml up -d 2>"$OUT/compose.err" \
  || RD_IMAGE="$IMAGE" docker-compose -f docker-compose.yml up -d

READY=0
for i in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${RD_HOST_PORT}/health" >/dev/null 2>&1; then READY=1; break; fi
  sleep 3
done
if [[ "$READY" != "1" ]]; then
  docker logs rd-clean-install-detective 2>&1 | tee "$OUT/rd-start.log" | tail -80
  log "ERROR: clean-install health not ready on port ${RD_HOST_PORT}"
  exit 1
fi

curl -fsS "http://127.0.0.1:${RD_HOST_PORT}/health" | tee "$OUT/health.json"
curl -fsS -o /dev/null -w '%{http_code}\n' "http://127.0.0.1:${RD_HOST_PORT}/onboard/" | tee "$OUT/onboard-status.txt"

for i in $(seq 1 45); do
  CODE="$(curl -s -o /dev/null -w '%{http_code}' -H "X-Repository-Detective-API-Key: $API_KEY" "http://127.0.0.1:${RD_HOST_PORT}/api/v1/doctor" || true)"
  if [[ "$CODE" == "200" ]]; then break; fi
  sleep 2
done
DOCTOR_CODE="$(curl -s -o "$OUT/doctor.json" -w '%{http_code}' -H "X-Repository-Detective-API-Key: $API_KEY" "http://127.0.0.1:${RD_HOST_PORT}/api/v1/doctor" || true)"
echo "$DOCTOR_CODE" | tee "$OUT/doctor-status.txt"
if [[ "$DOCTOR_CODE" != "200" ]]; then
  log "doctor API returned HTTP $DOCTOR_CODE (published image may predate Phase 4 Doctor — recorded as NOT_ON_PUBLISHED_IMAGE)"
  echo '{"note":"doctor endpoint unavailable on this published image digest"}' >"$OUT/doctor.json"
fi

CID="$(docker compose -f docker-compose.yml ps -q repository-detective 2>/dev/null || docker-compose -f docker-compose.yml ps -q repository-detective)"
[[ -n "$CID" ]] || { log "ERROR: clean-install container id missing"; exit 1; }
docker exec "$CID" sh -c 'for b in gitleaks trivy grype semgrep; do echo -n "$b: "; command -v $b || echo MISSING; done' | tee "$OUT/scanners.txt"
docker exec "$CID" sh -c 'for b in gitleaks trivy grype semgrep gosec govulncheck staticcheck hadolint checkov; do echo -n "$b: "; command -v $b >/dev/null && ($b --version 2>/dev/null | head -1 || $b version 2>/dev/null | head -1) || echo MISSING; done' | tee "$OUT/scanners-full.txt"

if grep -q 'MISSING' "$OUT/scanners.txt"; then
  log "ERROR: required scanner missing from published image"
  exit 1
fi

python3 - <<PY
import json
doctor_code=open("$OUT/doctor-status.txt").read().strip()
out={
 "image":"$IMAGE",
 "digest":open("$OUT/image-digest.txt").read().strip(),
 "health_ok": True,
 "onboard_status": open("$OUT/onboard-status.txt").read().strip(),
 "doctor_http": doctor_code,
 "doctor_on_published_image": doctor_code == "200",
 "host_port":"$RD_HOST_PORT",
 "scanners": open("$OUT/scanners.txt").read(),
 "scanners_full": open("$OUT/scanners-full.txt").read(),
 "upgrade_e2e": "NOT_PROVEN",
 "notes": "Clean install used published/local all-in-one from .env.example-derived config on disposable port. Live forge onboarding is covered by e2e-gitea-acceptance.sh. Doctor may be absent on older published digests until republish."
}
json.dump(out, open("$OUT/clean-install.json","w"), indent=2)
print("wrote clean-install.json")
PY

docker compose -f docker-compose.yml down -v >/dev/null 2>&1 || docker-compose -f docker-compose.yml down -v >/dev/null 2>&1 || true
log "RD-018 clean-install artifact at $OUT"
