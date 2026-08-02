# Gitea wiki HTTP 500 triage packet

Recorded: 2026-06-10 (updated)

## Wiki remote

`https://git.commsnet.org/commstech/repository-detective.wiki.git`

## Repo wiki enabled?

```bash
curl -s -H "Authorization: token $REPOSITORY_DETECTIVE_GITEA_TOKEN" \
  https://git.commsnet.org/api/v1/repos/commstech/Repository-Detective | jq '.has_wiki'
```

Result: **`true`** (admin/push/pull permissions present).

## Failing command (token redacted)

```bash
AUTH_URL="https://oauth2:***@git.commsnet.org/commstech/repository-detective.wiki.git"
git clone "$AUTH_URL" /tmp/wiki-test
# fatal: unable to access '...repository-detective.wiki.git/': The requested URL returned error: 500
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

## RC redeploy pass (2026-06-10)

| Step | Result |
|------|--------|
| `has_wiki` API | **true** |
| `WIKI_DRY_RUN=true ./scripts/publish-gitea-wiki.sh` | **success** (23 pages listed) |
| One-page git push | **still HTTP 500** (not re-tested live; prior failure unchanged) |
| Full wiki publish | **blocked** |

## Burn-down sprint re-test (2026-06-11)

| Step | Result |
|------|--------|
| `git clone https://git.commsnet.org/commstech/repository-detective.wiki.git` | **HTTP 500** (unchanged) |
| Manual wiki page in UI | not attempted (operator) |
| Gitea server logs | not available from this host |

**Operator action:** inspect Gitea server logs during wiki git push — server-side failure, not Repository Detective application code.

## Status

**Blocked** — wiki content prepared locally; cannot populate remote wiki until Gitea HTTP 500 resolved.

## Server log excerpt

Server-side Gitea logs not available from this host. Operator should inspect during push:

```bash
journalctl -u gitea --since '10 min ago'
# or tail /var/log/gitea/gitea.log
```

Look for: uninitialized `repository-detective.wiki.git`, hook failure, storage permissions, DB error on wiki metadata.

## Root cause

**Server-side Gitea wiki git HTTP backend** — clone and push both return HTTP 500 before any content volume is reached. Not a token-scope or client content issue (API `has_wiki: true`, dry-run succeeds).

## Fix applied

None on client. **Operator next step:**

1. On Gitea host, verify `{data}/gitea-repositories/commstech/Repository-Detective.wiki.git` exists and is writable.
2. Create one manual wiki page in Gitea UI (initializes wiki storage).
3. Retry one-page `git push`; then run `./scripts/publish-gitea-wiki.sh`.
4. If still 500, repair/recreate wiki bare repo per Gitea admin docs.

## Summary

- Wiki populated: **no**
- Product dogfood: monitor `active_present_open` after RC redeploy
