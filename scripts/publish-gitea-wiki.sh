#!/usr/bin/env bash
# Sync docs/wiki markdown into the Gitea wiki git repository.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GITEA_URL="${REPOSITORY_DETECTIVE_GITEA_URL:-${REPOSITORY_DETECTIVE_GITEA_URL:-https://git.commsnet.org}}"
OWNER="${REPOSITORY_DETECTIVE_GITEA_OWNER:-commstech}"
REPO="${REPOSITORY_DETECTIVE_GITEA_REPO:-repository-detective}"
TOKEN="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${REPOSITORY_DETECTIVE_GITEA_TOKEN:-}}"
SOURCE_DIR="${WIKI_SOURCE_DIR:-$ROOT/docs/wiki}"
WORK_DIR="${WIKI_WORK_DIR:-$(mktemp -d)}"
DRY_RUN="${WIKI_DRY_RUN:-false}"
WIKI_REMOTE="${WIKI_REMOTE_URL:-}"

log() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

usage() {
  cat <<'EOF'
Usage: publish-gitea-wiki.sh

Environment:
  REPOSITORY_DETECTIVE_GITEA_TOKEN / REPOSITORY_DETECTIVE_GITEA_TOKEN  Wiki write token (required unless DRY_RUN=true)
  WIKI_DRY_RUN=true                                      Prepare only; no clone/push
  WIKI_REMOTE_URL                                        Override wiki git URL
  WIKI_SOURCE_DIR                                        Default: docs/wiki
  KEEP_WIKI_WORKDIR=true                                 Keep temp clone for inspection

Target wiki: https://git.commsnet.org/commstech/repository-detective.wiki.git
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [ ! -d "$SOURCE_DIR" ]; then
  echo "wiki source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

if [ -z "$WIKI_REMOTE" ]; then
  WIKI_REMOTE="${GITEA_URL%/}/${OWNER}/${REPO}.wiki.git"
fi

if [ "$DRY_RUN" != "true" ] && [ -z "$TOKEN" ]; then
  echo "REPOSITORY_DETECTIVE_GITEA_TOKEN (or REPOSITORY_DETECTIVE_GITEA_TOKEN) is required (or set WIKI_DRY_RUN=true)" >&2
  exit 1
fi

AUTH_URL="${WIKI_REMOTE}"
if [ -n "$TOKEN" ]; then
  AUTH_URL="${WIKI_REMOTE/https:\/\//https://oauth2:${TOKEN}@}"
fi

cleanup() {
  if [ "${KEEP_WIKI_WORKDIR:-false}" != "true" ]; then
    rm -rf "$WORK_DIR"
  else
    log "keeping work dir: $WORK_DIR"
  fi
}
trap cleanup EXIT

generate_home() {
  local home="$WORK_DIR/Home.md"
  {
    echo "# Repository Detective wiki"
    echo ""
    echo "Operator documentation synced from the main repository (\`docs/wiki/\`)."
    echo ""
    echo "## Pages"
    echo ""
    for f in "$SOURCE_DIR"/*.md; do
      [ -f "$f" ] || continue
      base="$(basename "$f" .md)"
      [ "$base" = "Home" ] && continue
      title="${base//_/ }"
      echo "- [${title}](${base})"
    done
  } > "$home"
}

log "source: $SOURCE_DIR"
log "wiki remote: $WIKI_REMOTE"

if [ "$DRY_RUN" = "true" ]; then
  log "DRY RUN — pages that would be published:"
  find "$SOURCE_DIR" -maxdepth 1 -name '*.md' -printf '  %f\n' | sort
  generate_home
  log "DRY RUN complete (no git operations)"
  exit 0
fi

if [ -d "$WORK_DIR/.git" ]; then
  log "fetch existing wiki clone"
  git -C "$WORK_DIR" fetch origin
  git -C "$WORK_DIR" checkout master 2>/dev/null || git -C "$WORK_DIR" checkout main
  git -C "$WORK_DIR" pull --ff-only
else
  log "clone wiki repo"
  if ! git clone "$AUTH_URL" "$WORK_DIR" 2>/dev/null; then
    warn "wiki clone failed — initializing new wiki repository"
    git init "$WORK_DIR"
    git -C "$WORK_DIR" checkout -b main 2>/dev/null || git -C "$WORK_DIR" checkout -b master
    git -C "$WORK_DIR" remote add origin "$AUTH_URL"
  fi
fi

log "sync markdown from $SOURCE_DIR"
rsync -a --include='*.md' --include='*/' --exclude='*' "$SOURCE_DIR/" "$WORK_DIR/"
generate_home

if git -C "$WORK_DIR" status --porcelain | grep -q .; then
  git -C "$WORK_DIR" add -A
  git -C "$WORK_DIR" commit -m "Sync operator wiki docs from main repo"
  git -C "$WORK_DIR" push origin HEAD
  log "wiki published successfully"
else
  log "wiki already up to date — no push"
fi
