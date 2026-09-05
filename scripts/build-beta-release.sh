#!/usr/bin/env bash
# Build a prepackaged beta release without relying on Gitea Actions.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
FINAL_OUT="$ROOT/dist/repository-detective-beta"
STAGE="${BETA_RELEASE_STAGE:-$ROOT/dist/.beta-release-staging}"
OUT="$STAGE/repository-detective-beta"
BIN="$OUT/repository-detective"
CLEANUP_STAGE=0

if [[ -z "${BETA_RELEASE_STAGE:-}" ]]; then
  CLEANUP_STAGE=1
  rm -rf "$STAGE"
fi

cleanup() {
  if [[ "$CLEANUP_STAGE" -eq 1 && -n "${STAGE:-}" && -d "$STAGE" ]]; then
    rm -rf "$STAGE"
  fi
}
trap cleanup EXIT

ensure_dist_writable() {
  local dist_dir="$ROOT/dist"
  if [[ -e "$FINAL_OUT" && ! -w "$FINAL_OUT" ]]; then
    echo "ERROR: $FINAL_OUT is not writable (often root-owned from a prior Docker build)." >&2
    echo "Run: make clean-beta-release" >&2
    exit 1
  fi
  if [[ -e "$dist_dir" && ! -w "$dist_dir" ]]; then
    echo "ERROR: $dist_dir is not writable." >&2
    echo "Run: make clean-beta-release" >&2
    exit 1
  fi
}

install_final_package() {
  ensure_dist_writable
  mkdir -p "$ROOT/dist"
  rm -rf "$FINAL_OUT"
  mv "$OUT" "$FINAL_OUT"
  chmod -R u+rwX "$FINAL_OUT" 2>/dev/null || true
}

mkdir -p "$OUT"

echo "Building repository-detective $VERSION (stage: $STAGE)..."

if command -v go >/dev/null 2>&1; then
  CGO_ENABLED=1 go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$BIN" .
else
  echo "go not found locally — building in golang:1.25-bookworm container..."
  rel_out="${OUT#$ROOT/}"
  docker run --rm \
    -v "$ROOT:/src" \
    -w /src \
    -e CGO_ENABLED=1 \
    golang:1.25-bookworm \
    go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "/src/${rel_out}/repository-detective" .
  chown "$(id -u):$(id -g)" "$BIN" 2>/dev/null || \
    docker run --rm -v "$ROOT:/src" alpine:3.20 chown "$(id -u):$(id -g)" "/src/${rel_out}/repository-detective"
fi

( cd "$OUT" && sha256sum repository-detective > checksums.txt )

if command -v cyclonedx-gomod >/dev/null 2>&1; then
  ( cd "$ROOT" && cyclonedx-gomod mod -json -output "$OUT/sbom-go.cdx.json" ) || true
fi

if [[ -f config/private-beta.example.yaml ]]; then
  cp config/private-beta.example.yaml "$OUT/config.example.yaml"
elif [[ -f config/config.yaml.example ]]; then
  cp config/config.yaml.example "$OUT/config.example.yaml"
else
  cp docs/examples/homelab-minimal.yaml "$OUT/config.example.yaml" 2>/dev/null || true
fi
if [[ -f docker-compose.beta.yml ]]; then
  cp docker-compose.beta.yml "$OUT/docker-compose.beta.yml"
else
  cp docker-compose.yml "$OUT/docker-compose.beta.yml" 2>/dev/null || true
fi
cp .env.example "$OUT/.env.example" 2>/dev/null || true

# Never ship live secrets or local databases.
for forbidden in .env repository-detective.db config/config.yaml; do
  if [[ -e "$OUT/$forbidden" ]]; then
    echo "ERROR: forbidden artifact would be packaged: $forbidden" >&2
    exit 1
  fi
done

cat > "$OUT/README_BETA.md" <<EOF
# Repository Detective — Private Beta Package

Version: ${VERSION}
Built: ${BUILD_DATE}

## Quick start

1. Copy \`config.example.yaml\` to \`config/config.yaml\` and set secrets via environment or local file (never commit).
2. Run \`./repository-detective\` or \`docker compose -f docker-compose.beta.yml up -d\`.
3. Open UI at http://127.0.0.1:8081/ui (see config for port).

Verify checksum: \`sha256sum -c checksums.txt\`
EOF

cat > "$OUT/RELEASE_NOTES.md" <<EOF
# Release notes (beta)

- Version: ${VERSION}
- Build date: ${BUILD_DATE}
- See docs/beta/BETA_RELEASE_READINESS_REPORT.md in source repo for readiness status.
EOF

install_final_package

echo "Beta package ready: $FINAL_OUT"
ls -la "$FINAL_OUT"
