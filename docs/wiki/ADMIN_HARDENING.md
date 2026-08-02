# Administrator hardening

Privacy-aware deployment checklist for operators. **Not** a compliance certification.

## Network and access

- [ ] Expose UI/API only on trusted networks or behind reverse proxy + TLS
- [ ] Set strong `api_key` / `REPOSITORY_DETECTIVE_API_KEY`
- [ ] Restrict Gitea token to required repositories and scopes
- [ ] Configure `public_url` correctly for webhooks and callbacks

## Secrets

- [ ] Store tokens in `.env` or secret manager — never in `config.yaml` committed to git
- [ ] Rotate `gitea_token` and `webhook_secret` on compromise
- [ ] Verify `internal/security.MinimalSubprocessEnv` does not pass secrets to scanners (see `SubprocessEnvExposesSecrets` test)

## AI / LLM

- [ ] Set `enable_llm_auditors: false` until policy allows external inference
- [ ] Review [AI_PROVIDERS.md](AI_PROVIDERS.md) for data residency
- [ ] Disable `enable_ai_risk_checks` in regulated environments if not required

## Data

- [ ] Restrict SQLite file permissions (`chmod 600`)
- [ ] Schedule backups per [DATABASE.md](DATABASE.md) and [DATA_RETENTION.md](DATA_RETENTION.md)
- [ ] Define retention and deletion for findings and workspace clones

## Scanning

- [ ] Install only scanners enabled in config
- [ ] Review [SCANNER_HEALTH.md](SCANNER_HEALTH.md) for degraded coverage
- [ ] Use `reporting.mode` to limit Gitea issue noise

## Logging

- [ ] Run at `info` in production; avoid `debug` with sensitive repos
- [ ] Aggregate logs with access controls
- [ ] Scanner logs use redacted `detail` fields where updated (see closeout sprint)

## UI

- [ ] Require API key for `/ui/*` when exposed externally
- [ ] Review [ACCESSIBILITY.md](ACCESSIBILITY.md) if 508 procurement applies

## Incident response

- [ ] Revoke Gitea token if leaked via issue body or log
- [ ] Re-scan after secret rotation
- [ ] Purge workspace directories under scan work path if they may contain cloned secrets

See also [SECURITY_HARDENING.md](SECURITY_HARDENING.md), [PRIVACY_AND_DATA_PROTECTION.md](PRIVACY_AND_DATA_PROTECTION.md).

---

See also [Home](Home).
