#!/usr/bin/env bash
# One-command Repository-Detective / Repository Detective deployment.
#
# Usage:
#   ./deploy.sh              # build from Dockerfile + start on port 8081
#   ./deploy.sh --stop       # stop container
#   ./deploy.sh --restart    # restart container
#   ./deploy.sh --status     # health + container status
#   ./deploy.sh --scan       # trigger self-scan on commstech/Repository-Detective
#   ./deploy.sh --scan-all        # full scans on every Gitea + GitHub repo (global profile)
#   ./deploy.sh --scan-all-quick  # fast profile scans on every repo
#   FORGE=github ./deploy.sh --scan-all   # GitHub repos only
#   ./deploy.sh --webhooks-all  # register push/PR webhooks on all visible repos
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

COMPOSE_FILE="docker-compose.yml"
COMPOSE=(docker-compose -f "$COMPOSE_FILE")
CONTAINER="repository-detective"
HEALTH_URL="http://127.0.0.1:8081/health"
LEGACY_DIR="${REPOSITORY_DETECTIVE_LEGACY_DIR:-$HOME/repository-detective}"

log() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }
}

ensure_env() {
  if [[ ! -f .env ]]; then
    if [[ -f "$LEGACY_DIR/.env" ]]; then
      log "copying .env from $LEGACY_DIR"
      cp "$LEGACY_DIR/.env" .env
    else
      log "creating .env from .env.example — edit secrets before production use"
      cp .env.example .env
    fi
  fi
}

ensure_dirs() {
  mkdir -p build data certs config
}

ensure_vendor() {
  if [[ -f vendor/modules.txt ]]; then
    return
  fi
  if [[ -x scripts/vendor-deps.sh ]]; then
    log "vendor/ missing — generating for offline-friendly Docker build"
    bash scripts/vendor-deps.sh
    return
  fi
  warn "vendor/ missing; Docker build needs network access to proxy.golang.org"
}

build_image() {
  ensure_vendor
  log "building Docker image (target=all-in-one, external scanner tools enabled)"
  "${COMPOSE[@]}" build --build-arg INSTALL_EXTERNAL_TOOLS=true
}

ensure_certs() {
  if [[ -d "$LEGACY_DIR/certs" ]] && [[ -z "$(ls -A certs 2>/dev/null || true)" ]]; then
    log "copying TLS certs from $LEGACY_DIR/certs"
    cp -a "$LEGACY_DIR/certs/." certs/
  fi
}

migrate_legacy_config() {
  if [[ -f "$LEGACY_DIR/config/config.yaml" ]] && [[ ! -f config/config.yaml ]]; then
    log "copying config from legacy install"
    cp "$LEGACY_DIR/config/config.yaml" config/config.yaml
  fi
}

stop_legacy_process() {
  if pgrep -f '/home/commstech/Repository-Detective/(repository-detective|repository-detective)' >/dev/null 2>&1; then
    log "stopping legacy non-Docker Repository Detective process"
    pkill -f '/home/commstech/Repository-Detective/(repository-detective|repository-detective)' || true
    sleep 2
  fi
}

install_systemd_wrapper() {
  local run_sh="$LEGACY_DIR/run.sh"
  if [[ ! -f "$run_sh" ]]; then
    return
  fi
  if grep -q 'docker-compose.yml' "$run_sh" 2>/dev/null; then
    return
  fi
  log "updating $run_sh to manage Docker (disables legacy binary on systemd restart)"
  cat > "$run_sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
cd "$ROOT"
exec docker-compose -f docker-compose.yml up -d --remove-orphans
EOF
  chmod +x "$run_sh"
  warn "run 'sudo systemctl disable repository-detective.service' (legacy unit) when ready and rely on Docker restart policy instead"
}

start_stack() {
  stop_legacy_process
  log "starting $CONTAINER"
  "${COMPOSE[@]}" up -d --remove-orphans
}

wait_healthy() {
  log "waiting for $HEALTH_URL"
  for _ in $(seq 1 30); do
    if curl -sf -m 3 "$HEALTH_URL" >/dev/null 2>&1; then
      curl -s "$HEALTH_URL"
      echo
      return 0
    fi
    sleep 2
  done
  warn "health check timed out — see: docker-compose -f $COMPOSE_FILE logs --tail=50"
  return 1
}

