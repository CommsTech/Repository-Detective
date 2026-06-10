# Gitea wiki HTTP 500 triage packet

Recorded: 2026-06-10 (updated)

## Wiki remote

`https://git.commsnet.org/commstech/Bugbot.wiki.git`

## Repo wiki enabled?

```bash
curl -s -H "Authorization: token $BUGBOT_GITEA_TOKEN" \
  https://git.commsnet.org/api/v1/repos/commstech/Bugbot | jq '.has_wiki'
```

Result: **`true`** (admin/push/pull permissions present).

## Failing command (token redacted)

```bash
AUTH_URL="https://oauth2:***@git.commsnet.org/commstech/Bugbot.wiki.git"
git clone "$AUTH_URL" /tmp/wiki-test
# fatal: unable to access '...Bugbot.wiki.git/': The requested URL returned error: 500
```

## One-page push (2026-06-10)

| Step | Result |
|---|---|
| Clone existing wiki | **HTTP 500** |
| Init local repo + single `One-Page-Test.md` | commit ok |
| `git push origin HEAD` | **HTTP 500** |

## Full publish

**Not attempted** — blocked until one-page push succeeds.

## Dry-run publish script

```bash
WIKI_DRY_RUN=true ./scripts/publish-gitea-wiki.sh
```

Result: **success** — 23 pages listed; no git operations.

## Server log excerpt

Server-side Gitea logs not available from this host. Operator should inspect during push:

```bash
journalctl -u gitea --since '10 min ago'
# or tail /var/log/gitea/gitea.log
```

Look for: uninitialized `bugbot.wiki.git`, hook failure, storage permissions, DB error on wiki metadata.

## Root cause

**Server-side Gitea wiki git HTTP backend** — clone and push both return HTTP 500 before any content volume is reached. Not a token-scope or client content issue (API `has_wiki: true`, dry-run succeeds).

## Fix applied

None on client. **Operator next step:**

1. On Gitea host, verify `{data}/gitea-repositories/commstech/bugbot.wiki.git` exists and is writable.
2. Create one manual wiki page in Gitea UI (initializes wiki storage).
3. Retry one-page `git push`; then run `./scripts/publish-gitea-wiki.sh`.
4. If still 500, repair/recreate wiki bare repo per Gitea admin docs.

## Status

- Wiki populated: **no**
- Product dogfood: **not blocked** (0 active-present)
