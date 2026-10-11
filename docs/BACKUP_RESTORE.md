# Backup and restore

Repository Detective stores scan history, findings, suppressions, remediation plans, and operator settings in a local SQLite database by default.

## What to back up

| Asset | Default path | Notes |
|-------|----------------|-------|
| SQLite database | `./data/repository-detective.db` (or `database.path` in config) | Primary state: repos, scans, findings, suppressions, remediation, forge issue mappings |
| Configuration | `config/config.yaml`, `.env` | Not in git if gitignored; includes Gitea token, API keys, AI endpoints |

## SQLite backup (recommended)

Stop writes briefly for a consistent copy, or use SQLite online backup:

```bash
# Graceful: pause container or stop service
docker compose stop repository-detective

# File copy (simple homelab)
cp /path/to/data/repository-detective.db /backups/repository-detective-$(date +%F).db

# Online backup (no full stop)
sqlite3 /path/to/data/repository-detective.db ".backup '/backups/repository-detective-$(date +%F).db'"
```

Restart the service after copy.

## Restore procedure

1. Stop Repository Detective.
2. Replace `repository-detective.db` with the backup file.
3. Ensure file permissions allow the container user to read/write.
4. Start the service — migrations run forward automatically; they do not downgrade schema.
5. Verify: open `/ui` dashboard, confirm repository and finding counts match expectations.

## Config restore

1. Restore `config/config.yaml` and `.env` from secure backup.
2. Rotate tokens if the backup may have been exposed.
3. Reconcile `gitea_token`, `github_token`, `runner_shared_secret`, and `api_key` with current forge settings.

## Verification checklist

- [ ] Dashboard loads with expected repo count
- [ ] Recent scans visible
- [ ] Suppression rules present under repo settings
- [ ] Manual scan on one repo completes

---

## Validated backup/restore drill

**Drill date:** 2026-06-02  
**Image:** `repository-detective:latest` (homelab host, port 8081)  
**Artifacts:** `deployment-backups/drill-20260602-141809/` (counts and health snapshots only — no secrets)

### What was backed up

| Asset | Path | Notes |
|-------|------|-------|
| SQLite database | `data/repository-detective.db` (~30 MB) | All repos, scans, findings, remediation, closure |
| Configuration | `config/config.yaml` | Operator config (tokens live in host `.env`, not copied into docs) |
| TLS CAs | `certs/` | Optional; copied into restore test for parity |

### Procedure (executed)

```bash
# 1. Baseline
curl -s http://127.0.0.1:8081/health | jq .
curl -s -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" http://127.0.0.1:8081/api/v1/dashboard/summary | jq .

# 2. Stop
docker stop repository-detective

# 3. Backup
DRILL=deployment-backups/drill-$(date +%Y%m%d-%H%M%S)
mkdir -p "$DRILL"
cp -a data/repository-detective.db "$DRILL/repository-detective.db"
cp -a config/config.yaml "$DRILL/config.yaml"

# 4. Restore into isolated directory
RESTORE=restore-drill-test
rm -rf "$RESTORE"
mkdir -p "$RESTORE/data" "$RESTORE/config" "$RESTORE/certs"
cp -a "$DRILL/repository-detective.db" "$RESTORE/data/"
cp -a "$DRILL/config.yaml" "$RESTORE/config/"
cp -a certs/. "$RESTORE/certs/" 2>/dev/null || true

# 5. Start fresh container against restore tree (alternate port)
docker run -d --name rd-restore-drill --network host --env-file .env \
  -e REPOSITORY_DETECTIVE_PORT=18082 \
  -e REPOSITORY_DETECTIVE_DATABASE_PATH=/app/data/repository-detective.db \
  -e REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true \
  -v "$PWD/$RESTORE/data:/app/data" \
  -v "$PWD/$RESTORE/config:/app/config:ro" \
  -v "$PWD/$RESTORE/certs:/app/certs:ro" \
  repository-detective:latest

# 6. Validate
curl -s http://127.0.0.1:18082/health | jq .
curl -s -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" http://127.0.0.1:18082/api/v1/dashboard/summary | jq .

# 7. Post-restore manual scan
curl -s -X POST -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" -H "Content-Type: application/json" \
  http://127.0.0.1:18082/api/v1/analyze \
  -d '{"owner":"commstech","repository":"Repository-Detective","ref":"main","scan_profile":"fast"}'

# 8. Teardown drill container; restart production
docker rm -f rd-restore-drill
docker start repository-detective
```

### Validation checklist (results)

| Check | Pre-backup | Post-restore | Pass |
|-------|------------|--------------|------|
| Repositories | 39 | 39 | Yes |
| Scans | 125 | 125 → 126 after manual scan | Yes |
| Findings | 2,549 | 2,549 | Yes |
| Remediation plans | 26 | 26 | Yes |
| Patch attempts | 0 | 0 | Yes |
| Closure evidence | 0 | 0 | Yes |
| Runner jobs | 0 | 0 | Yes |
| Schema migration max | 13 | 13 | Yes |
| `/health` | healthy | healthy | Yes |
| Dashboard API | 200 | 200 | Yes |

### Known caveats

- **`.env` secrets** are not in the DB backup; restore host must supply the same `.env` or equivalent secrets.
- **Schema forward migration:** Restoring an older DB into a **newer** binary applies pending migrations on startup (tested at v13; v14+ will apply when image is upgraded).
- **File copy vs online backup:** Drill used `cp` while container was stopped for consistency; for zero-downtime, use `sqlite3 .backup`.
- **Concurrent API + DB:** Brief `database is closed` errors were seen when hitting the API during health transitions; restart cleared them.

### Rollback procedure

1. `docker stop repository-detective` (or drill container).
2. Replace `data/repository-detective.db` with the known-good file from `deployment-backups/drill-*/repository-detective.db`.
3. Restore `config/config.yaml` from the same drill folder if needed.
4. `docker start repository-detective` and verify `/health` + dashboard counts.
5. If migrations were applied on a bad restore, roll back DB file from backup **before** migrations ran (never downgrade schema manually).
