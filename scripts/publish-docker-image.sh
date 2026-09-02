#!/usr/bin/env bash
# Tag and push Repository Detective images.
#
# Canonical registry: Gitea Package Registry (git.commsnet.org)
# Optional mirror:    GHCR (ghcr.io) for public discovery
#
# Prefer CI (.gitea/workflows/release.yml / docker-publish.yml).
# Use this when you already have a sanitized local image ready.
#
# Examples:
#   ./scripts/publish-docker-image.sh --source repository-detective:ghcr-publish --tag v0.1.0-beta.1
#   ./scripts/publish-docker-image.sh --tag v0.1.0 --mirror-ghcr
#   ./scripts/publish-docker-image.sh --dry-run
#
# Prerequisites:
#   docker login git.commsnet.org -u USER --password-stdin   # Gitea token with package write
#   docker login ghcr.io -u USER --password-stdin            # only if --mirror-ghcr

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

GITEA_REGISTRY="${RD_GITEA_REGISTRY:-git.commsnet.org/commstech}"
GHCR_REGISTRY="${RD_GHCR_REGISTRY:-ghcr.io/commstech}"
IMAGE_NAME="${RD_IMAGE_NAME:-repository-detective}"
SOURCE="${RD_SOURCE_IMAGE:-repository-detective:all-in-one}"
TAG=""
PUSH_LATEST=1
MIRROR_GHCR=0
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage: publish-docker-image.sh [options]

  --source IMAGE     Local image to publish (default: repository-detective:all-in-one)
  --tag TAG          Version tag (default: git describe or short SHA)
  --gitea-registry   Gitea registry/org (default: git.commsnet.org/commstech)
  --no-latest        Do not also push :latest and :all-in-one
  --mirror-ghcr      Also push the same tags to ghcr.io/commstech (public mirror)
  --dry-run          Print tags only
  -h, --help         Show help

Environment:
  RD_GITEA_REGISTRY, RD_GHCR_REGISTRY, RD_IMAGE_NAME, RD_SOURCE_IMAGE, RD_VERSION
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source) SOURCE="$2"; shift 2 ;;
    --tag) TAG="$2"; shift 2 ;;
    --gitea-registry|--registry) GITEA_REGISTRY="$2"; shift 2 ;;
    --no-latest) PUSH_LATEST=0; shift ;;
    --mirror-ghcr) MIRROR_GHCR=1; shift ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$TAG" ]]; then
  TAG="${RD_VERSION:-}"
fi
if [[ -z "$TAG" ]]; then
  if git describe --tags --exact-match >/dev/null 2>&1; then
    TAG="$(git describe --tags --exact-match)"
  else
    TAG="sha-$(git rev-parse --short=12 HEAD)"
  fi
fi

GITEA_REGISTRY="$(echo "$GITEA_REGISTRY" | tr '[:upper:]' '[:lower:]')"
GHCR_REGISTRY="$(echo "$GHCR_REGISTRY" | tr '[:upper:]' '[:lower:]')"
IMAGE_NAME="$(echo "$IMAGE_NAME" | tr '[:upper:]' '[:lower:]')"

if ! docker image inspect "$SOURCE" >/dev/null 2>&1; then
  echo "Source image not found: $SOURCE" >&2
  echo "Build/sanitize first (see docs/DOCKER.md)." >&2
  exit 1
fi

build_tags() {
  local reg="$1"
  local dest="${reg}/${IMAGE_NAME}"
  local tags=("${dest}:${TAG}" "${dest}:${TAG}-all-in-one")
  if [[ "$PUSH_LATEST" -eq 1 ]]; then
    tags+=("${dest}:all-in-one" "${dest}:latest")
  fi
  printf '%s\n' "${tags[@]}"
}

mapfile -t GITEA_TAGS < <(build_tags "$GITEA_REGISTRY")
MIRROR_TAGS=()
if [[ "$MIRROR_GHCR" -eq 1 ]]; then
  mapfile -t MIRROR_TAGS < <(build_tags "$GHCR_REGISTRY")
fi

echo "Source:        $SOURCE"
echo "Primary (Gitea): ${GITEA_TAGS[*]}"
if [[ "$MIRROR_GHCR" -eq 1 ]]; then
  echo "Mirror (GHCR):   ${MIRROR_TAGS[*]}"
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
  exit 0
fi

push_tags() {
  local t
  for t in "$@"; do
    docker tag "$SOURCE" "$t"
    echo "Pushing $t ..."
    docker push "$t"
  done
}

push_tags "${GITEA_TAGS[@]}"
if [[ "$MIRROR_GHCR" -eq 1 ]]; then
  push_tags "${MIRROR_TAGS[@]}"
fi

PRIMARY="${GITEA_REGISTRY}/${IMAGE_NAME}"
echo "Published to Gitea: ${PRIMARY}"
echo "Consumers (canonical):"
echo "  export RD_IMAGE=${PRIMARY}:all-in-one"
echo "  docker login git.commsnet.org   # if package is private"
echo "  docker compose pull && docker compose up -d"
if [[ "$MIRROR_GHCR" -eq 1 ]]; then
  echo "Mirrored to GHCR: ${GHCR_REGISTRY}/${IMAGE_NAME}"
fi
