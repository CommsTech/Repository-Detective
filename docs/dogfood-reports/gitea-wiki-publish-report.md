# Gitea wiki publish report

**Last updated:** 2026-06-09

## Source

- `docs/wiki/` (23 markdown pages)
- Publish script: `scripts/publish-gitea-wiki.sh`

## Wiki remote

`https://git.commsnet.org/commstech/Bugbot.wiki.git`

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
Remote:  https://git.commsnet.org/commstech/Bugbot.wiki.git/
Error:   fatal: unable to access '…Bugbot.wiki.git/': The requested URL returned error: 500
```

Workdir preserved with `KEEP_WIKI_WORKDIR=true` for operator inspection.

## Likely cause

Server-side Gitea wiki git backend issue (not client auth — push reaches server and returns 500). Prior sprint saw same failure.

## Operator next steps

1. Check Gitea server logs at push time for wiki repository creation/storage errors.
2. Confirm wiki enabled on `commstech/Bugbot` (`has_wiki: true` via API).
3. Token must include **wiki write** scope.
4. Retry from UI: create one manual wiki page, then re-run `./scripts/publish-gitea-wiki.sh`.
5. If repo wiki git storage is corrupt, repair or recreate `Bugbot.wiki` on server.

**Wiki is not populated until `git push` to `Bugbot.wiki.git` succeeds.**
