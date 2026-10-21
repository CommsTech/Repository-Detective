# Upgrade guide

Use this guide when moving Repository Detective between versions on a homelab or dogfood deployment.

## Before upgrade

1. **Backup** — Follow [BACKUP_RESTORE.md](./BACKUP_RESTORE.md) and copy `repository-detective.db` plus config.
2. **Note schema version** — On startup, migrations apply sequentially (`schema_migrations` table).
3. **Review release notes** — Check for new required env vars, scanner binaries, or breaking API changes.

## Automated upgrade acceptance (RD-033)

Disposable harness (does not mutate production):

```bash
./scripts/e2e-upgrade-from-beta3.sh
```

Flow:

1. Deploy exact `v0.1.0-beta.3` digest `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727`
2. Seed representative state against disposable Gitea 1.22.3
3. Snapshot the SQLite DB (immutable baseline copy retained)
4. Upgrade to `repository-detective:upgrade-candidate` (current tree overlaid via `Dockerfile.binary-overlay`)
5. Run migrations, restart, verify persistence, auth, Doctor, and post-upgrade activity

Classification until the candidate is a published release digest:

`UPGRADE_FROM_BETA3_TO_CURRENT_MAIN_INTEGRATION_PROVEN`

After publishing the next release, re-run with exact digests beta.3 → beta.4 for `PUBLISHED_RELEASE_UPGRADE_E2E_PROVEN`.

Artifacts land under `e2e/results/<run-id>/` with sanitized diagnostics on failure.

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