register_webhook() {
  # shellcheck disable=SC1091
  set -a && source .env && set +a
  local api_key="${REPOSITORY_DETECTIVE_API_KEY:-${REPOSITORY_DETECTIVE_API_KEY:-}}"
  local gitea_url="${REPOSITORY_DETECTIVE_GITEA_URL:-${REPOSITORY_DETECTIVE_GITEA_URL:-}}"
  local gitea_token="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}"
  local public_url="${REPOSITORY_DETECTIVE_PUBLIC_URL:-${REPOSITORY_DETECTIVE_PUBLIC_URL:-}}"
  local webhook_secret="${REPOSITORY_DETECTIVE_WEBHOOK_SECRET:-${REPOSITORY_DETECTIVE_WEBHOOK_SECRET:-}}"
  local owner="${REPOSITORY_DETECTIVE_REPO_OWNER:-commstech}"
  local repo="${REPOSITORY_DETECTIVE_REPO_NAME:-Repository-Detective}"

  [[ -n "$gitea_url" && -n "$gitea_token" && -n "$public_url" && -n "$webhook_secret" ]] || {
    warn "skipping webhook registration — set Gitea URL, token, public URL, and webhook secret in .env"
    return 0
  }

  log "registering webhook for $owner/$repo -> ${public_url%/}/webhook"
  curl -sf -X POST "$gitea_url/api/v1/repos/$owner/$repo/hooks" \
    -H "Authorization: token $gitea_token" \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"gitea\",\"config\":{\"url\":\"${public_url%/}/webhook\",\"content_type\":\"json\",\"secret\":\"$webhook_secret\"},\"events\":[\"push\",\"pull_request\"],\"active\":true}" \
    >/dev/null 2>&1 && log "webhook registered" || warn "webhook registration failed (may already exist)"
}

trigger_scan() {
  # shellcheck disable=SC1091
  set -a && source .env && set +a
  local api_key="${REPOSITORY_DETECTIVE_API_KEY:-${REPOSITORY_DETECTIVE_API_KEY:-}}"
  local public_url="${REPOSITORY_DETECTIVE_PUBLIC_URL:-${REPOSITORY_DETECTIVE_PUBLIC_URL:-http://127.0.0.1:8081}}"
  local owner="${REPOSITORY_DETECTIVE_REPO_OWNER:-commstech}"
  local repo="${REPOSITORY_DETECTIVE_REPO_NAME:-Repository-Detective}"

  [[ -n "$api_key" ]] || { warn "REPOSITORY_DETECTIVE_API_KEY not set"; return 1; }

  log "triggering scan on $owner/$repo@main"
  curl -sf -X POST "${public_url%/}/api/v1/analyze" \
    -H "X-Repository-Detective-API-Key: $api_key" \
    -H "Content-Type: application/json" \
    -d "{\"owner\":\"$owner\",\"repository\":\"$repo\",\"ref\":\"main\"}"
  echo
}

trigger_scan_all() {
  local profile="${1:-}"
  # shellcheck disable=SC1091
  set -a && source .env && set +a
  local api_key="${REPOSITORY_DETECTIVE_API_KEY:-${REPOSITORY_DETECTIVE_API_KEY:-}}"
  local public_url="${REPOSITORY_DETECTIVE_PUBLIC_URL:-${REPOSITORY_DETECTIVE_PUBLIC_URL:-http://127.0.0.1:8081}}"
  [[ -n "$api_key" ]] || { warn "REPOSITORY_DETECTIVE_API_KEY not set"; return 1; }

  local org=""
  if [[ -f .env ]]; then
    org="$(grep -E '^[[:space:]]*GITEA_SCAN_ORGS=' .env 2>/dev/null | tail -1 | cut -d= -f2- | tr -d " \r\"'" | cut -d, -f1 || true)"
  fi
  local body
  if [[ -n "$org" && -n "$profile" ]]; then
    body=$(printf '{"orgs":["%s"],"scan_profile":"%s"}' "$org" "$profile")
    log "queueing $profile scans (user repos + org $org)"
  elif [[ -n "$org" ]]; then
    body=$(printf '{"orgs":["%s"]}' "$org")
    log "queueing scans (user repos + org $org)"
  elif [[ -n "$profile" ]]; then
    body=$(printf '{"scan_profile":"%s"}' "$profile")
    log "queueing $profile scans on all user-visible Gitea repositories"
  else
    body='{}'
    log "queueing scans on all user-visible Gitea and GitHub repositories"
  fi
  if [[ -n "${FORGE:-}" ]]; then
    if [[ -n "$profile" ]]; then
      body=$(printf '{"forge":"%s","scan_profile":"%s"}' "$FORGE" "$profile")
    else
      body=$(printf '{"forge":"%s"}' "$FORGE")
    fi
  fi
  curl -sf -X POST "${public_url%/}/api/v1/analyze/all" \
    -H "X-Repository-Detective-API-Key: $api_key" \
    -H "Content-Type: application/json" \
    -d "$body"
  echo
}

