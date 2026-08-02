#!/usr/bin/env bash
# Push main to Gitea (canonical) and mirror to GitHub.
# Usage:
#   ./scripts/sync-gitea-to-github.sh           # push both
#   ./scripts/sync-gitea-to-github.sh --dry-run # show remotes / planned pushes
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" || "${1:-}" == "-n" ]]; then
  DRY_RUN=true
fi

GITEA_URL="${REPOSITORY_DETECTIVE_GITEA_PUSH_URL:-https://git.commsnet.org/commstech/Repository-Detective.git}"
GITHUB_URL="${REPOSITORY_DETECTIVE_GITHUB_PUSH_URL:-https://github.com/CommsTech/Repository-Detective.git}"
BRANCH="${REPOSITORY_DETECTIVE_SYNC_BRANCH:-main}"

GITEA_TOKEN="${REPOSITORY_DETECTIVE_GITEA_TOKEN:-${BUGBOT_GITEA_TOKEN:-}}"
GITHUB_TOKEN="${REPOSITORY_DETECTIVE_GITHUB_TOKEN:-${GITHUB_TOKEN:-${GH_TOKEN:-}}}"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

if [[ -n "$(git status --porcelain)" ]]; then
  die "working tree is dirty — commit or stash before syncing"
fi

current="$(git branch --show-current)"
[[ "$current" == "$BRANCH" ]] || die "checkout $BRANCH first (on $current)"

auth_url() {
  local base="$1" token="$2" user="$3"
  if [[ -z "$token" ]]; then
    printf '%s\n' "$base"
    return
  fi
  # oauth2:token works for Gitea; x-access-token works for GitHub PATs
  printf 'https://%s:%s@%s\n' "$user" "$token" "${base#https://}"
}

ensure_remote() {
  local name="$1" url="$2"
  if git remote get-url "$name" >/dev/null 2>&1; then
    git remote set-url "$name" "$url"
  else
    git remote add "$name" "$url"
  fi
}

# Keep origin as the human-readable Gitea URL (no embedded token in git config).
ensure_remote origin "$GITEA_URL"
ensure_remote github "$GITHUB_URL"

log "remotes:"
git remote -v

if $DRY_RUN; then
  log "dry-run: would push $BRANCH to origin (Gitea) and github"
  exit 0
fi

[[ -n "$GITEA_TOKEN" ]] || die "REPOSITORY_DETECTIVE_GITEA_TOKEN required for Gitea push"
[[ -n "$GITHUB_TOKEN" ]] || die "REPOSITORY_DETECTIVE_GITHUB_TOKEN (or GITHUB_TOKEN) required for GitHub push"

gitea_push="$(auth_url "$GITEA_URL" "$GITEA_TOKEN" "oauth2")"
github_push="$(auth_url "$GITHUB_URL" "$GITHUB_TOKEN" "x-access-token")"

log "push $BRANCH → Gitea (canonical)"
git push "$gitea_push" "HEAD:refs/heads/$BRANCH"

log "push $BRANCH → GitHub (mirror)"
git push "$github_push" "HEAD:refs/heads/$BRANCH"

log "done"
log "Gitea:  https://git.commsnet.org/commstech/repository-detective"
log "GitHub: https://github.com/CommsTech/Repository-Detective"
