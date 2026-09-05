#!/usr/bin/env bash
# Publish a sanitized GitHub release tag + GitHub Release for a Gitea release.
# Does NOT overwrite existing GitHub tags (immutable).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TAG=""
SOURCE_COMMIT=""
ALSO_MAIN=0
DRY_RUN=0
NOTES_FILE=""

usage() {
  cat <<'EOF'
Usage: publish-github-release-snapshot.sh --tag vX.Y.Z --source-commit <sha> [options]

  --tag TAG              Release tag (e.g. v0.1.0-beta.3)
  --source-commit SHA    Canonical Gitea commit baked into the release image
  --notes-file PATH      Release notes markdown (default: docs/release/GITHUB_RELEASE_<tag>.md)
  --also-refresh-main    Also force-update GitHub main to this snapshot (rare)
  --dry-run              Print actions only
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) TAG="$2"; shift 2 ;;
    --source-commit) SOURCE_COMMIT="$2"; shift 2 ;;
    --notes-file) NOTES_FILE="$2"; shift 2 ;;
    --also-refresh-main) ALSO_MAIN=1; shift ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ -n "$TAG" && -n "$SOURCE_COMMIT" ]] || { usage >&2; exit 2; }
NOTES_FILE="${NOTES_FILE:-$ROOT/docs/release/GITHUB_RELEASE_${TAG}.md}"
DEPLOY_KEY="${REPOSITORY_DETECTIVE_GITHUB_DEPLOY_KEY:-$HOME/.ssh/repository-detective-github-deploy}"

SOURCE_COMMIT="$(git rev-parse "$SOURCE_COMMIT")"
SHORT="$(git rev-parse --short "$SOURCE_COMMIT")"

if git ls-remote --tags github "refs/tags/$TAG" 2>/dev/null | grep -q "$TAG"; then
  echo "ERROR: GitHub tag $TAG already exists — refusing to overwrite" >&2
  exit 1
fi

[[ -f "$NOTES_FILE" ]] || { echo "missing notes: $NOTES_FILE" >&2; exit 1; }

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

echo "==> sanitized snapshot of $SHORT for GitHub tag $TAG"
git archive "$SOURCE_COMMIT" | tar -x -C "$tmp"
(
  cd "$tmp"
  git init -q
  git checkout -q -b release-snapshot
  git add -A
  git -c user.email="release@commsnet.org" -c user.name="Repository Detective Release" \
    commit -qm "Public sanitized snapshot for ${TAG} (Gitea ${SHORT})

Canonical: https://git.commsnet.org/commstech/repository-detective
Source commit: ${SOURCE_COMMIT}
"
  git tag -a "$TAG" -m "Repository Detective ${TAG} (sanitized public snapshot of ${SHORT})"
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "DRY-RUN: would push tag $TAG (tree $(git rev-parse HEAD^{tree}))"
    exit 0
  fi
  if [[ -f "$DEPLOY_KEY" ]]; then
    export GIT_SSH_COMMAND="ssh -i $DEPLOY_KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
    git push git@github.com-repository-detective:CommsTech/Repository-Detective.git "refs/tags/$TAG"
    if [[ "$ALSO_MAIN" == "1" ]]; then
      git push --force git@github.com-repository-detective:CommsTech/Repository-Detective.git HEAD:main
    fi
  else
    echo "ERROR: deploy key required at $DEPLOY_KEY" >&2
    exit 1
  fi
)

echo "==> creating GitHub Release $TAG"
if [[ "$DRY_RUN" == "1" ]]; then
  echo "DRY-RUN: gh release create $TAG"
  exit 0
fi
gh release create "$TAG" \
  --repo CommsTech/Repository-Detective \
  --title "Repository Detective ${TAG}" \
  --notes-file "$NOTES_FILE"

echo "OK: GitHub tag + release $TAG published (immutable)"
