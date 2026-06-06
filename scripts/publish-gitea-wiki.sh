#!/usr/bin/env bash
# Sync docs/wiki markdown into the Gitea wiki git repository.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GITEA_URL="${REPOSITORY_DETECTIVE_GITEA_URL:-${BUGBOT_GITEA_URL:-https://git.commsnet.org}}"
OWNER="${REPOSITORY_DETECTIVE_GITEA_OWNER:-commstech}"
REPO="${REPOSITORY_DETECTIVE_GITEA_REPO:-Bugbot}"
TOKEN="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${BUGBOT_GITEA_TOKEN:-}}"
SOURCE_DIR="${WIKI_SOURCE_DIR:-$ROOT/docs/wiki}"
WORK_DIR="${WIKI_WORK_DIR:-$(mktemp -d)}"

log() { printf '==> %s\n' "$*"; }

if [ ! -d "$SOURCE_DIR" ]; then
  echo "wiki source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

if [ -z "$TOKEN" ]; then
  echo "REPOSITORY_DETECTIVE_GITEA_TOKEN (or BUGBOT_GITEA_TOKEN) is required" >&2
  exit 1
fi

WIKI_URL="${GITEA_URL%/}/${OWNER}/${REPO}.wiki.git"
AUTH_URL="${WIKI_URL/https:\/\//https://oauth2:${TOKEN}@}"

cleanup() {
  if [ "${KEEP_WIKI_WORKDIR:-false}" != "true" ]; then
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

if [ -d "$WORK_DIR/.git" ]; then
  log "fetch existing wiki clone"
  git -C "$WORK_DIR" fetch origin
  git -C "$WORK_DIR" checkout master 2>/dev/null || git -C "$WORK_DIR" checkout main
  git -C "$WORK_DIR" pull --ff-only
else
  log "clone wiki repo"
  git clone "$AUTH_URL" "$WORK_DIR"
fi

log "sync markdown from $SOURCE_DIR"
rsync -a --delete --include='*.md' --include='*/' --exclude='*' "$SOURCE_DIR/" "$WORK_DIR/"

if git -C "$WORK_DIR" status --porcelain | grep -q .; then
  git -C "$WORK_DIR" add -A
  git -C "$WORK_DIR" commit -m "Sync operator wiki docs from main repo"
  git -C "$WORK_DIR" push origin HEAD
  log "wiki published"
else
  log "wiki already up to date — no push"
fi
