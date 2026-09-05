#!/usr/bin/env bash
# Generate container SBOM for an exact image reference (digest preferred).
# Fails if output would be empty or target is missing.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

IMAGE="${1:?usage: $0 <image-ref> [version-label]}"
LABEL="${2:-sbom}"
OUT_DIR="${RD_SBOM_OUT:-$ROOT/docs/release/sbom}"
mkdir -p "$OUT_DIR"

command -v syft >/dev/null || { echo "syft required" >&2; exit 1; }

STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SYFT_VER="$(syft version -o json 2>/dev/null | jq -r '.version // empty' || syft version 2>/dev/null | head -1)"
DIGEST="$(docker image inspect "$IMAGE" --format '{{index .RepoDigests 0}}' 2>/dev/null || echo "local-only:$IMAGE")"

SPDX="$OUT_DIR/repository-detective-${LABEL}.spdx.json"
CDX="$OUT_DIR/repository-detective-${LABEL}.cdx.json"
META="$OUT_DIR/GENERATION.md"

echo "==> syft $IMAGE"
syft "$IMAGE" -o "spdx-json=$SPDX" -o "cyclonedx-json=$CDX"
[[ -s "$SPDX" && -s "$CDX" ]] || { echo "ERROR: empty SBOM output" >&2; exit 1; }

(
  cd "$OUT_DIR"
  sha256sum "repository-detective-${LABEL}.spdx.json" "repository-detective-${LABEL}.cdx.json" > SHA256SUMS
)

cat >"$META" <<EOF
# SBOM generation record

| Field | Value |
|-------|-------|
| Target image | \`$IMAGE\` |
| Repo digest | \`$DIGEST\` |
| Formats | SPDX JSON, CycloneDX JSON |
| Generator | Syft |
| Syft version | \`$SYFT_VER\` |
| Timestamp (UTC) | \`$STAMP\` |
| Scope | Container filesystem inventory (OS packages + installed software as reported by Syft) |

These files are **container SBOMs**. They are not a substitute for claiming SLSA provenance or signatures.
EOF

echo "wrote $SPDX $CDX $META"
cat "$OUT_DIR/SHA256SUMS"
