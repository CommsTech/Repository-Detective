#!/usr/bin/env bash
# Keep Gitea (canonical) up to date. Optionally mirror to GitHub when the
# release is ready for public discovery.
#
# Usage:
#   ./scripts/sync-gitea-to-github.sh              # push main → Gitea only
#   ./scripts/sync-gitea-to-github.sh --github     # push main → Gitea + GitHub
#   ./scripts/sync-gitea-to-github.sh --github-only # push main → GitHub only
#   ./scripts/sync-gitea-to-github.sh --dry-run    # show remotes / planned pushes
#
# GitHub auth (first match wins):
#   1) SSH deploy key ~/.ssh/repository-detective-github-deploy
#   2) REPOSITORY_DETECTIVE_GITHUB_TOKEN / GITHUB_TOKEN / GH_TOKEN (HTTPS PAT)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DRY_RUN=false
PUSH_GITEA=true
PUSH_GITHUB=false

usage() {
  cat <<'EOF'
Keep Gitea (canonical) up to date. Mirror to GitHub only for public releases.

Usage:
  ./scripts/sync-gitea-to-github.sh              # push main → Gitea only
  ./scripts/sync-gitea-to-github.sh --github     # push main → Gitea + GitHub
  ./scripts/sync-gitea-to-github.sh --github-only
  ./scripts/sync-gitea-to-github.sh --dry-run
EOF
  exit 0
}

for arg in "$@"; do
  case "$arg" in
    --dry-run|-n) DRY_RUN=true ;;
    --github|--with-github|--public) PUSH_GITHUB=true ;;
    --github-only) PUSH_GITEA=false; PUSH_GITHUB=true ;;
    --gitea-only) PUSH_GITEA=true; PUSH_GITHUB=false ;;
    -h|--help) usage ;;
    *) printf 'error: unknown arg %s (try --help)\n' "$arg" >&2; exit 1 ;;
  esac
done

GITEA_URL="${REPOSITORY_DETECTIVE_GITEA_PUSH_URL:-https://git.commsnet.org/commstech/Repository-Detective.git}"
GITHUB_HTTPS_URL="${REPOSITORY_DETECTIVE_GITHUB_PUSH_URL:-https://github.com/CommsTech/Repository-Detective.git}"
GITHUB_SSH_URL="${REPOSITORY_DETECTIVE_GITHUB_SSH_URL:-git@github.com-repository-detective:CommsTech/Repository-Detective.git}"
DEPLOY_KEY="${REPOSITORY_DETECTIVE_GITHUB_DEPLOY_KEY:-$HOME/.ssh/repository-detective-github-deploy}"
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

use_github_ssh=false
if [[ -f "$DEPLOY_KEY" ]]; then
  use_github_ssh=true
fi

# Keep origin as the human-readable Gitea URL (no embedded token in git config).
ensure_remote origin "$GITEA_URL"
if $use_github_ssh; then
  ensure_remote github "$GITHUB_SSH_URL"
else
  ensure_remote github "$GITHUB_HTTPS_URL"
fi

log "policy: Gitea=canonical (default push); GitHub=public mirror (explicit --github only)"
log "remotes:"
git remote -v
log "plan: gitea=$PUSH_GITEA github=$PUSH_GITHUB dry_run=$DRY_RUN"

if $DRY_RUN; then
  exit 0
fi

if $PUSH_GITEA; then
  [[ -n "$GITEA_TOKEN" ]] || die "REPOSITORY_DETECTIVE_GITEA_TOKEN required for Gitea push"
  gitea_push="$(auth_url "$GITEA_URL" "$GITEA_TOKEN" "oauth2")"
  log "push $BRANCH → Gitea (canonical)"
  git push "$gitea_push" "HEAD:refs/heads/$BRANCH"
fi

if $PUSH_GITHUB; then
  log "push $BRANCH → GitHub (public mirror)"
  if $use_github_ssh; then
    GIT_SSH_COMMAND="ssh -i $DEPLOY_KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new" \
      git push github "HEAD:refs/heads/$BRANCH"
  else
    [[ -n "$GITHUB_TOKEN" ]] || die "Add deploy key ~/.ssh/repository-detective-github-deploy or set REPOSITORY_DETECTIVE_GITHUB_TOKEN"
    github_push="$(auth_url "$GITHUB_HTTPS_URL" "$GITHUB_TOKEN" "x-access-token")"
    git push "$github_push" "HEAD:refs/heads/$BRANCH"
  fi
else
  log "skipping GitHub (not a public release push — re-run with --github when ready)"
fi

log "done"
log "Gitea:  https://git.commsnet.org/commstech/repository-detective"
if $PUSH_GITHUB; then
  log "GitHub: https://github.com/CommsTech/Repository-Detective"
fi
