# Gitea wiki publishing

Repository Detective keeps **source** wiki pages under `docs/wiki/`. The live Gitea wiki is a **separate git repository**:

```text
https://git.commsnet.org/commstech/repository-detective.wiki.git
```

## Source of truth

| Location | Role |
|----------|------|
| `docs/wiki/*.md` | Editable source copies in the main repo |
| `*.wiki.git` | Published operator wiki (Gitea UI) |

Edit pages in `docs/wiki/`, then publish with the script below.

## Publish (manual)

```bash
export REPOSITORY_DETECTIVE_GITEA_URL=https://git.commsnet.org
export REPOSITORY_DETECTIVE_GITEA_OWNER=commstech
export REPOSITORY_DETECTIVE_GITEA_REPO=repository-detective
export REPOSITORY_DETECTIVE_GITEA_TOKEN=your-token-with-wiki-write

./scripts/publish-gitea-wiki.sh
```

Environment variables:

| Variable | Default |
|----------|---------|
| `REPOSITORY_DETECTIVE_GITEA_URL` | `BUGBOT_GITEA_URL` or `https://git.commsnet.org` |
| `REPOSITORY_DETECTIVE_GITEA_OWNER` | `commstech` |
| `REPOSITORY_DETECTIVE_GITEA_REPO` | `Bugbot` |
| `REPOSITORY_DETECTIVE_GITEA_TOKEN` | `BUGBOT_GITEA_TOKEN` |
| `WIKI_SOURCE_DIR` | `docs/wiki` |
| `WIKI_WORK_DIR` | temp directory |
| `KEEP_WIKI_WORKDIR` | `false` — set `true` to keep clone for debugging |

## Safety

- Never commit secrets or `docs/dogfood-reports/` content to the wiki.
- Do not force-push the wiki remote.
- CI does **not** auto-publish on every commit — use manual dispatch or release tagging when trusted.

## Gitea wiki link format

Gitea wiki pages use names **without** `.md`:

```markdown
[Dashboard guide](DASHBOARD_GUIDE)
```

## Deprecated: docs-only wiki

If pages only exist under `docs/wiki/` in the main repo, the Gitea wiki UI will **not** show them until `publish-gitea-wiki.sh` runs successfully.
