# Product naming

| Name | Usage |
|------|--------|
| **Repository Detective** | Product name — docs, UI, reports, issue bodies, binaries, compose, Gitea repo |
| **Tagline** | Inspect. Analyze. Improve. |

## Release branding

Public release surfaces must say **Repository Detective** only. Do not document or display the old internal project name.

| Surface | Public release |
|---------|----------------|
| Env vars (docs/examples) | `REPOSITORY_DETECTIVE_*` only |
| API key header (docs/UI) | `X-Repository-Detective-API-Key` |
| Issue labels (new writes) | `repository-detective/*` (`label_compat_mode: new_only`) |
| Fingerprint body marker | `Repository Detective fingerprint:` |
| Go module / Gitea repo | `repository-detective` |
| Binary / container / image | `repository-detective` |

## Silent compatibility (not advertised)

Existing deployments may still send legacy env prefixes or headers. Runtime continues to accept them via `internal/config/envcompat` and dual API-key header parsing. Fingerprint **values** keep their historical prefix so dedup is not broken. Label **lookup** still finds older issues. These are compatibility shims — not product branding.

See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md) for operator migration details.

## Cursor product comparisons

Docs that compare against **Cursor Bugbot** keep that external product name on purpose (`docs/beta/CURSOR_BUGBOT_*`).
