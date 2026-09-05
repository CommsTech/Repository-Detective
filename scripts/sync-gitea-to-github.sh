#!/usr/bin/env bash
# Keep Gitea (canonical) up to date. Optionally refresh the public GitHub mirror.
#
# Usage:
#   ./scripts/sync-gitea-to-github.sh                 # push main → Gitea only
#   ./scripts/sync-gitea-to-github.sh --github         # Gitea + full-history GitHub push
#   ./scripts/sync-gitea-to-github.sh --github-only    # full-history GitHub push only
#   ./scripts/sync-gitea-to-github.sh --github-snapshot  # Gitea + tree snapshot → GitHub
#   ./scripts/sync-gitea-to-github.sh --dry-run
#
# Prefer --github-snapshot while GitHub push protection blocks full history
# (historical Stripe-shaped test fixture). After allowlisting or history rewrite,
# use --github for a normal fast-forward mirror.
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
GITHUB_SNAPSHOT=false

usage() {
  cat <<'EOF'
Keep Gitea (canonical) up to date. Refresh GitHub for public testers.

Usage:
  ./scripts/sync-gitea-to-github.sh                    # push main → Gitea only
  ./scripts/sync-gitea-to-github.sh --github-snapshot  # Gitea + clean tree snapshot → GitHub
  ./scripts/sync-gitea-to-github.sh --github           # Gitea + full-history GitHub (after allowlist)
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
    --github-snapshot) PUSH_GITHUB=true; GITHUB_SNAPSHOT=true ;;
    --gitea-only) PUSH_GITEA=true; PUSH_GITHUB=false; GITHUB_SNAPSHOT=false ;;
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

# Refuse to publish if live secrets/configs leaked into the git tree.
"$ROOT/scripts/check-public-release-secrets.sh" || die "public secret gate failed — fix before sync"

current="$(git branch --show-current)"
[[ "$current" == "$BRANCH" ]] || die "checkout $BRANCH first (on $current)"

auth_url() {
  local base="$1" token="$2" user="$3"
  if [[ -z "$token" ]]; then
    printf '%s\n' "$base"
    return
  fi
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

push_github_snapshot() {
  local tmp sha msg
  tmp="$(mktemp -d)"
  sha="$(git rev-parse --short HEAD)"
  msg="Public mirror snapshot of Gitea ${BRANCH}@${sha}

Canonical history: https://git.commsnet.org/commstech/repository-detective
License: AGPL-3.0-or-later"
  log "building clean GitHub tree snapshot from $BRANCH@$sha"
  git clone --no-local --quiet "$ROOT" "$tmp/rd"
  (
    cd "$tmp/rd"
    git checkout --orphan "public-snapshot"
    git rm -rf . >/dev/null 2>&1 || true
    git checkout "$BRANCH" -- .
    git add -A
    git -c user.email="release@commsnet.org" -c user.name="Repository Detective Release" \
      commit -m "$msg" >/dev/null
    if [[ -f "$DEPLOY_KEY" ]]; then
      GIT_SSH_COMMAND="ssh -i $DEPLOY_KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new" \
        git push --force git@github.com-repository-detective:CommsTech/Repository-Detective.git HEAD:main
    else
      [[ -n "$GITHUB_TOKEN" ]] || die "deploy key or GITHUB token required for snapshot push"
      git push --force "$(auth_url "$GITHUB_HTTPS_URL" "$GITHUB_TOKEN" "x-access-token")" HEAD:main
    fi
  )
  rm -rf "$tmp"
}

use_github_ssh=false
if [[ -f "$DEPLOY_KEY" ]]; then
  use_github_ssh=true
fi

ensure_remote origin "$GITEA_URL"
if $use_github_ssh; then
  ensure_remote github "$GITHUB_SSH_URL"
else
  ensure_remote github "$GITHUB_HTTPS_URL"
fi

log "policy: Gitea=canonical; GitHub=public mirror"
log "remotes:"
git remote -v
log "plan: gitea=$PUSH_GITEA github=$PUSH_GITHUB snapshot=$GITHUB_SNAPSHOT dry_run=$DRY_RUN"

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
  if $GITHUB_SNAPSHOT; then
    log "push tree snapshot → GitHub (public mirror)"
    push_github_snapshot
  else
    log "push $BRANCH → GitHub (full history)"
    if $use_github_ssh; then
      GIT_SSH_COMMAND="ssh -i $DEPLOY_KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new" \
        git push github "HEAD:refs/heads/$BRANCH"
    else
      [[ -n "$GITHUB_TOKEN" ]] || die "Add deploy key or set REPOSITORY_DETECTIVE_GITHUB_TOKEN"
      github_push="$(auth_url "$GITHUB_HTTPS_URL" "$GITHUB_TOKEN" "x-access-token")"
      git push "$github_push" "HEAD:refs/heads/$BRANCH"
    fi
  fi
else
  log "skipping GitHub — use --github-snapshot (testers) or --github (full history)"
fi

log "done"
log "Gitea:  https://git.commsnet.org/commstech/repository-detective"
if $PUSH_GITHUB; then
  log "GitHub: https://github.com/CommsTech/Repository-Detective"
fi