register_webhooks_all() {
  # shellcheck disable=SC1091
  set -a && source .env && set +a
  local gitea_url="${REPOSITORY_DETECTIVE_GITEA_URL:-${REPOSITORY_DETECTIVE_GITEA_URL:-}}"
  local gitea_token="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}"
  local public_url="${REPOSITORY_DETECTIVE_PUBLIC_URL:-${REPOSITORY_DETECTIVE_PUBLIC_URL:-}}"
  local webhook_secret="${REPOSITORY_DETECTIVE_WEBHOOK_SECRET:-${REPOSITORY_DETECTIVE_WEBHOOK_SECRET:-}}"

  [[ -n "$gitea_url" && -n "$gitea_token" && -n "$public_url" && -n "$webhook_secret" ]] || {
    warn "skipping webhook registration — set Gitea URL, token, public URL, and webhook secret in .env"
    return 1
  }

  local webhook_url="${public_url%/}/webhook"
  local created=0 failed=0

  log "listing repositories from Gitea"
  mapfile -t repos < <(
    GITEA_URL="${gitea_url%/}" GITEA_TOKEN="$gitea_token" python3 <<'PY'
import json, os, urllib.request

base = os.environ["GITEA_URL"]
token = os.environ["GITEA_TOKEN"]
names = []
page = 1
while True:
    req = urllib.request.Request(
        f"{base}/api/v1/user/repos?limit=50&page={page}",
        headers={"Authorization": f"token {token}"},
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        batch = json.load(resp)
    if not batch:
        break
    names.extend(r["full_name"] for r in batch)
    if len(batch) < 50:
        break
    page += 1
print("\n".join(names))
PY
  )

  for full_name in "${repos[@]}"; do
    [[ -n "$full_name" ]] || continue
    IFS=/ read -r owner repo <<<"$full_name"
    if curl -sf -X POST "${gitea_url%/}/api/v1/repos/$owner/$repo/hooks" \
      -H "Authorization: token $gitea_token" \
      -H "Content-Type: application/json" \
      -d "{\"type\":\"gitea\",\"config\":{\"url\":\"$webhook_url\",\"content_type\":\"json\",\"secret\":\"$webhook_secret\"},\"events\":[\"push\",\"pull_request\"],\"active\":true}" \
      >/dev/null 2>&1; then
      created=$((created + 1))
    else
      failed=$((failed + 1))
    fi
  done

  log "webhooks: $created created, $failed failed or already present (of ${#repos[@]} repos)"
}

cmd="${1:-deploy}"

need_cmd docker
need_cmd docker-compose
need_cmd curl

case "$cmd" in
  --stop)
    "${COMPOSE[@]}" down
    ;;
  --restart)
    "${COMPOSE[@]}" restart
    wait_healthy || true
    ;;
  --status)
    docker ps --filter "name=$CONTAINER"
    curl -s -m 5 "$HEALTH_URL" || true
    echo
    ;;
  --scan)
    trigger_scan
    ;;
  --scan-all)
    trigger_scan_all
    ;;
  --scan-all-quick)
    trigger_scan_all fast
    ;;
  --webhooks-all)
    register_webhooks_all
    ;;
  deploy|--deploy|"")
    ensure_dirs
    ensure_env
    migrate_legacy_config
    ensure_certs
    install_systemd_wrapper
    build_image
    start_stack
    wait_healthy || true
    register_webhook
    log "done — UI: http://127.0.0.1:8081/ui  onboard: http://127.0.0.1:8081/onboard"
    log "run ./deploy.sh --scan to dogfood this repository"
    log "run ./deploy.sh --scan-all to scan every repo your Gitea/GitHub tokens can access"
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 1
    ;;
esac
