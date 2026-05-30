#!/usr/bin/env bash
# Install cloudflared to ~/bin (no sudo required).
set -euo pipefail

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  CF_ARCH=amd64 ;;
  aarch64) CF_ARCH=arm64 ;;
  arm64)   CF_ARCH=arm64 ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

mkdir -p "$HOME/bin"
URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${CF_ARCH}"

echo "Downloading cloudflared from $URL"
curl -fsSL "$URL" -o "$HOME/bin/cloudflared"
chmod +x "$HOME/bin/cloudflared"

echo ""
echo "Installed: $HOME/bin/cloudflared"
"$HOME/bin/cloudflared" --version
echo ""
echo "Add to PATH if needed:  export PATH=\"\$HOME/bin:\$PATH\""
echo ""
echo "Quick tunnel to Bugbot:"
echo "  cloudflared tunnel --url http://127.0.0.1:8081"
