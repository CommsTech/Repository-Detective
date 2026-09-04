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
| **GitHub** | Public community mirror (git + optional GHCR) + **public feedback Issues** | After each publish-ready `main` update (or batch with `--github`) |

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

## History note (public seed)

The first GitHub `main` was seeded as a **clean public snapshot** (single commit) because GitHub push protection blocked a full-history mirror on a Stripe-shaped string inside an old analyzer *test* fixture (`analyzers/hardcoded_secret_test.go`). That fixture is fixed on Gitea `main`; the historical blob remains in Gitea history.

To replace the GitHub seed with full Gitea history later:

1. Allow the false-positive at GitHub → Settings → Secret scanning alerts / the unblock link from the rejected push, **or**
2. Coordinate a history rewrite of that test string on Gitea (force-push) and then `./scripts/sync-gitea-to-github.sh --github-only` (may need `--force` once).

Until then, refresh GitHub with:

```bash
./scripts/sync-gitea-to-github.sh --github-snapshot
```

(rewrites GitHub `main` to match the current Gitea tree as a fresh snapshot — for tester-facing updates without full history).
