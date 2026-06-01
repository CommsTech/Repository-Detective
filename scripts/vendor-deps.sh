#!/usr/bin/env bash
# Populate vendor/ for offline or DNS-filtered Docker builds.
#
# When storage.googleapis.com is blocked, the default Go proxy redirects fail.
# This script tries proxy.golang.org first, then goproxy.io.
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

need_go() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go is not installed. Install Go 1.21+ or use a pre-built image." >&2
    exit 1
  fi
}

need_go

export GOSUMDB=off

echo "==> vendoring with proxy.golang.org"
if GOPROXY=https://proxy.golang.org go mod vendor 2>/tmp/bugbot-vendor.log; then
  echo "==> vendor/ ready ($(du -sh vendor | cut -f1))"
  exit 0
fi

echo "==> proxy.golang.org failed; retrying with goproxy.io"
GOPROXY=https://goproxy.io,direct go mod vendor
echo "==> vendor/ ready ($(du -sh vendor | cut -f1))"
