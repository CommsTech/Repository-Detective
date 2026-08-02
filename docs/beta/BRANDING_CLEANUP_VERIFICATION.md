# Repository Detective branding cleanup verification

Date: 2026-06-08

## Product UI

- Layout title: `Repository Detective — Inspect. Analyze. Improve.`
- `/ui/preinstall`: no Bugbot product name in enabled state
- Configure page: Repository Detective naming

## Allowed legacy references

- `REPOSITORY_DETECTIVE_*` env vars (envcompat)
- `X-Repository-Detective-API-Key` header
- `data/bugbot.db` path
- `legacy_name: Bugbot` in `/api/v1/about` compatibility block
- Comparison docs (`CURSOR_BUGBOT_COMPARISON.md`)

## Intentional archive/historical

- Gitea repo path `commstech/repository-detective`
- Go module import path

No new product-facing Bugbot branding added in this sprint.
