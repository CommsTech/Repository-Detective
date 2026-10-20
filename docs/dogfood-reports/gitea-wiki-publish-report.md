# Gitea wiki publish report

**Last updated:** 2026-06-09

## Source

- `docs/wiki/` (23 markdown pages)
- Publish script: `scripts/publish-gitea-wiki.sh`

## Wiki remote

`https://git.commsnet.org/commstech/repository-detective.wiki.git`

## Latest attempt (2026-06-09)

| Step | Result |
|------|--------|
| Dry-run | **success** — 23 pages listed |
| Clone existing wiki | failed (empty wiki — init fallback) |
| Local commit in temp workdir | **success** — 23 files |
| `git push origin HEAD` | **failed** — HTTP **500** |

### Exact failure

```text
Command: ./scripts/publish-gitea-wiki.sh
Remote:  https://git.commsnet.org/commstech/repository-detective.wiki.git/
Error:   fatal: unable to access '…repository-detective.wiki.git/': The requested URL returned error: 500
```

Workdir preserved with `KEEP_WIKI_WORKDIR=true` for operator inspection.

## Likely cause

Server-side Gitea wiki git backend issue (not client auth — push reaches server and returns 500). Prior sprint saw same failure.

## Operator next steps

1. Check Gitea server logs at push time for wiki repository creation/storage errors.
2. Confirm wiki enabled on `commstech/Repository-Detective` (`has_wiki: true` via API).
3. Token must include **wiki write** scope.
4. Retry from UI: create one manual wiki page, then re-run `./scripts/publish-gitea-wiki.sh`.
5. If repo wiki git storage is corrupt, repair or recreate `Repository-Detective.wiki` on server.

## Server-log checklist (2026-06-09)

| Check | Command / location |
|-------|-------------------|
| Gitea app log at push time | `journalctl -u gitea --since '5 min ago'` or `/var/log/gitea/gitea.log` |
| Wiki repo exists | `GET /api/v1/repos/commstech/Repository-Detective/wiki/page` |
| Wiki git bare repo on disk | `{GITEA_DATA}/gitea-repositories/commstech/Repository-Detective.wiki.git` |
| Permissions | token has `write:repository` + wiki enabled on repo |
| Remote URL | `https://git.commsnet.org/commstech/repository-detective.wiki.git` |
| Bad init | empty wiki → first push may require UI "Initialize Wiki" |
| Size/content | 23 md pages; unlikely size limit on homelab |
| TLS/proxy | 500 after auth suggests server handler error, not client |

## Re-run (2026-06-09 stabilization batch)

| Step | Result |
|------|--------|
| Dry-run | success (23 pages) |
| Push | **not attempted** (prior HTTP 500; server fix required first) |
| HTTP code captured | **500** (from prior attempt) |

**Wiki is not populated until `git push` to `repository-detective.wiki.git` succeeds.**
