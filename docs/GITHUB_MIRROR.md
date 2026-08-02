# Sync Gitea → GitHub

Repository Detective’s **canonical** git host is Gitea:

- Gitea: https://git.commsnet.org/commstech/repository-detective  
- GitHub mirror: https://github.com/CommsTech/Repository-Detective  

**Policy**

| Host | Role | When to push |
|------|------|--------------|
| **Gitea** | Canonical — day-to-day commits, Issues, Actions, wiki | Keep `main` updated continuously |
| **GitHub** | Public discovery mirror | Only when a release is ready for public visibility (`--github`) |

Do not treat GitHub as a second active development remote until you intentionally open that release.

## Everyday: keep Gitea updated

```bash
set -a && source .env && set +a
./scripts/sync-gitea-to-github.sh --dry-run   # confirms Gitea-only plan
./scripts/sync-gitea-to-github.sh             # push main → Gitea only
```

Or a normal `git push origin main` after committing.

## Public release: mirror to GitHub

When the product is ready for a public GitHub presence:

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

### Deploy key (already generated on the operator host)

Public key path: `~/.ssh/repository-detective-github-deploy.pub`  
Add under GitHub → **Settings** → **Deploy keys** → enable **Allow write access**.

SSH host alias: `github.com-repository-detective` (see `~/.ssh/config`).

The script never stores tokens in `git remote` URLs.

## Safety

- Do not push `.env`, `config/config.yaml`, or `data/*.db` (gitignored).
- Do not force-push `main` on either remote unless recovering a broken mirror intentionally.
- After the first public GitHub push, set the GitHub repo description/website to point at the Gitea project and wiki.
