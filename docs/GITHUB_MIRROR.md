# Sync Gitea → GitHub

Repository Detective’s **canonical** git host is Gitea; **GitHub** is the public discovery mirror.

The same policy applies to **container images**:

| Artifact | Canonical | Public mirror |
|----------|-----------|---------------|
| Git | https://git.commsnet.org/commstech/repository-detective | https://github.com/CommsTech/Repository-Detective |
| Container | `git.commsnet.org/commstech/repository-detective` | `ghcr.io/commstech/repository-detective` |

- Gitea: https://git.commsnet.org/commstech/repository-detective  
- GitHub: https://github.com/CommsTech/Repository-Detective  

**Policy**

| Host | Role | When to push |
|------|------|--------------|
| **Gitea** | Canonical — day-to-day commits, Actions, wiki, **container packages**, maintainer issues | Keep `main` / packages updated continuously |
| **GitHub** | Public community mirror (git + optional GHCR) + **public feedback Issues** | After each publish-ready `main` update via **history-preserving** `--github` |

Day-to-day development stays on Gitea. Public bug/feature reports should use [GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose). Security: [SECURITY.md](../SECURITY.md).

### Container publish order

1. Build/sanitize on an operator host or Gitea Actions  
2. `./scripts/publish-docker-image.sh --tag vX.Y.Z` → **Gitea packages**  
3. Optional: add `--mirror-ghcr` (or run the GitHub **Docker publish (GHCR mirror)** workflow)  

See [DOCKER.md](DOCKER.md).

## Everyday: keep Gitea updated

```bash
set -a && source .env && set +a
./scripts/sync-gitea-to-github.sh --dry-run   # confirms Gitea-only plan
./scripts/sync-gitea-to-github.sh             # push main → Gitea only
```

Or a normal `git push origin main` after committing.

## Publish / refresh the public GitHub mirror

**Prefer full history** so GitHub looks like a real repository (not a one-commit orphan):

```bash
set -a && source .env && set +a
./scripts/sync-gitea-to-github.sh --github        # Gitea + history-preserving GitHub push
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

## History policy (storefront trust)

GitHub is part of the public storefront. Buyers evaluating a security product expect a **conventional commit graph**, not a single orphan snapshot labeled “1 Commit.”

| Mode | Flag | When to use |
|------|------|-------------|
| **History-preserving** (default for public refresh) | `--github` / `--github-only` | Normal releases and discovery updates |
| **Orphan tree snapshot** (emergency only) | `--github-snapshot` | Only if push protection still blocks full history; document why |

### Emergency snapshot (`--github-snapshot`)

Force-pushes an orphan commit of the current tree. This damages trust (GitHub shows ~1 commit). Prefer rewriting/allowlisting the historical blob that blocked the first full mirror (legacy Stripe-shaped test fixture) and returning to `--github`.

Release tags (`v*`) must remain immutable once published — see [RELEASE_MIRROR.md](RELEASE_MIRROR.md).
