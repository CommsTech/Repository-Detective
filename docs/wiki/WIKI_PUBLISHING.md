# Wiki publishing (Gitea)

Repository Detective docs live in the **`docs/`** directory of the main repository. Gitea wikis use a **separate git remote** (`*.wiki.git`).

## Wiki-ready copies

Markdown prepared for wiki import is under **`docs/wiki/`** (mirrors key operator pages). Update those files when changing the corresponding `docs/*.md` sources.

## Verify remote (do not force-push)

```bash
cd /path/to/repository-detective
git remote -v
# Look for a wiki remote, e.g.:
# wiki  https://git.commsnet.org/commstech/repository-detective.wiki.git (fetch)
```

If no wiki remote exists, the wiki has not been linked from this clone.

## Manual publish (recommended)

From a machine with credentials to `git.commsnet.org`:

```bash
# One-time clone of the empty wiki repo
git clone https://git.commsnet.org/commstech/repository-detective.wiki.git /tmp/repository-detective-wiki
cd /tmp/repository-detective-wiki

# Copy prepared pages (adjust paths to your checkout)
cp /path/to/Repository-Detective/docs/wiki/*.md .

git add -A
git status
git commit -m "Sync operator docs from main repo"
git push origin master   # or main — match your wiki default branch
```

### Initial wiki home page

Create `Home.md` in the wiki repo with:

```markdown
# Repository Detective wiki

Operator documentation synced from the main repository.

- [Privacy and data protection](PRIVACY_AND_DATA_PROTECTION)
- [Scanner health](SCANNER_HEALTH)
- [Dashboard guide](DASHBOARD_GUIDE)
- [Setup](SETUP) — copy from docs/SETUP.md when needed
```

Gitea wiki links use page names without `.md`.

## Automated sync (recommended script)

```bash
# Dry-run (lists pages, no credentials required for listing)
WIKI_DRY_RUN=true ./scripts/publish-gitea-wiki.sh

# Publish (token via env — not stored in git config)
export REPOSITORY_DETECTIVE_GITEA_TOKEN='…'   # wiki-write scope
./scripts/publish-gitea-wiki.sh
```

Options:

| Variable | Purpose |
|----------|---------|
| `WIKI_DRY_RUN=true` | List pages only |
| `WIKI_REMOTE_URL` | Override wiki git URL |
| `KEEP_WIKI_WORKDIR=true` | Inspect clone after run |

Never use `git push --force` on the wiki remote.

## Status in this environment

Wiki push requires credentials to `git.commsnet.org`. **Prepared copies** are under `docs/wiki/`; verify push locally with the commands above.

**Do not claim the wiki was updated unless `git push` to the wiki remote succeeded.**
