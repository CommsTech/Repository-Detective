# Gitea wiki publishing

Repository Detective keeps **source** wiki pages under `docs/wiki/`. The live operator wiki is:

```text
https://git.commsnet.org/commstech/repository-detective/wiki
```

Gitea’s HTML/wiki slug for this product is **lowercase** `repository-detective`. Clone URLs may still use `Repository-Detective` casing; both should resolve on this forge. Prefer the lowercase slug for wiki API and wiki git remotes.

## Source of truth

| Location | Role |
|----------|------|
| `docs/wiki/*.md` | Editable source copies in the main repo |
| Gitea wiki UI / API | Published operator wiki |

Edit pages in `docs/wiki/`, then publish.

## Publish (preferred): Gitea Wiki API

Git push to `*.wiki.git` may return **HTTP 500** on this forge. Use the API publisher:

```bash
export REPOSITORY_DETECTIVE_GITEA_URL=https://git.commsnet.org
export REPOSITORY_DETECTIVE_GITEA_OWNER=commstech
export REPOSITORY_DETECTIVE_GITEA_REPO=repository-detective
export REPOSITORY_DETECTIVE_GITEA_TOKEN=your-token-with-wiki-write

python3 scripts/publish-gitea-wiki-api.py
```

The script creates or updates each `docs/wiki/*.md` page via `POST /wiki/new` and `PATCH /wiki/page/{title}`.

## Publish (fallback): wiki git remote

```bash
export REPOSITORY_DETECTIVE_GITEA_REPO=repository-detective
export REPOSITORY_DETECTIVE_GITEA_TOKEN=your-token-with-wiki-write
./scripts/publish-gitea-wiki.sh
```

If clone/push fails with HTTP 500, use the API publisher above.

Environment variables:

| Variable | Default |
|----------|---------|
| `REPOSITORY_DETECTIVE_GITEA_URL` | `https://git.commsnet.org` |
| `REPOSITORY_DETECTIVE_GITEA_OWNER` | `commstech` |
| `REPOSITORY_DETECTIVE_GITEA_REPO` | `repository-detective` |
| `REPOSITORY_DETECTIVE_GITEA_TOKEN` | (required) |
| `WIKI_SOURCE_DIR` | `docs/wiki` |

# Publish to GitHub wiki

GitHub stores the wiki in a separate git remote (`Repository-Detective.wiki.git`).
That remote is **created only after** the first page is saved in the GitHub UI.

```bash
# One-time: open https://github.com/CommsTech/Repository-Detective/wiki
# → “Create the first page” → Save (Home)

export PATH="$HOME/.local/bin:$PATH"   # if using user-local gh
gh auth status                          # needs repo scope
./scripts/publish-github-wiki.sh
# or: ./scripts/publish-github-wiki.sh --wait
```

Auth: `gh auth login` (recommended) or `REPOSITORY_DETECTIVE_GITHUB_TOKEN` with `repo` scope.
Main-repo **deploy keys cannot** push the wiki remote.

In-repo copies stay at `docs/wiki/` (always browsable on GitHub without the wiki tab).

## Gitea wiki link format

Gitea wiki pages use names **without** `.md`:

```markdown
[Dashboard guide](DASHBOARD_GUIDE)
```

Curated `docs/wiki/Home.md` is the published Home page — do not overwrite it with auto-generated bullet lists.
