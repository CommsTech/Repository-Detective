# Branding and legacy compatibility

**Product name:** Repository Detective — Inspect. Analyze. Improve.

## Product-facing (use Repository Detective)

- Browser titles (`ui/templates/layout.html`)
- Dashboard, fleet control, scan modals, reports
- Beta tester guides (except legacy notes below)
- Release package display name

## Legacy compatibility (keep)

| Item | Notes |
|------|-------|
| `BUGBOT_*` environment variables | Aliased by `REPOSITORY_DETECTIVE_*` via envcompat |
| `X-Bugbot-API-Key` header | Accepted alongside `X-Repository-Detective-API-Key` |
| `data/bugbot.db` default path | SQLite filename; document as legacy |
| Gitea repo path `commstech/Bugbot` | Git remote path unchanged |
| Go module `git.commsnet.org/commstech/bugbot` | Internal import path |

## Historical / comparison docs (keep Bugbot name)

- `docs/beta/CURSOR_BUGBOT_COMPARISON.md` — external product comparison
- `docs/beta/CURSOR_BUGBOT_BENCHMARK_RESULTS.md` — benchmark context

## Preferred new identifiers

- `REPOSITORY_DETECTIVE_*` env prefix
- `X-Repository-Detective-API-Key` header
- `repository-detective` binary / container name
