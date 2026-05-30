#!/usr/bin/env bash
# Quick Cloudflare tunnel to expose internal Bugbot for Gitea webhooks.
# Usage: ./scripts/setup-cloudflared-tunnel.sh [host:port]
#
# Example:
#   ./scripts/setup-cloudflared-tunnel.sh 192.168.255.11:8081

set -euo pipefail

TARGET="${1:-127.0.0.1:8080}"

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "cloudflared not found. Install it first:"
  echo "  https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
  echo ""
  echo "Debian/Ubuntu:"
  echo "  curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg"
  echo "  echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared \$(lsb_release -cs) main' | sudo tee /etc/apt/sources.list.d/cloudflared.list"
  echo "  sudo apt update && sudo apt install -y cloudflared"
  exit 1
fi

echo "Starting Cloudflare quick tunnel → http://${TARGET}"
echo ""
echo "When the URL appears below, set:"
echo "  BUGBOT_PUBLIC_URL=<https-url-from-cloudflared>"
echo ""
echo "Webhook URL for Gitea:"
echo "  \${BUGBOT_PUBLIC_URL}/webhook"
echo ""

exec cloudflared tunnel --url "http://${TARGET}"
