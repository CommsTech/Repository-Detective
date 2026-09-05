# Upgrade guide

Use this guide when moving Repository Detective between versions on a homelab or dogfood deployment.

## Before upgrade

1. **Backup** — Follow [BACKUP_RESTORE.md](./BACKUP_RESTORE.md) and copy `repository-detective.db` plus config.
2. **Note schema version** — On startup, migrations apply sequentially (`schema_migrations` table).
3. **Review release notes** — Check for new required env vars, scanner binaries, or breaking API changes.

## Standard upgrade (Docker Compose)

```bash
cd /path/to/repository-detective
git pull   # or replace image tag
docker compose build repository-detective
docker compose up -d repository-detective
```

Watch logs for migration success:

```text
Local database enabled (driver=sqlite path=...)
```

## Binary hotfix (Alpine container)

When rebuilding the full image is slow:

```bash
# Build static binary (example: golang Alpine)
docker run --rm -v "$PWD":/src -w /src golang:1.23-alpine \
  sh -c 'CGO_ENABLED=0 go build -o repository-detective .'

docker cp repository-detective repository-detective:/app/repository-detective
docker restart repository-detective
```

**Caution:** `docker compose up --force-recreate` can replace the container and lose an unstored hotfix binary. Prefer image rebuild for production.

## Configuration migration

- New settings in `config/config.yaml` merge with defaults on boot.
- Per-repo overrides in SQLite survive upgrades.
- After upgrade, open **Repo settings** and confirm effective profile/scanners.

## Token rotation

| Secret | Action |
|--------|--------|
| Gitea token | Update `.env` / config, restart, test manual scan |
| GitHub token | Same; invalid token returns 401 on GitHub scans |
| Runner shared secret | Update server and Gitea Actions workflow together |
| API key | Update clients and UI bookmarks (`?api_key=`)

## Rollback

1. Stop the new container.
2. Restore previous `repository-detective.db` **only if** the new version migrated forward and you need to revert (avoid mixing old binary with newer schema).
3. Run the previous image tag or binary.
4. If migrations already advanced schema, prefer fixing forward rather than downgrading the DB file.

## Restart window

During `docker stop` / `docker restart` / image recreate, the UI and API may briefly return `database is closed` while SQLite is torn down. This is expected; wait for `/health` to report ready before probing dashboards or running bulk scans.

## Post-upgrade validation

- [ ] `GET /health` returns ready
- [ ] Dashboard summary loads
- [ ] One manual scan completes
- [ ] Suppression / false-positive actions work (calibration phase)
- [ ] PR status gate behaves as expected on a test repo
