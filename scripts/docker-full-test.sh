#!/usr/bin/env bash
# Full Docker build and smoke test in an isolated worktree (own directory).
# Usage: ./scripts/docker-full-test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${DOCKER_TEST_WORK:-/tmp/repository-detective-docker-test}"
IMAGE="${DOCKER_TEST_IMAGE:-repository-detective:docker-test}"
PORT="${DOCKER_TEST_PORT:-18081}"

echo "==> Isolated worktree at ${WORK}"
rm -rf "${WORK}"
mkdir -p "${WORK}"
rsync -a --delete \
  --exclude '.git' \
  --exclude 'data' \
  --exclude 'vendor' \
  --exclude '.env' \
  "${ROOT}/" "${WORK}/"

echo "==> Docker build (INSTALL_EXTERNAL_TOOLS=true)"
docker build -t "${IMAGE}" \
  --build-arg INSTALL_EXTERNAL_TOOLS=true \
  "${WORK}"

echo "==> Run tests inside builder-equivalent (Go container)"
docker run --rm -v "${WORK}:/src" -w /src golang:1.25-bookworm \
  go test ./... -count=1

echo "==> Smoke container"
cid=$(docker run -d --rm \
  -p "${PORT}:8081" \
  -e REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true \
  -e REPOSITORY_DETECTIVE_PORT=8081 \
  -e REPOSITORY_DETECTIVE_LISTEN_HOST=0.0.0.0 \
  -e REPOSITORY_DETECTIVE_GITEA_URL=http://example.com \
  -e REPOSITORY_DETECTIVE_GITEA_TOKEN=test-token \
  -e REPOSITORY_DETECTIVE_AI_PROVIDER=ollama \
  -e REPOSITORY_DETECTIVE_AI_BASE_URL=http://127.0.0.1:11434/v1 \
  "${IMAGE}")

cleanup() { docker stop "${cid}" >/dev/null 2>&1 || true; }
trap cleanup EXIT

for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null; then
    echo "==> Health OK on port ${PORT}"
    curl -sf "http://127.0.0.1:${PORT}/health" | head -c 400
    echo
    echo "Docker full test passed."
    exit 0
  fi
  sleep 1
done

echo "Health check failed on port ${PORT}" >&2
docker logs "${cid}" 2>&1 | tail -40
exit 1
