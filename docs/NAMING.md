# Product naming

| Name | Usage |
|------|--------|
| **Repository Detective** | Product name — docs, UI, reports, issue bodies, binaries, compose, Gitea repo |
| **Tagline** | Inspect. Analyze. Improve. |

## Release branding

Public release surfaces say **Repository Detective** only.

| Surface | Value |
|---------|--------|
| Env vars | `REPOSITORY_DETECTIVE_*` |
| API key header | `X-Repository-Detective-API-Key` |
| Issue labels | `repository-detective/*` |
| Fingerprint body marker | `Repository Detective fingerprint:` |
| Fingerprint values | `rd-<hex>` |
| Go module | `git.commstech.org/commstech/repository-detective` (development deployment identifier; runtime is fully configurable via `gitea_url`/env) |
| Gitea repo (canonical) | `https://git.example.com/<your-org>/Repository-Detective.git` |
| GitHub mirror | `https://github.com/<your-org>/Repository-Detective.git` |
| Binary / container / image | `repository-detective` |
| SQLite path | `./data/repository-detective.db` |

## Cursor product comparisons

Docs that compare against **Cursor Bugbot** keep that external product name on purpose (`docs/beta/CURSOR_BUGBOT_*`).
