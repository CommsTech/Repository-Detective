# Troubleshooting

## Health check fails

**Wrong port?**

| Compose file | curl |
|--------------|------|
| `docker-compose.public.yml` / `offline` | `http://127.0.0.1:8081/health` |
| `docker-compose.minimal.yml` | `http://127.0.0.1:8080/health` |

```bash
docker ps | grep bugbot
ss -tlnp | grep -E '8080|8081'
docker logs gitea-bugbot --tail 50
```

**Config errors** (in `docker logs`):

| Message | Fix |
|---------|-----|
| `gitea_url is required` | `BUGBOT_GITEA_URL` in `.env` |
| `gitea_token is required` | `BUGBOT_GITEA_TOKEN` |
| `configure ai_provider` | `BUGBOT_AI_PROVIDER` + `BUGBOT_AI_BASE_URL` if needed |
| Connection timeout at startup | `BUGBOT_SKIP_STARTUP_CHECKS=true` |

Run in foreground:

```bash
docker compose -f docker-compose.public.yml up --build
```

**503 on /health:** Normal for a few seconds while components initialize.

---

## Gitea webhooks fail

Gitea cannot reach private IPs. Bugbot needs a public URL — [NETWORKING.md](NETWORKING.md).

1. Confirm external access: `curl https://bugbot.example.com/health`
2. Set `BUGBOT_PUBLIC_URL` in `.env`, restart container
3. Webhook URL: `{PUBLIC_URL}/webhook`
4. Secret in Gitea must match `BUGBOT_WEBHOOK_SECRET`
5. Gitea signs each delivery with **HMAC-SHA256** of the raw JSON body and sends the hex digest in the `X-Gitea-Signature` header (Bugbot verifies this automatically)
6. Test delivery in Gitea webhook settings (expect HTTP 200)

**401 Unauthorized on webhook test:** Secret mismatch, or missing signature header. Confirm the secret matches on both sides and that Gitea is sending `X-Gitea-Signature`.

---

## `cannot unmarshal array into RepositoryContent`

Fixed in commit `c0580f6`. Pull latest and rebuild or reload image.

---

## AI provider TLS / certificate errors

Log example:

```text
AI provider connection check failed: tls: failed to verify certificate: x509: certificate signed by unknown authority
```

For **trusted homelab** endpoints with a private CA (e.g. OpenClaw on HTTPS):

```yaml
ai_insecure_skip_tls_verify: true
```

Or `BUGBOT_AI_INSECURE_SKIP_TLS_VERIFY=true` in `.env`. Prefer installing the CA on the host instead when possible.

---

## Too many false-positive Gitea issues

Static heuristics can flag safe patterns (shell env vars, `data-api-key` in templates, SQL fragments with `?` placeholders). See [FALSE_POSITIVES.md](FALSE_POSITIVES.md).

Quick mitigations:

- Raise `min_issue_confidence` (e.g. `0.7`)
- Lower `max_issues_per_run`
- Rebuild after analyzer updates and re-run `./deploy.sh --scan-all`
- Close stale tickets #33-style with a comment referencing the fix commit

---

## No issues after push

- `BUGBOT_AUTO_CREATE_ISSUES=true`
- Token needs issue write permission
- Check `docker logs gitea-bugbot`
- Repo may match `repository_exclude_patterns` in `config/config.yaml`

---

## Wizard API returns 401

`X-Bugbot-API-Key` must match `BUGBOT_API_KEY` in `.env`.

---

## Labels not attached to issues

Bugbot uses `POST /issues/{index}/labels` with body `{"labels":["security",1080]}`. If Gitea returns `[]`, Bugbot re-fetches labels via GET to verify attachment.

Check token issue-write permission. Missing label names are auto-created on the repository.

---

## Scanner tools missing in logs

```
[SCANNER:trivy] binary not found
[SCANNER:grype] binary not found
```

Rebuild using the current `Dockerfile` (installs Trivy, Grype, linters) or install binaries on `PATH` manually. See [SCANNERS.md](SCANNERS.md).

Verify inside the container:

```bash
docker exec gitea-bugbot sh -c 'command -v trivy && command -v grype && command -v golangci-lint'
```

---

## Scans run but no LLM output

Expected when `BUGBOT_ENABLE_LLM_AUDITORS=false` or when deterministic scanners found no flagged files. Check `[CAH:SCAN] External scanners found N candidate(s)` in logs.

---

## Cannot build image on target host

Build elsewhere, transfer image:

```bash
docker build -t gitea-bugbot:latest .
docker save gitea-bugbot:latest -o gitea-bugbot-image.tar
# copy to target
docker load -i gitea-bugbot-image.tar
docker compose -f docker-compose.offline.yml up -d
```
