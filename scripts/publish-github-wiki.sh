#!/usr/bin/env bash
# Publish docs/wiki/*.md to https://github.com/CommsTech/Repository-Detective/wiki
#
# Prerequisites:
#   1) Wiki enabled on the repo (Settings → Features → Wikis)
#   2) Create the first page once in the GitHub UI (initializes *.wiki.git)
#   3) Auth: `gh auth login` (repo scope) OR REPOSITORY_DETECTIVE_GITHUB_TOKEN
#
# Usage:
#   ./scripts/publish-github-wiki.sh
#   ./scripts/publish-github-wiki.sh --wait   # poll until wiki remote exists, then publish
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SOURCE_DIR="${WIKI_SOURCE_DIR:-$ROOT/docs/wiki}"
OWNER="${REPOSITORY_DETECTIVE_GITHUB_OWNER:-CommsTech}"
REPO="${REPOSITORY_DETECTIVE_GITHUB_REPO:-Repository-Detective}"
WAIT=false
DRY_RUN=false

for arg in "$@"; do
  case "$arg" in
    --wait) WAIT=true ;;
    --dry-run|-n) DRY_RUN=true ;;
    -h|--help)
      sed -n '2,14p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) printf 'error: unknown arg %s\n' "$arg" >&2; exit 1 ;;
  esac
done

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

resolve_token() {
  if [[ -n "${REPOSITORY_DETECTIVE_GITHUB_TOKEN:-}" ]]; then
    printf '%s' "$REPOSITORY_DETECTIVE_GITHUB_TOKEN"
    return
  fi
  if [[ -n "${GH_TOKEN:-}" ]]; then
    printf '%s' "$GH_TOKEN"
    return
  fi
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    printf '%s' "$GITHUB_TOKEN"
    return
  fi
  if command -v gh >/dev/null 2>&1; then
    gh auth token 2>/dev/null && return
  fi
  if [[ -x "${HOME}/.local/bin/gh" ]]; then
    "${HOME}/.local/bin/gh" auth token 2>/dev/null && return
  fi
  return 1
}

TOKEN="$(resolve_token || true)"
[[ -n "$TOKEN" ]] || die "GitHub auth required (gh auth login or REPOSITORY_DETECTIVE_GITHUB_TOKEN)"
[[ -d "$SOURCE_DIR" ]] || die "missing $SOURCE_DIR"

AUTH_URL="https://x-access-token:${TOKEN}@github.com/${OWNER}/${REPO}.wiki.git"
PUBLIC_URL="https://github.com/${OWNER}/${REPO}/wiki"

wiki_remote_ready() {
  git ls-remote "$AUTH_URL" >/dev/null 2>&1
}

if $DRY_RUN; then
  log "dry-run: would publish $(find "$SOURCE_DIR" -maxdepth 1 -name '*.md' | wc -l) pages to $PUBLIC_URL"
  exit 0
fi

if ! wiki_remote_ready; then
  if $WAIT; then
    log "waiting for wiki remote (create the first page in the GitHub UI once): $PUBLIC_URL"
    for _ in $(seq 1 180); do
      if wiki_remote_ready; then
        log "wiki remote is ready"
        break
      fi
      sleep 5
    done
  fi
fi

if ! wiki_remote_ready; then
  die "GitHub wiki git remote does not exist yet.

GitHub only creates ${OWNER}/${REPO}.wiki.git after you create the first page in the UI:

  1) Open ${PUBLIC_URL}
  2) Click “Create the first page”
  3) Save (title Home is fine)
  4) Re-run: ./scripts/publish-github-wiki.sh

Or run: ./scripts/publish-github-wiki.sh --wait   (then do steps 1–3)
"
fi

WORK="$(mktemp -d)"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT

log "clone wiki"
git clone --depth 1 "$AUTH_URL" "$WORK/wiki"

log "sync $SOURCE_DIR → wiki"
rsync -a --delete \
  --include='*.md' --include='_Sidebar.md' --include='_Footer.md' --include='*/' --exclude='*' \
  "$SOURCE_DIR/" "$WORK/wiki/"

cat >"$WORK/wiki/_Footer.md" <<'EOF'
Source: `docs/wiki/` in the main repository · [Public beta](https://github.com/CommsTech/Repository-Detective/blob/main/docs/PUBLIC_BETA.md) · AGPL-3.0-or-later
EOF

if [[ ! -f "$WORK/wiki/_Sidebar.md" ]]; then
  {
    echo '**Repository Detective**'
    echo
    echo '[Home](Home)'
    echo
    for f in "$SOURCE_DIR"/*.md; do
      base="$(basename "$f" .md)"
      [[ "$base" == "Home" ]] && continue
      echo "- [${base//_/ }](${base})"
    done
  } >"$WORK/wiki/_Sidebar.md"
fi

git -C "$WORK/wiki" add -A
if git -C "$WORK/wiki" status --porcelain | grep -q .; then
  git -C "$WORK/wiki" \
    -c user.email="release@commsnet.org" \
    -c user.name="Repository Detective Release" \
    commit -m "Sync public wiki from docs/wiki"
  git -C "$WORK/wiki" push origin HEAD
  log "published → $PUBLIC_URL"
else
  log "wiki already up to date"
fi
