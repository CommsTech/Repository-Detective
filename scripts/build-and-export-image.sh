#!/usr/bin/env bash
# Run on 192.168.255.11 (ai host) where Docker can reach the Go module proxy.
#
#   cd /path/to/bugbot
#   git pull   # optional — use commit a52646e or later
#   ./scripts/build-and-export-image.sh
#
# Then copy gitea-bugbot-image.tar to clustermgr (.10):
#   scp gitea-bugbot-image.tar commstech@192.168.255.10:/home/commstech/bugbot/

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

IMAGE="gitea-bugbot:latest"
TAR="${1:-gitea-bugbot-image.tar}"

echo "==> Building ${IMAGE} from ${ROOT}"
docker build -t "${IMAGE}" .

echo "==> Saving to ${TAR}"
docker save "${IMAGE}" -o "${TAR}"

ls -lh "${TAR}"
echo ""
echo "Copy to clustermgr:"
echo "  scp ${TAR} commstech@192.168.255.10:/home/commstech/bugbot/"
echo ""
echo "On clustermgr:"
echo "  cd /home/commstech/bugbot && ./scripts/import-and-start.sh"
