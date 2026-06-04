#!/usr/bin/env bash
# Build all Dockerfile targets and smoke-test all-in-one /health (and /api/v1/status when API key set).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${RD_VERSION:-dev}"
COMMIT="${RD_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo local)}"
BUILD_DATE="${RD_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
INSTALL="${INSTALL_EXTERNAL_TOOLS:-true}"
PORT="${VERIFY_PORT:-18081}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-${BUGBOT_API_KEY:-}}"

log() { printf '==> %s\n' "$*"; }

need_docker() {
  command -v docker >/dev/null 2>&1 || { echo "docker not available" >&2; exit 1; }
}

build_target() {
  local target=$1 tag=$2
  log "docker build --target $target -t $tag"
  docker build --target "$target" -t "$tag" \
    --build-arg INSTALL_EXTERNAL_TOOLS="$INSTALL" \
    --build-arg VERSION="$VERSION" \
    --build-arg COMMIT="$COMMIT" \
    --build-arg BUILD_DATE="$BUILD_DATE" \
    .
}

verify_no_secrets_in_image() {
  local tag=$1
  log "checking image $tag does not contain .env"
  if docker run --rm "$tag" sh -c 'test -f /.env -o -f /app/.env' 2>/dev/null; then
    echo "secret file found in image" >&2
    exit 1
  fi
  if docker history "$tag" 2>/dev/null | grep -qi 'REPOSITORY_DETECTIVE_GITEA_TOKEN=.'; then
    echo "token-like build arg in history" >&2
    exit 1
  fi
}

smoke_all_in_one() {
  local tag=repository-detective:all-in-one-verify
  local cname=rd-verify-$$

  docker rm -f "$cname" 2>/dev/null || true
  log "starting smoke container on port $PORT"
  docker run -d --name "$cname" \
    -e REPOSITORY_DETECTIVE_PORT="$PORT" \
    -e BUGBOT_PORT="$PORT" \
    -e REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true \
    -e REPOSITORY_DETECTIVE_DATABASE_PATH=/app/data/bugbot.db \
    -p "${PORT}:${PORT}" \
    "$tag" >/dev/null

  trap 'docker rm -f "$cname" 2>/dev/null || true' EXIT

  for i in $(seq 1 60); do
    if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
      log "/health OK"
      break
    fi
    if [ "$i" -eq 60 ]; then
      docker logs "$cname" 2>&1 | tail -40
      echo "health check timed out" >&2
      exit 1
    fi
    sleep 2
  done

  if [ -n "$API_KEY" ]; then
    log "/api/v1/status (scanner tools)"
    curl -sf -H "Authorization: Bearer ${API_KEY}" "http://127.0.0.1:${PORT}/api/v1/status" | head -c 400
    echo
  else
    log "skip /api/v1/status (set REPOSITORY_DETECTIVE_API_KEY to verify)"
  fi

  docker rm -f "$cname" >/dev/null
  trap - EXIT
}

main() {
  need_docker
  build_target core repository-detective:core
  build_target runner repository-detective:runner
  build_target all-in-one repository-detective:all-in-one

  for tag in repository-detective:core repository-detective:runner repository-detective:all-in-one; do
    verify_no_secrets_in_image "$tag"
  done

  docker tag repository-detective:all-in-one repository-detective:all-in-one-verify
  smoke_all_in_one

  log "all Docker targets built and smoke-tested"
}

main "$@"
