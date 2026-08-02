# Sync Gitea → GitHub

Repository Detective’s **canonical** git host is Gitea:

- Gitea: https://git.commsnet.org/commstech/repository-detective  
- GitHub mirror: https://github.com/CommsTech/Repository-Detective  

GitHub is a public mirror for discovery. Day-to-day development, Issues (beta), Actions CI, and the operator wiki stay on Gitea unless explicitly moved.

## One-shot publish

From a clean `main` with all publish-ready commits:

```bash
set -a && source .env && set +a
chmod +x scripts/sync-gitea-to-github.sh
./scripts/sync-gitea-to-github.sh --dry-run
./scripts/sync-gitea-to-github.sh
```

Required env (gitignored `.env`):

| Variable | Purpose |
|----------|---------|
| `REPOSITORY_DETECTIVE_GITEA_TOKEN` | Gitea push (wiki/API token with repo write) |
| `REPOSITORY_DETECTIVE_GITHUB_TOKEN` | GitHub PAT with `contents:write` on `CommsTech/Repository-Detective` |

The script never stores tokens in `git remote` URLs (avoids leaking secrets into `.git/config`).

## Manual equivalent

```bash
git push origin main
git remote add github https://github.com/CommsTech/Repository-Detective.git  # once
git push https://x-access-token:${REPOSITORY_DETECTIVE_GITHUB_TOKEN}@github.com/CommsTech/Repository-Detective.git HEAD:main
```

## Safety

- Do not push `.env`, `config/config.yaml`, or `data/*.db` (gitignored).
- Do not force-push `main` on either remote unless recovering a broken mirror intentionally.
- After the first GitHub push, set the GitHub repo description/website to point at the Gitea project and wiki if desired.
