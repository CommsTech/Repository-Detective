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
API_KEY="${REPOSITORY_DETECTIVE_API_KEY:-}"

log() { printf '==> %s\n' "$*"; }

# Minimum free space on the filesystem hosting the build context (default 10 GB).
VERIFY_MIN_DISK_GB="${VERIFY_MIN_DISK_GB:-10}"

check_disk_space() {
  local required_kb=$((VERIFY_MIN_DISK_GB * 1024 * 1024))
  local available_kb
  available_kb=$(df -Pk . | awk 'NR==2 {print $4}')
  if [ -z "$available_kb" ] || [ "$available_kb" -lt "$required_kb" ]; then
    echo "ERROR: not enough free disk for Docker build verify." >&2
    echo "Available: $((available_kb / 1024 / 1024)) GB on $(df -Pk . | awk 'NR==2 {print $6}')" >&2
    echo "Required: ${VERIFY_MIN_DISK_GB} GB minimum (${VERIFY_MIN_DISK_GB}–30+ GB recommended for all-in-one builds)" >&2
    echo "Remediation:" >&2
    echo "  docker system df" >&2
    echo "  docker container prune -f" >&2
    echo "  docker builder prune -f" >&2
    echo "  docker image prune -af" >&2
    echo "Do not run 'docker volume prune' if production SQLite lives in an anonymous volume." >&2
    echo "Repository Detective homelab deploy uses bind mount ./data — volume prune is usually safe for RD data." >&2
    exit 1
  fi
  log "disk OK: $((available_kb / 1024 / 1024)) GB free (require ${VERIFY_MIN_DISK_GB} GB)"
}

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
  SMOKE_CONTAINER_NAME="rd-verify-$$"

  # Remove stale verify containers that may still hold VERIFY_PORT.
  docker ps -aq --filter "name=rd-verify-" 2>/dev/null | xargs -r docker rm -f 2>/dev/null || true
  docker rm -f "$SMOKE_CONTAINER_NAME" 2>/dev/null || true
  log "starting smoke container on port $PORT"
  docker run -d --name "$SMOKE_CONTAINER_NAME" \
    -e REPOSITORY_DETECTIVE_PORT="$PORT" \
    -e REPOSITORY_DETECTIVE_LISTEN_HOST=0.0.0.0 \
    -e REPOSITORY_DETECTIVE_PORT="$PORT" \
    -e REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true \
    -e REPOSITORY_DETECTIVE_DATABASE_PATH=/app/data/repository-detective.db \
    -e REPOSITORY_DETECTIVE_GITEA_URL=http://example.com \
    -e REPOSITORY_DETECTIVE_GITEA_TOKEN=verify-smoke \
    -e REPOSITORY_DETECTIVE_API_KEY="${API_KEY:-verify-smoke-key}" \
    -e REPOSITORY_DETECTIVE_GITEA_URL=http://example.com \
    -e REPOSITORY_DETECTIVE_GITEA_TOKEN=verify-smoke \
    -p "${PORT}:${PORT}" \
    "$tag" >/dev/null

  cleanup_smoke() {
    docker rm -f "${SMOKE_CONTAINER_NAME:-}" 2>/dev/null || true
  }
  trap cleanup_smoke EXIT

  for i in $(seq 1 60); do
    if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
      log "/health OK"
      break
    fi
    if [ "$i" -eq 60 ]; then
      docker logs "$SMOKE_CONTAINER_NAME" 2>&1 | tail -40
      echo "health check timed out" >&2
      exit 1
    fi
    sleep 2
  done

  if [ -n "$API_KEY" ]; then
    log "/api/v1/status (scanner tools)"
    curl -sf -H "X-Repository-Detective-API-Key: ${API_KEY}" \
      "http://127.0.0.1:${PORT}/api/v1/status" | head -c 400 || true
    echo
  else
    log "skip /api/v1/status (set REPOSITORY_DETECTIVE_API_KEY to verify)"
  fi

  cleanup_smoke
  trap - EXIT
  unset SMOKE_CONTAINER_NAME
}

main() {
  need_docker
  check_disk_space
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
