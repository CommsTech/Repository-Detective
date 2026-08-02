# Repo bloat audit — 2026-06-06

## Workspace size

| Path | Size / note |
|------|-------------|
| Total workspace | ~848M (mostly `vendor/` when present, local DBs) |
| `data/repository-detective.db` | Local runtime DB — **gitignored** |
| `deployment-backups/` | Operator backups — **gitignored** |
| `restore-drill-test/` | Drill artifacts — **gitignored** |
| `deploy/bin/trivy` | ~152M staged binary — **gitignored** |
| `repository-detective` | Local ELF build output — **now gitignored** |

## Large files (>5M, non-.git)

- `vendor/modernc.org/sqlite/lib/*` — vendored; directory gitignored
- No large dogfood JSON dumps committed

## Fixes applied

1. Added `/repository-detective` and `repository-detective` to `.gitignore`
2. Confirmed `*.db`, `data/`, `deployment-backups/`, `docs/dogfood-reports/*` ignore rules
3. Force-add only sanitized summary dogfood reports (see `docs/dogfood-reports/README.md`)

## Retained intentionally

- Summary dogfood markdown (CI gate, batch queues, lifecycle)
- `deploy/bin/README.md` (Docker build context)
- Small JSON run summaries (e.g. `lifecycle-burndown-run.json`)

## Not in repo (keep local)

- Raw scanner exports, full issue CSV dumps, unsanitized operator reports
- Local binaries and SQLite databases

## Before scanning all Gitea repos

- Do not commit per-scan raw exports
- Use wiki or external storage for large historical dumps
- Cap dogfood report growth via README retention policy
