#!/usr/bin/env bash
# Remove generated beta release artifacts, including root-owned dist/ from Docker builds.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="$ROOT/dist/repository-detective-beta"
DIST="$ROOT/dist"

remove_path() {
  local path="$1"
  if [[ ! -e "$path" ]]; then
    return 0
  fi
  if rm -rf "$path" 2>/dev/null; then
    echo "Removed $path"
    return 0
  fi
  if command -v docker >/dev/null 2>&1; then
    echo "Using Docker to remove root-owned path: $path"
    docker run --rm -v "$ROOT:/work" alpine:3.20 sh -c "rm -rf \"/work/${path#$ROOT/}\""
    echo "Removed $path via Docker"
    return 0
  fi
  echo "ERROR: cannot remove $path — run as root or install Docker." >&2
  return 1
}

remove_path "$TARGET"
remove_path "$ROOT/dist/.beta-release-staging"
if [[ -d "$DIST" && -z "$(ls -A "$DIST" 2>/dev/null || true)" ]]; then
  remove_path "$DIST" || true
fi
