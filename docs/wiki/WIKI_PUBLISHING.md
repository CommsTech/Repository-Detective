# Wiki publishing (Gitea)

Repository Detective docs live in the **`docs/`** directory of the main repository. Gitea wikis use a **separate git remote** (`*.wiki.git`).

## Wiki-ready copies

Markdown prepared for wiki import is under **`docs/wiki/`** (mirrors key operator pages). Update those files when changing the corresponding `docs/*.md` sources.

## Verify remote (do not force-push)

```bash
cd /path/to/Bugbot
git remote -v
# Look for a wiki remote, e.g.:
# wiki  https://git.commsnet.org/commstech/Bugbot.wiki.git (fetch)
```

If no wiki remote exists, the wiki has not been linked from this clone.

## Manual publish (recommended)

From a machine with credentials to `git.commsnet.org`:

```bash
# One-time clone of the empty wiki repo
git clone https://git.commsnet.org/commstech/Bugbot.wiki.git /tmp/Bugbot-wiki
cd /tmp/Bugbot-wiki

# Copy prepared pages (adjust paths to your checkout)
cp /path/to/Bugbot/docs/wiki/*.md .

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

## Automated sync (optional)

No CI job pushes the wiki by default. To automate:

1. Add a protected deploy key or token with wiki write access
2. Run the copy + commit steps in `.gitea/workflows/` only on release tags
3. Never use `git push --force` on the wiki remote

## Status in this environment

Wiki push requires credentials to `git.commsnet.org`. **Prepared copies** are under `docs/wiki/`; verify push locally with the commands above.

**Do not claim the wiki was updated unless `git push` to the wiki remote succeeded.**
