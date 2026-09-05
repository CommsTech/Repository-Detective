#!/usr/bin/env bash
# Populate vendor/ for offline or DNS-filtered Docker builds.
#
# Recommended module proxy (default):
#   GOPROXY=https://proxy.golang.org,direct
#
# Enterprise / air-gapped:
#   GOPROXY=https://your-internal-artifact-proxy,direct
#   GOSUMDB=sum.golang.org
#
# Fully offline after vendor/ exists:
#   GOPROXY=off
#
# Emergency local workaround only (not recommended for DoD/government environments):
#   GOPROXY=https://goproxy.io,direct
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

GOSUMDB="${GOSUMDB:-sum.golang.org}"

echo "==> vendoring with proxy.golang.org"
if GOPROXY=https://proxy.golang.org,direct GOSUMDB="$GOSUMDB" go mod vendor 2>/tmp/rd-vendor.log; then
  echo "==> vendor/ ready ($(du -sh vendor | cut -f1))"
  exit 0
fi

echo "==> proxy.golang.org failed; retrying with goproxy.io (temporary local workaround only)"
echo "    Not recommended for security-sensitive or government deployments."
GOPROXY=https://goproxy.io,direct GOSUMDB="$GOSUMDB" go mod vendor
echo "==> vendor/ ready ($(du -sh vendor | cut -f1))"
