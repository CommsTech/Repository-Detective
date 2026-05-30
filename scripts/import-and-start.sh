#!/usr/bin/env bash
# Run on 192.168.255.10 (clustermgr) after the image tar is copied here.
#
#   cd /home/commstech/bugbot
#   ./scripts/import-and-start.sh [gitea-bugbot-image.tar]

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TAR="${1:-gitea-bugbot-image.tar}"
COMPOSE_FILE="docker-compose.clustermgr.offline.yml"

if [[ ! -f "${TAR}" ]]; then
  echo "Missing ${TAR} — copy it from the ai host (.11) first."
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "Missing .env — copy from .env.clustermgr.example and edit."
  exit 1
fi

if ! command -v docker-compose >/dev/null 2>&1; then
  echo "docker-compose not found (need legacy 1.x on clustermgr)."
  exit 1
fi

echo "==> Loading image from ${TAR}"
docker load -i "${TAR}"

echo "==> Starting Bugbot (legacy docker-compose)"
docker-compose -f "${COMPOSE_FILE}" down 2>/dev/null || true
docker-compose -f "${COMPOSE_FILE}" up -d

echo "==> Waiting for health"
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS -m 3 http://127.0.0.1:8081/health >/dev/null 2>&1; then
    echo "OK"
    curl -s -m 3 http://127.0.0.1:8081/health
    echo ""
    docker-compose -f "${COMPOSE_FILE}" ps
    exit 0
  fi
  sleep 2
done

echo "Health check did not pass yet — check logs:"
echo "  docker logs gitea-bugbot --tail 50"
exit 1
