# Private beta operator runbook

For operators distributing and supporting the Repository Detective private beta.

## Release build steps

```bash
cd /path/to/repository-detective
git checkout main && git pull
make clean-beta-release
make beta-release
./scripts/check-beta-package-secrets.sh
```

Distribute `dist/repository-detective-beta/` via secure channel (not public git).

Optional SBOM:

```bash
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
make beta-release   # produces sbom-go.cdx.json when tool on PATH
```

## Release artifact verification

| Step | Command |
|------|---------|
| Checksum | `cd dist/repository-detective-beta && sha256sum -c checksums.txt` |
| Binary runs | `./repository-detective --help` |
| Secrets | `./scripts/check-beta-package-secrets.sh` |
| Config safety | Confirm `auto_create_issues: false` in `config.example.yaml` |
| Compose safety | Confirm `REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES: "false"` in compose |

See [PRIVATE_BETA_PACKAGE_VERIFICATION.md](PRIVATE_BETA_PACKAGE_VERIFICATION.md).

## Deployment options

| Option | Use when |
|--------|----------|
| Docker Compose (`docker-compose.beta.yml`) | Default for testers |
| Binary + systemd | Homelab without Docker |
| Existing `docker-compose.yml` + host network | Production homelab (operator-managed) |

Beta bundle ships `docker-compose.beta.yml` with bridge networking and localhost bind.

## Environment variables

Prefer `REPOSITORY_DETECTIVE_*` prefix. Legacy `REPOSITORY_DETECTIVE_*` still supported.

| Variable | Required | Notes |
|----------|----------|-------|
| `REPOSITORY_DETECTIVE_API_KEY` | Yes | UI/API auth |
| `REPOSITORY_DETECTIVE_GITEA_URL` | If using Gitea | Forge base URL |
| `REPOSITORY_DETECTIVE_GITEA_TOKEN` | If using Gitea | API token |
| `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` | If webhooks | HMAC secret |
| `REPOSITORY_DETECTIVE_PUBLIC_URL` | Recommended | Webhook + links |

Full list: `.env.example`

## Scanner installation checks

```bash
curl -s http://127.0.0.1:8081/health | jq '.tools_summary'
# or
REPOSITORY_DETECTIVE_API_KEY=... ./scripts/operator-smoke-test.sh
```

All-in-one Docker image includes external tools when built with `INSTALL_EXTERNAL_TOOLS=true`.

## SBOM checks

If `sbom-go.cdx.json` present in bundle, verify module list matches release tag. If absent, document as optional gap for private beta.

## Backup / restore

```bash
# Backup
cp data/repository-detective.db backups/repository-detective-$(date +%Y%m%d).db

# Restore (stop service first)
docker compose -f docker-compose.beta.yml down
cp backups/repository-detective-YYYYMMDD.db data/repository-detective.db
docker compose -f docker-compose.beta.yml up -d
```

## Upgrade path

1. Backup `data/repository-detective.db`
2. Pull new image or replace binary
3. Start service — migrations run automatically
4. Verify `/health` and `/api/v1/status`
5. Re-run operator smoke test

## Rollback path

1. Stop service
2. Restore previous binary/image tag and database backup
3. Start service
4. Confirm scan history intact

Keep at least one previous beta bundle checksum on file.

## Health checks

```bash
curl -s http://127.0.0.1:8081/health
# Expect: "status":"healthy", database_healthy:true

curl -s -H "X-Repository-Detective-API-Key: $KEY" http://127.0.0.1:8081/api/v1/status
```

Automated: `./scripts/operator-smoke-test.sh` with `RD_BASE_URL` set.

## Log locations

| Deployment | Logs |
|------------|------|
| Docker | `docker logs repository-detective-beta` |
| Binary | stdout/stderr or systemd journal |

## Troubleshooting

| Issue | Action |
|-------|--------|
| DB readonly | Fix `data/` ownership (`chown` to container user or run as matching UID) |
| Scanners unavailable | Rebuild all-in-one image; check `/health` tools_summary |
| Webhook 401 | Verify secret matches config |
| Scan stuck | Check `analysis_timeout`, concurrent scan limit |
| Learning UI empty | Rebuild image post-learning-engine commits |

## Known CI runner wrapper lag

Gitea Actions runners may lag main by several commits. Beta release uses `make beta-release` independently of CI green status. Track CI separately for public beta readiness.

## Known Gitea runner issues

- Runner may not have all external scanners — use all-in-one Docker for parity
- Long staticcheck runs may timeout on large monorepos — increase `staticcheck_timeout_seconds`

## Confirm issue filing is off

```bash
curl -s -H "X-Repository-Detective-API-Key: $KEY" http://127.0.0.1:8081/api/v1/status | jq '.auto_create_issues'
# Expect: false

# After report-only scan, confirm Gitea open issue count unchanged
```

Log line when skipped: `Forge issue creation skipped (policy_level=... issue_policy=off)`

## Confirm report-only mode

API request must include `"report_only_dry_run": true`. Scan record shows `dry_run_report_only: true`. `issue_sync_status: skipped` in scan metadata.

## Rotate API keys

1. Generate new `REPOSITORY_DETECTIVE_API_KEY`
2. Update `.env`, restart service
3. Update tester curl scripts / CI integrations
4. Invalidate old key (remove from env)

Forge tokens: rotate in Gitea → update `.env` → restart.

## Handle suspected secrets in package

1. Stop distribution immediately
2. Run `./scripts/check-beta-package-secrets.sh`
3. Grep bundle for patterns (see package verification doc)
4. Rebuild from clean tree; rotate any exposed credentials
5. Document incident in operator notes

## Enable limited issue filing safely (later)

Only after explicit operator approval:

1. Confirm product repo active-present = 0 on latest scan
2. Enable for **one** non-product repo via Configure → issue policy
3. Set conservative gates (high severity, high confidence)
4. Keep `dogfood_backlog_control_enabled: true`
5. Monitor open issue count for 24h before expanding

Never enable all-repo scan or global `auto_create_issues: true` during private beta without separate gate review.

## Related docs

- [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
- [PRIVATE_BETA_RELEASE_NOTES.md](PRIVATE_BETA_RELEASE_NOTES.md)
- [PRIVATE_BETA_RELEASE_GO_NO_GO.md](PRIVATE_BETA_RELEASE_GO_NO_GO.md)
