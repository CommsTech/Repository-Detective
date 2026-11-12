#!/usr/bin/env bash
# Release regression checks — run before tagging a beta or release.
# Usage: ./scripts/release-test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { printf '==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

need_go() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  log "go not in PATH — trying Docker golang:1.25-alpine"
  export USE_DOCKER_GO=1
}

run_go() {
  if [ "${USE_DOCKER_GO:-0}" = "1" ]; then
    docker run --rm -v "$ROOT:/app" -w /app golang:1.25-alpine sh -c "$1"
  else
    bash -c "$1"
  fi
}

main() {
  need_go

  log "go test ./..."
  run_go 'go test ./... -count=1'

  log "go vet ./..."
  run_go 'go vet ./...'

  log "staticcheck ./..."
  run_go 'if ! command -v staticcheck >/dev/null 2>&1; then go install honnef.co/go/tools/cmd/staticcheck@v0.5.1; fi; staticcheck ./...'

  if command -v gosec >/dev/null 2>&1; then
    log "gosec ./..."
    gosec -quiet ./... || fail "gosec reported issues"
  else
    log "gosec skipped (not installed — optional; install: go install github.com/securego/gosec/v2/cmd/gosec@latest)"
  fi

  if command -v docker >/dev/null 2>&1; then
    log "docker build verify"
    "$ROOT/scripts/docker-build-verify.sh"
  else
    log "docker-build-verify.sh skipped (docker not available)"
  fi

  log "release-test: all required checks passed"
}

main "$@"
