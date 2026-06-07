#!/usr/bin/env bash
# Build a prepackaged beta release without relying on Gitea Actions.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
OUT="$ROOT/dist/repository-detective-beta"
BIN="$OUT/repository-detective"

rm -rf "$OUT"
mkdir -p "$OUT"

echo "Building repository-detective $VERSION..."
CGO_ENABLED=1 go build -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$BIN" .

( cd "$OUT" && sha256sum repository-detective > checksums.txt )

if command -v cyclonedx-gomod >/dev/null 2>&1; then
  ( cd "$ROOT" && cyclonedx-gomod mod -json -output "$OUT/sbom-go.cdx.json" ) || true
fi

cp config/config.yaml "$OUT/config.example.yaml" 2>/dev/null || cp docs/examples/homelab-minimal.yaml "$OUT/config.example.yaml" 2>/dev/null || true
cp docker-compose.yml "$OUT/docker-compose.beta.yml" 2>/dev/null || true
cp .env.example "$OUT/.env.example" 2>/dev/null || true

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

echo "Beta package ready: $OUT"
ls -la "$OUT"
