# Gitea wiki HTTP 500 triage packet

Recorded: 2026-06-09

## Wiki remote

`https://git.commsnet.org/commstech/Bugbot.wiki.git`

## Failing command

```bash
./scripts/publish-gitea-wiki.sh
# or: git push origin HEAD
```

## Exact failure

```text
fatal: unable to access 'https://git.commsnet.org/commstech/Bugbot.wiki.git/': The requested URL returned error: 500
```

## Token scope assumptions

- Gitea personal access token with **read/write repository** and **wiki** permissions
- Token passed via `BUGBOT_GITEA_TOKEN` in `.env` (never commit)

## Repo wiki enabled?

Check via API:

```bash
curl -s -H "Authorization: token $BUGBOT_GITEA_TOKEN" \
  https://git.commsnet.org/api/v1/repos/commstech/Bugbot | jq '.has_wiki'
```

## Does `Bugbot.wiki.git` exist on server?

On Gitea host, inspect:

```text
{GITEA_DATA}/gitea-repositories/commstech/bugbot.wiki.git
```

If missing or corrupt, initialize wiki from UI (create one manual page) or repair bare repo permissions.

## Push failure modes

| Scenario | Observed |
|----------|----------|
| Empty wiki, first push (23 pages) | **HTTP 500** |
| Single-page push | **not attempted** (server fix required first) |
| Dry-run page list | **success** (23 pages) |

## Likely server logs to inspect

```bash
journalctl -u gitea --since '10 min ago'
# or tail /var/log/gitea/gitea.log during push
```

Look for: wiki repo creation failure, git hook error, storage permission, LFS mismatch, database error on wiki metadata.

## Rollback-safe repair steps

1. Confirm `has_wiki: true` on `commstech/Bugbot`.
2. Create one manual wiki page in Gitea UI (initializes wiki git storage).
3. Re-run dry-run: `KEEP_WIKI_WORKDIR=true ./scripts/publish-gitea-wiki.sh` (list only).
4. Attempt single-page push to `Bugbot.wiki.git` before full 23-page publish.
5. If still 500, repair/recreate `bugbot.wiki.git` on server; do not delete main repo.
6. Re-run full publish after server confirms wiki git backend healthy.

## Status

- Wiki populated: **no**
- Blocker: **server-side Gitea wiki git backend**
- Product active-present burn-down: **not blocked**

## Update 2026-06-02

| Check | Result |
|---|---|
| Current wiki status | Push still fails with HTTP 500 |
| Server logs checked | **no** (requires Gitea host operator access) |
| Wiki repo exists | **unknown** — API `has_wiki` check not re-run this sprint |
| Single-page push tested | **no** — deferred until server repair |
| Next operator action | Gitea admin: verify `bugbot.wiki.git` exists, create one manual wiki page in UI, inspect `journalctl -u gitea` during push, then retry single-page publish |

This sprint focused on active-present burn-down; wiki repair remains option **A** for the next batch.
