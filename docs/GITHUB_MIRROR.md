# Sync Gitea → GitHub

Repository Detective’s **canonical** git host is Gitea; **GitHub** is the public discovery mirror.

- Gitea: https://git.commsnet.org/commstech/repository-detective  
- GitHub: https://github.com/CommsTech/Repository-Detective  

**Policy**

| Host | Role | When to push |
|------|------|--------------|
| **Gitea** | Canonical — day-to-day commits, Issues, Actions, wiki | Keep `main` updated continuously |
| **GitHub** | Public community mirror | After each publish-ready `main` update (or batch with `--github`) |

Day-to-day development stays on Gitea. GitHub is for testers and discovery — not a second active development remote.

## Everyday: keep Gitea updated

```bash
set -a && source .env && set +a
./scripts/sync-gitea-to-github.sh --dry-run   # confirms Gitea-only plan
./scripts/sync-gitea-to-github.sh             # push main → Gitea only
```

Or a normal `git push origin main` after committing.

## Publish / refresh the public GitHub mirror

```bash
set -a && source .env && set +a
./scripts/sync-gitea-to-github.sh --github        # Gitea + GitHub
# or, if Gitea is already current:
./scripts/sync-gitea-to-github.sh --github-only
```

Required credentials:

| Credential | Purpose |
|------------|---------|
| `REPOSITORY_DETECTIVE_GITEA_TOKEN` | Gitea push |
| **GitHub deploy key** (preferred) | `~/.ssh/repository-detective-github-deploy` with **write** on the mirror repo |
| `REPOSITORY_DETECTIVE_GITHUB_TOKEN` | Fallback HTTPS PAT if no deploy key |

### Deploy key

Public key path: `~/.ssh/repository-detective-github-deploy.pub`  
Add under GitHub → **Settings** → **Deploy keys** → enable **Allow write access**.

SSH host alias: `github.com-repository-detective` (see `~/.ssh/config`).

The script never stores tokens in `git remote` URLs.

## Safety

- Do not push `.env`, `config/config.yaml`, or `data/*.db` (gitignored).
- Do not force-push `main` on either remote unless recovering a broken mirror intentionally.
- After publish, keep the GitHub repo description/website pointing at the Gitea project and wiki.
